package appliance

import (
	"io"
	"os"
	"path/filepath"

	"github.com/go-openapi/swag/conv"
	"github.com/openshift/assisted-image-service/pkg/isoeditor"
	"github.com/pkg/errors"
)

// fipsKargsValue is the kernel argument that places the booted system in FIPS
// mode. The leading space is required: isoeditor splices appended arguments
// over the newline terminating the "linux ..." line, so the content must carry
// its own separator.
const fipsKargsValue = " fips=1"

// kargsReader is indirected so tests can stub it; isoeditor.NewKargsReader
// needs a real ISO on disk.
var kargsReader = isoeditor.NewKargsReader

// fipsKargs returns the kernel arguments to append to the live ISO for the
// given enableFips setting, or an empty string when FIPS is not enabled.
func fipsKargs(enableFips *bool) string {
	if !conv.Value(enableFips) {
		return ""
	}
	return fipsKargsValue
}

// appendFipsKargs appends the FIPS kernel argument to the boot configuration of
// the ISO extracted at workDir, when enableFips is set. It is a no-op otherwise.
//
// coreosIsoPath is the unmodified base ISO; isoeditor reads the kernel argument
// embed area from it and returns the rewritten boot files, which overwrite the
// extracted copies in workDir before the ISO is repacked.
func appendFipsKargs(coreosIsoPath, workDir string, enableFips *bool) error {
	kargs := fipsKargs(enableFips)
	if kargs == "" {
		return nil
	}

	files, err := kargsReader(coreosIsoPath, kargs)
	if err != nil {
		return errors.Wrapf(err, "failed to read kernel arguments from %s", coreosIsoPath)
	}

	// Reachable when /coreos/kargs.json lists no paths. Succeeding here would
	// ship a non-FIPS ISO to a user who asked for FIPS.
	if len(files) == 0 {
		return errors.Errorf("FIPS was requested but %s declares no boot configuration files to patch", coreosIsoPath)
	}

	return writeKargFiles(workDir, files)
}

// writeKargFiles overwrites the extracted boot configuration files at workDir
// with the rewritten ones. Every reader is closed, including after a failure.
func writeKargFiles(workDir string, files []isoeditor.FileData) error {
	var firstErr error

	for _, fileData := range files {
		if err := writeOneKargFile(workDir, fileData); err != nil && firstErr == nil {
			firstErr = err
		}
		if err := fileData.Data.Close(); err != nil && firstErr == nil {
			firstErr = errors.Wrapf(err, "failed to close reader for %s", fileData.Filename)
		}
	}

	return firstErr
}

// writeOneKargFile truncates and rewrites a single boot configuration file. The
// file must already exist: creating one would leave the real boot config
// unpatched while reporting success.
func writeOneKargFile(workDir string, fileData isoeditor.FileData) error {
	path := filepath.Join(workDir, fileData.Filename)

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		return errors.Wrapf(err, "failed to open %s in the extracted ISO", fileData.Filename)
	}

	if _, err := io.Copy(file, fileData.Data); err != nil {
		_ = file.Close() // the write error below is the one worth reporting
		return errors.Wrapf(err, "failed to write %s", fileData.Filename)
	}

	if err := file.Close(); err != nil {
		return errors.Wrapf(err, "failed to close %s", fileData.Filename)
	}

	return nil
}
