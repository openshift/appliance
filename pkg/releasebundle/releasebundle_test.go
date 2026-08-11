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
	if err := os.WriteFile(filepath.Join(bundleDir, "Dockerfile.bundle"), []byte("FROM scratch\n"), 0o644); err != nil {
		t.Fatalf("write bundle dockerfile: %v", err)
	}
}

func TestBundlePush(t *testing.T) {
	wd := t.TempDir()
	writeBundleDockerfile(t, wd)
	chdir(t, wd)

	ctrl := gomock.NewController(t)
	mockExec := executer.NewMockExecuter(ctrl)

	const port = 5005
	tag := Tag("4.22.0-0.ci-2026-03-23-012741")
	imageRef := registryImageRef(port, tag)
	relDigest := "sha256:58bdf24405449be5c78a1f27a7b64fc9ee980e4bc3c9b169e8b3da08e50e0388"
	mockExec.EXPECT().Execute("podman build --annotation 'com.openshift.release-image-digest="+relDigest+"' -f bundle/Dockerfile.bundle -t "+imageRef+" bundle").Return("", nil)
	mockExec.EXPECT().Execute("podman push --tls-verify=false "+imageRef).Return("", nil)

	b := NewBundle(BundleConfig{
		Executer:       mockExec,
		Port:           port,
		ReleaseVersion: "4.22.0-0.ci-2026-03-23-012741",
		ReleaseImage:   "registry.ci.openshift.org/ocp/release@" + relDigest,
	})

	if err := b.Push(); err != nil {
		t.Fatalf("push should succeed: %v", err)
	}
}

func TestBundlePushBuildFails(t *testing.T) {
	wd := t.TempDir()
	writeBundleDockerfile(t, wd)
	chdir(t, wd)

	ctrl := gomock.NewController(t)
	mockExec := executer.NewMockExecuter(ctrl)

	const port = 5005
	tag := Tag("4.20.5-x86_64")
	imageRef := registryImageRef(port, tag)
	relDigest := "sha256:58bdf24405449be5c78a1f27a7b64fc9ee980e4bc3c9b169e8b3da08e50e0388"
	mockExec.EXPECT().Execute("podman build --annotation 'com.openshift.release-image-digest="+relDigest+"' -f bundle/Dockerfile.bundle -t "+imageRef+" bundle").Return("", errors.New("boom"))

	b := NewBundle(BundleConfig{
		Executer:       mockExec,
		Port:           port,
		ReleaseVersion: "4.20.5-x86_64",
		ReleaseImage:   "registry.ci.openshift.org/ocp/release@" + relDigest,
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
	chdir(t, wd)

	ctrl := gomock.NewController(t)
	mockExec := executer.NewMockExecuter(ctrl)

	const port = 5005
	tag := Tag("4.20.5-x86_64")
	imageRef := registryImageRef(port, tag)
	relDigest := "sha256:58bdf24405449be5c78a1f27a7b64fc9ee980e4bc3c9b169e8b3da08e50e0388"
	mockExec.EXPECT().Execute("podman build --annotation 'com.openshift.release-image-digest="+relDigest+"' -f bundle/Dockerfile.bundle -t "+imageRef+" bundle").Return("", nil)
	mockExec.EXPECT().Execute("podman push --tls-verify=false "+imageRef).Return("", errors.New("push boom"))

	b := NewBundle(BundleConfig{
		Executer:       mockExec,
		Port:           port,
		ReleaseVersion: "4.20.5-x86_64",
		ReleaseImage:   "registry.ci.openshift.org/ocp/release@" + relDigest,
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

func TestResolveDockerfileNotFound(t *testing.T) {
	wd := t.TempDir()
	chdir(t, wd)

	_, _, err := resolveDockerfile()
	if err == nil {
		t.Fatal("expected resolveDockerfile to fail when no Dockerfile exists")
	}
}
