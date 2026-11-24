package ignition

import (
	"bytes"
	"io"
	"path/filepath"

	igntypes "github.com/coreos/ignition/v2/config/v3_2/types"
	"github.com/pkg/errors"
	"github.com/vincent-petithory/dataurl"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
)

// validateSystemManifests checks that every file stored directly under dir is a single
// YAML document defining apiVersion and kind. assisted-service loads these files as-is
// (it doesn't split multi-document YAML for system manifests), so a bad file must fail
// the build rather than reach a cluster.
func validateSystemManifests(cfg *igntypes.Config, dir string) error {
	for _, f := range cfg.Storage.Files {
		if filepath.Dir(f.Path) != dir {
			continue
		}
		if err := validateSystemManifest(f); err != nil {
			return err
		}
	}
	return nil
}

func validateSystemManifest(f igntypes.File) error {
	var content []byte
	if f.Contents.Source != nil {
		decoded, err := dataurl.DecodeString(*f.Contents.Source)
		if err != nil {
			return errors.Wrapf(err, "invalid system manifest %s", f.Path)
		}
		content = decoded.Data
	}

	var docs []map[string]interface{}
	decoder := k8syaml.NewYAMLOrJSONDecoder(bytes.NewReader(content), 4096)
	for {
		var doc map[string]interface{}
		if err := decoder.Decode(&doc); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return errors.Wrapf(err, "invalid system manifest %s", f.Path)
		}
		if doc != nil {
			docs = append(docs, doc)
		}
	}
	if len(docs) != 1 {
		return errors.Errorf("system manifest %s must contain a single YAML document, found %d", f.Path, len(docs))
	}
	for _, field := range []string{"apiVersion", "kind"} {
		if value, ok := docs[0][field].(string); !ok || value == "" {
			return errors.Errorf("system manifest %s must define %s", f.Path, field)
		}
	}
	return nil
}
