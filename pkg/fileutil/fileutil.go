package fileutil

import (
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

func CopyFile(source, dest string) error {
	// Read source file
	bytesRead, err := ioutil.ReadFile(source)
	if err != nil {
		return err
	}

	// Get source file info
	fileinfo, err := os.Stat(source)
	if err != nil {
		return err
	}

	// Create dest dir
	if err := os.MkdirAll(filepath.Dir(dest), os.ModePerm); err != nil {
		return err
	}

	// Copy file to dest
	if err = ioutil.WriteFile(dest, bytesRead, fileinfo.Mode().Perm()); err != nil {
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

	_, err = io.Copy(writer, archive)
	return target, err
}
