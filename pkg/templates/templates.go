package templates

import (
	"bytes"
	"embed"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

//go:embed scripts
var Scripts embed.FS

func RenderTemplateFile(fileName string, templateData interface{}, outputDir string) error {
	logrus.Debugf("Rendering %s", fileName)

	// Read the template file
	content, err := Scripts.ReadFile(fileName)
	if err != nil {
		return errors.Wrapf(err, "Failed reading file: %s", fileName)
	}

	// Apply data on template
	renderedFileName := strings.TrimSuffix(fileName, ".template")
	data, err := applyTemplateData(renderedFileName, content, templateData)
	if err != nil {
		return err
	}

	// Write the rendered file
	if err := writeFile(renderedFileName, data, outputDir); err != nil {
		return err
	}
	return nil
}

func GetFilePathByTemplate(templateFile, location string) string {
	fileName := strings.TrimSuffix(templateFile, ".template")
	return filepath.Join(location, fileName)
}

func writeFile(name string, data []byte, directory string) error {
	path := filepath.Join(directory, name)
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, os.ModePerm); err != nil {
		return errors.Wrap(err, "failed to write file")
	}

	return nil
}

func applyTemplateData(fileName string, templateFileContent []byte, templateData interface{}) ([]byte, error) {
	tmpl := template.New(fileName)
	tmpl, err := tmpl.Parse(string(templateFileContent))
	if err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}
	if err := tmpl.Execute(buf, templateData); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
