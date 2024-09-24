package fileutil

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/openshift/appliance/pkg/executer"
)

const (
	splitCmd = "split %s %s -b %s"
)

type OSInterface interface {
	MkdirTemp(dir, prefix string) (string, error)
	Stat(name string) (os.FileInfo, error)
	Remove(name string) error
	UserHomeDir() (string, error)
	MkdirAll(path string, perm os.FileMode) error
	WriteFile(name string, data []byte, perm os.FileMode) error
	ReadFile(name string) ([]byte, error)
	RemoveAll(path string) error
}

type OSFS struct{}

func (OSFS) MkdirTemp(dir, prefix string) (string, error) {
	return os.MkdirTemp(dir, prefix)
}

func (OSFS) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (OSFS) Remove(name string) error {
	return os.Remove(name)
}

func (OSFS) UserHomeDir() (string, error) {
	return os.UserHomeDir()
}

func (OSFS) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (OSFS) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}

func (OSFS) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

func (OSFS) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

func CopyFile(source, dest string) error {
	// Read source file
	bytesRead, err := os.ReadFile(source)
	if err != nil {
		return err
	}

	// Get source file info
	fileinfo, err := os.Stat(source)
	if err != nil {
		return err
	}

	// Create dest dir
	if err = os.MkdirAll(filepath.Dir(dest), os.ModePerm); err != nil {
		return err
	}

	// Copy file to dest
	if err = os.WriteFile(dest, bytesRead, fileinfo.Mode().Perm()); err != nil {
		return err
	}

	return nil
}

func ExtractCompressedFile(source, target string) (string, error) {
	reader, err := os.Open(source)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	archive, err := gzip.NewReader(reader)
	if err != nil {
		return "", err
	}
	defer archive.Close()

	target = filepath.Join(target, archive.Name)
	writer, err := os.Create(target)
	if err != nil {
		return "", err
	}
	defer writer.Close()

	_, err = io.Copy(writer, archive) // #nosec G110
	return target, err
}

func SplitFile(filePath, destPath, partSize string) error {
	exec := executer.NewExecuter()
	_, err := exec.Execute(fmt.Sprintf(splitCmd, filePath, destPath, partSize))
	return err
}
