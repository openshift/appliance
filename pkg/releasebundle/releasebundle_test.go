package releasebundle

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/openshift/appliance/pkg/executer"
)

const (
	testPort    = 5005
	testVersion = "4.20.5-x86_64"
)

func chdir(t *testing.T, dir string) {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err = os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})
}

// useBundleDockerfile runs the test from a temp dir containing bundle/Dockerfile.bundle.
func useBundleDockerfile(t *testing.T) {
	t.Helper()
	wd := t.TempDir()
	writeFile(t, filepath.Join(wd, "bundle", "Dockerfile.bundle"), fallbackDockerfile)
	chdir(t, wd)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// listContext returns the manifest files (relative paths) staged in the build context
// of a podman build command.
func listContext(t *testing.T, buildCmd string) []string {
	t.Helper()
	fields := strings.Fields(buildCmd)
	contextDir := fields[len(fields)-1]
	var files []string
	err := filepath.WalkDir(filepath.Join(contextDir, ManifestsDir), func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(contextDir, path)
		files = append(files, rel)
		return nil
	})
	if err != nil {
		t.Fatalf("walk context: %v", err)
	}
	sort.Strings(files)
	return files
}

func TestBundlePush(t *testing.T) {
	src := t.TempDir()
	imageSet := filepath.Join(src, "imageset.yaml")
	resources := filepath.Join(src, "cluster-resources")
	writeFile(t, imageSet, "kind: ImageSetConfiguration\n")
	writeFile(t, filepath.Join(resources, "idms-oc-mirror.yaml"), "kind: ImageDigestMirrorSet\n")
	writeFile(t, filepath.Join(resources, "itms-oc-mirror.yaml"), "kind: ImageTagMirrorSet\n")

	cases := []struct {
		name      string
		manifests []string
		expected  []string
	}{
		{
			name:      "no manifests",
			manifests: nil,
			expected:  nil,
		},
		{
			name:      "imageset file and cluster-resources dir",
			manifests: []string{imageSet, resources},
			expected: []string{
				"manifests/cluster-resources/idms-oc-mirror.yaml",
				"manifests/cluster-resources/itms-oc-mirror.yaml",
				"manifests/imageset.yaml",
			},
		},
		{
			name:      "missing paths are skipped",
			manifests: []string{imageSet, filepath.Join(src, "mapping.txt")},
			expected:  []string{"manifests/imageset.yaml"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			useBundleDockerfile(t)
			tempDir := t.TempDir()
			ctrl := gomock.NewController(t)
			mockExec := executer.NewMockExecuter(ctrl)
			imageRef := registryImageRef(testPort, Tag(testVersion))

			var staged []string
			gomock.InOrder(
				mockExec.EXPECT().Execute(gomock.Any()).DoAndReturn(func(cmd string) (string, error) {
					if !strings.HasPrefix(cmd, "podman build -f bundle/Dockerfile.bundle -t "+imageRef+" "+tempDir) {
						t.Fatalf("unexpected build command: %s", cmd)
					}
					staged = listContext(t, cmd)
					return "", nil
				}),
				mockExec.EXPECT().Execute("podman push --tls-verify=false "+imageRef).Return("", nil),
			)

			b := NewBundle(BundleConfig{
				Executer:       mockExec,
				Port:           testPort,
				ReleaseVersion: testVersion,
				TempDir:        tempDir,
				Manifests:      tc.manifests,
			})
			if err := b.Push(); err != nil {
				t.Fatalf("push should succeed: %v", err)
			}

			if strings.Join(staged, ",") != strings.Join(tc.expected, ",") {
				t.Fatalf("staged manifests = %v, want %v", staged, tc.expected)
			}
			if entries, _ := os.ReadDir(tempDir); len(entries) != 0 {
				t.Fatalf("build context not cleaned up: %v", entries)
			}
		})
	}
}

func TestBundlePushFails(t *testing.T) {
	imageRef := registryImageRef(testPort, Tag(testVersion))

	cases := []struct {
		name        string
		buildErr    error
		pushErr     error
		expectedMsg string
	}{
		{
			name:        "build fails",
			buildErr:    errors.New("boom"),
			expectedMsg: "build release bundle image: boom",
		},
		{
			name:        "push fails",
			pushErr:     errors.New("push boom"),
			expectedMsg: "push release bundle image: push boom",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			useBundleDockerfile(t)
			ctrl := gomock.NewController(t)
			mockExec := executer.NewMockExecuter(ctrl)
			mockExec.EXPECT().Execute(gomock.Any()).Return("", tc.buildErr)
			if tc.buildErr == nil {
				mockExec.EXPECT().Execute("podman push --tls-verify=false "+imageRef).Return("", tc.pushErr)
			}

			b := NewBundle(BundleConfig{
				Executer:       mockExec,
				Port:           testPort,
				ReleaseVersion: testVersion,
				TempDir:        t.TempDir(),
			})

			err := b.Push()
			if err == nil || !strings.Contains(err.Error(), tc.expectedMsg) {
				t.Fatalf("expected error containing %q, got: %v", tc.expectedMsg, err)
			}
		})
	}
}

func TestBundleDockerfileMatchesFallback(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "bundle", "Dockerfile.bundle"))
	if err != nil {
		t.Fatalf("read Dockerfile.bundle: %v", err)
	}
	var instructions strings.Builder
	for _, line := range strings.Split(string(content), "\n") {
		if line != "" && !strings.HasPrefix(line, "#") {
			instructions.WriteString(line + "\n")
		}
	}
	if instructions.String() != fallbackDockerfile {
		t.Fatalf("bundle/Dockerfile.bundle = %q, want %q", instructions.String(), fallbackDockerfile)
	}
}

func TestResolveDockerfileNotFound(t *testing.T) {
	chdir(t, t.TempDir())

	if _, err := resolveDockerfile(); err == nil {
		t.Fatal("expected resolveDockerfile to fail when no Dockerfile exists")
	}
}
