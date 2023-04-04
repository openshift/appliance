package asset

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// user.cfg template
const (
	UserCfgFileName = "user.cfg"
	GrubTimeout = 10
	GrubDefault = 0
	GrubMenuEntryName = "Agent-Based Installer"
	LiveISO = "rhcos-412.86.202301311551-0"
)

// guestfish.sh template
const (
	GuestfishScriptFileName = "guestfish.sh"
	ApplianceFileName = "openshift-appliance"
	// ReservedPartitionGUID Set partition as Linux reserved partition: https://en.wikipedia.org/wiki/GUID_Partition_Table
	ReservedPartitionGUID = "8DA63339-0007-60C0-C436-083AC8230908"
)

const (
	RecoveryPartitionName = "agentrecovery"
)

func applyTemplateData(template *template.Template, templateData interface{}) (string, error) {
	buf := &bytes.Buffer{}
	if err := template.Execute(buf, templateData); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ReadFile Read data from the string reader, and, if the name ends with
// '.template', strip that extension from the name and render the template.
func ReadFile(name string, reader io.Reader, templateData interface{}) (finalName string, data []byte, err error) {
	data, err = io.ReadAll(reader)
	if err != nil {
		return name, []byte{}, err
	}
	if filepath.Ext(name) == ".template" {
		name = strings.TrimSuffix(name, ".template")
		tmpl := template.New(name)
		tmpl, err = tmpl.Parse(string(data))
		if err != nil {
			return name, data, err
		}
		stringData, err1 := applyTemplateData(tmpl, templateData)
		if err1 != nil {
			return "", nil, err1
		}
		data = []byte(stringData)
	}

	return name, data, nil
}


func WriteFile(name string, data []byte, directory string) error {
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, data, 0o644); err != nil { //nolint:gosec // no sensitive info
		return errors.Wrap(err, "failed to write file")
	}

	return nil
}

func RenderTemplateFile(filename string, templateData interface{}, templatesDir string, outputDir string) error {
	logrus.Infof("Rendering %s", filename)
	AssetsDir := http.Dir(templatesDir)
	templateName := fmt.Sprintf("%s.template", filename)
	file, err := AssetsDir.Open(templateName)
	defer file.Close()

	if err != nil {
		return err
	}
	name, data, err := ReadFile(templateName, file, templateData)
	if err != nil {
		return err
	}
	err = WriteFile(name, data, outputDir)
	if err != nil {
		return err
	}
	return nil
}