package releasebundle

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/openshift/appliance/pkg/executer"
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

func writeBundleDockerfile(t *testing.T, dir string) {
	t.Helper()
	bundleDir := filepath.Join(dir, "bundle")
	if err := os.MkdirAll(bundleDir, os.ModePerm); err != nil {
		t.Fatalf("mkdir bundle dir: %v", err)
	}
	content := `FROM scratch
ARG BUNDLE_VERSION=unknown
ARG BUNDLE_RELEASE=1
COPY imageset.yaml /manifests/imageset.yaml
COPY mapping.txt /mirror/mapping.txt
COPY Dockerfile.bundle /root/buildinfo/Dockerfile
LABEL version=$BUNDLE_VERSION
`
	if err := os.WriteFile(filepath.Join(bundleDir, "Dockerfile.bundle"), []byte(content), 0o644); err != nil {
		t.Fatalf("write bundle dockerfile: %v", err)
	}
}

func writeImageSet(t *testing.T, dir string) string {
	t.Helper()
	p := filepath.Join(dir, "imageset.yaml")
	if err := os.WriteFile(p, []byte("kind: ImageSetConfiguration\napiVersion: mirror.openshift.io/v1alpha2\n"), 0o644); err != nil {
		t.Fatalf("write imageset: %v", err)
	}
	return p
}

func TestBundlePush(t *testing.T) {
	wd := t.TempDir()
	writeBundleDockerfile(t, wd)
	imageSetPath := writeImageSet(t, wd)
	chdir(t, wd)

	ctrl := gomock.NewController(t)
	mockExec := executer.NewMockExecuter(ctrl)

	const port = 5005
	tag := Tag("4.22.0-0.ci-2026-03-23-012741")
	imageRef := registryImageRef(port, tag)
	mockExec.EXPECT().Execute(gomock.Any()).DoAndReturn(func(cmd string) (string, error) {
		if !strings.Contains(cmd, "podman build") || !strings.Contains(cmd, imageRef) {
			t.Fatalf("unexpected build cmd: %q", cmd)
		}
		if !strings.Contains(cmd, "--build-arg BUNDLE_VERSION=") {
			t.Fatalf("expected BUNDLE_VERSION build-arg in cmd: %q", cmd)
		}
		return "", nil
	})
	mockExec.EXPECT().Execute("podman push --tls-verify=false "+imageRef).Return("", nil)

	b := NewBundle(BundleConfig{
		Executer:       mockExec,
		Port:           port,
		ReleaseVersion: "4.22.0-0.ci-2026-03-23-012741",
		ImageSetPath:   imageSetPath,
		MappingBytes:   []byte("registry.example/a:b=localhost:5005/foo/bar:b\n"),
	})

	if err := b.Push(); err != nil {
		t.Fatalf("push should succeed: %v", err)
	}
}

func TestBundlePushBuildFails(t *testing.T) {
	wd := t.TempDir()
	writeBundleDockerfile(t, wd)
	imageSetPath := writeImageSet(t, wd)
	chdir(t, wd)

	ctrl := gomock.NewController(t)
	mockExec := executer.NewMockExecuter(ctrl)

	const port = 5005
	mockExec.EXPECT().Execute(gomock.Any()).Return("", errors.New("boom"))

	b := NewBundle(BundleConfig{
		Executer:       mockExec,
		Port:           port,
		ReleaseVersion: "4.20.5-x86_64",
		ImageSetPath:   imageSetPath,
	})

	err := b.Push()
	if err == nil {
		t.Fatal("expected push to fail on build error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "build release bundle image") || !strings.Contains(msg, "boom") {
		t.Fatalf("expected wrapped build error, got: %v", err)
	}
}

func TestBundlePushPushFails(t *testing.T) {
	wd := t.TempDir()
	writeBundleDockerfile(t, wd)
	imageSetPath := writeImageSet(t, wd)
	chdir(t, wd)

	ctrl := gomock.NewController(t)
	mockExec := executer.NewMockExecuter(ctrl)

	const port = 5005
	tag := Tag("4.20.5-x86_64")
	imageRef := registryImageRef(port, tag)
	mockExec.EXPECT().Execute(gomock.Any()).Return("", nil)
	mockExec.EXPECT().Execute("podman push --tls-verify=false "+imageRef).Return("", errors.New("push boom"))

	b := NewBundle(BundleConfig{
		Executer:       mockExec,
		Port:           port,
		ReleaseVersion: "4.20.5-x86_64",
		ImageSetPath:   imageSetPath,
	})

	err := b.Push()
	if err == nil {
		t.Fatal("expected push to fail on push error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "push release bundle image") || !strings.Contains(msg, "push boom") {
		t.Fatalf("expected wrapped push error, got: %v", err)
	}
}

func TestBundlePushMissingImageSetPath(t *testing.T) {
	wd := t.TempDir()
	writeBundleDockerfile(t, wd)
	chdir(t, wd)

	b := NewBundle(BundleConfig{
		Port:           5005,
		ReleaseVersion: "4.20.0",
	})
	if err := b.Push(); err == nil || !strings.Contains(err.Error(), "ImageSetPath") {
		t.Fatalf("expected ImageSetPath error, got %v", err)
	}
}

func TestResolveDockerfileNotFound(t *testing.T) {
	wd := t.TempDir()
	chdir(t, wd)

	_, _, err := resolveDockerfile()
	if err == nil {
		t.Fatal("expected resolveDockerfile to fail when no Dockerfile exists")
	}
}
