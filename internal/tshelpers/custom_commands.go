package tshelpers

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cavaliercoder/go-cpio"
	igntypes "github.com/coreos/ignition/v2/config/v3_2/types"
	"github.com/diskfs/go-diskfs"
	"github.com/openshift/assisted-image-service/pkg/isoeditor"
	"github.com/pkg/diff"
	"github.com/pkg/errors"
	"github.com/rogpeppe/go-internal/testscript"
	"github.com/vincent-petithory/dataurl"
)

// Command         | Short description                                                                     | Usage example
// ----------------|---------------------------------------------------------------------------------------|--------------------------------------------------------------------------------------
// IsoCmp          | check that the content of the file extracted from the ISO embedded configuration file | isocmp agent.x86_64.iso /etc/assisted/manifests/infraenv.yaml expected/infraenv.yaml
//                 | matches the content of the local file                                                 |
// IsoCmpRegEx     | like IsoCmp, but the local file is a regex matched against the ISO file               | isocmpregex agent.x86_64.iso /etc/assisted/rendezvous-host.env expected/rendezvous-host.env
// IsoFileCmpRegEx | match a file of the ISO filesystem (e.g. boot configuration) against a regex          | isoFileCmpRegEx agent.x86_64.iso /EFI/centos/grub.cfg expected/grub.cfg
// SetReleaseImage | resolve the tag of a release image to its digest-pinned pullspec (name@digest) and    | setreleaseimage RELEASE_IMAGE quay.io/okd/scos-release:5.0.0-okd-scos.8
//                 | store it in an env var. The ISO manifests reference the release image by digest,      |
//                 | while the config only has a tag, and the digest is known only at build time (it can   |
//                 | change if the tag is re-pushed, e.g. OKD). Expected files use the env var instead of  |
//                 | a hardcoded digest (releaseImage: $RELEASE_IMAGE). Needs tools/oc and                 |
//                 | auth/pull-secret.json                                                                 |

const (
	ignitionImgPath = "/images/ignition.img"
	ignitionFile    = "config.ign"
)

// IsoCmp `isoPath` `isoFile` `expectedFile` check that the content of the file
// `isoFile` - extracted from the ISO embedded configuration file referenced
// by `isoPath` - matches the content of the local file `expectedFile`.
// Environment variables in `expectedFile` are substituted before the comparison.
func IsoCmp(ts *testscript.TestScript, neg bool, args []string) {
	isoCmp(ts, neg, args, "isocmp", byteCompare)
}

// IsoCmpRegEx behaves like IsoCmp, but `expectedFile` is a regular expression
// matched against the content of `isoFile`.
func IsoCmpRegEx(ts *testscript.TestScript, neg bool, args []string) {
	isoCmp(ts, neg, args, "isocmpregex", byteCompareRegEx)
}

type compareFunc func(ts *testscript.TestScript, neg bool, aData, eData []byte, aFilePath, eFilePath string)

func isoCmp(ts *testscript.TestScript, neg bool, args []string, name string, compare compareFunc) {
	if len(args) != 3 {
		ts.Fatalf("usage: %s isoPath file1 file2", name)
	}

	workDir := ts.Getenv("WORK")
	isoPath, aFilePath, eFilePath := args[0], args[1], args[2]

	aData, err := readFileFromISO(filepath.Join(workDir, isoPath), aFilePath)
	ts.Check(err)

	eData, err := os.ReadFile(filepath.Join(workDir, eFilePath))
	ts.Check(err)

	compare(ts, neg, aData, eData, aFilePath, eFilePath)
}

// SetReleaseImage `var` `image` resolves the current digest of the release image
// `image` (usually referenced by tag in the config) using the oc binary and the
// pull secret stored by the build in the work directory (tools/oc, auth/pull-secret.json),
// and sets the environment variable `var` to its pinned pullspec (name@digest).
// The ISO manifests reference the release image by digest, which is known only at build
// time and may change when the tag is re-pushed (e.g. the OKD ones): expected files
// reference `var` instead of hardcoding it.
func SetReleaseImage(ts *testscript.TestScript, neg bool, args []string) {
	if len(args) != 2 {
		ts.Fatalf("usage: setreleaseimage var image")
	}

	workDir := ts.Getenv("WORK")
	cmd := exec.Command(filepath.Join(workDir, "tools", "oc"), "adm", "release", "info", args[1],
		"-o", "template", "--template", "{{.digest}}",
		"--registry-config", filepath.Join(workDir, "auth", "pull-secret.json"))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	ts.Check(errors.Wrapf(err, "failed to get the digest of %s: %s", args[1], stderr.String()))

	image := args[1]
	if idx := strings.LastIndex(image, ":"); idx > strings.LastIndex(image, "/") {
		image = image[:idx]
	}
	ts.Setenv(args[0], image+"@"+strings.TrimSpace(string(out)))
}

// IsoFileCmpRegEx `isoPath` `isoFile` `expectedFile` check that the content of the ISO
// filesystem file `isoFile` matches the regular expression in the local file `expectedFile`.
// Environment variables in `expectedFile` are substituted before the comparison.
func IsoFileCmpRegEx(ts *testscript.TestScript, neg bool, args []string) {
	if len(args) != 3 {
		ts.Fatalf("usage: isofilecmpregex isoPath file1 file2")
	}

	workDir := ts.Getenv("WORK")
	isoPath, aFilePath, eFilePath := args[0], args[1], args[2]

	disk, err := diskfs.Open(filepath.Join(workDir, isoPath), diskfs.WithOpenMode(diskfs.ReadOnly))
	ts.Check(err)

	fs, err := disk.GetFilesystem(0)
	ts.Check(err)

	aFile, err := fs.OpenFile(aFilePath, os.O_RDONLY)
	ts.Check(err)
	defer func() { _ = aFile.Close() }()

	aData, err := io.ReadAll(aFile)
	ts.Check(err)

	eData, err := os.ReadFile(filepath.Join(workDir, eFilePath))
	ts.Check(err)

	byteCompareRegEx(ts, neg, aData, eData, aFilePath, eFilePath)
}

func expand(ts *testscript.TestScript, s []byte) string {
	return os.Expand(string(s), func(key string) string {
		if key == "$" {
			return "$"
		}
		return ts.Getenv(key)
	})
}

func byteCompare(ts *testscript.TestScript, neg bool, aData, eData []byte, aFilePath, eFilePath string) {
	byteCompareInternal(ts, neg, aData, eData, aFilePath, eFilePath, func(aText, eText string) (bool, error) {
		return aText == eText, nil
	})
}

func byteCompareRegEx(ts *testscript.TestScript, neg bool, aData, eData []byte, aFilePath, eFilePath string) {
	byteCompareInternal(ts, neg, aData, eData, aFilePath, eFilePath, func(aText, eText string) (bool, error) {
		return regexp.MatchString(eText, aText)
	})
}

func byteCompareInternal(ts *testscript.TestScript, neg bool, aData, eData []byte, aFilePath, eFilePath string, cmp func(string, string) (bool, error)) {
	aText := string(aData)
	eText := expand(ts, eData)

	eq, err := cmp(aText, eText)
	if err != nil {
		ts.Fatalf("unexpected error while comparing strings: %v", err)
	}
	if neg {
		if eq {
			ts.Fatalf("%s and %s do not differ", aFilePath, eFilePath)
		}
		return
	}
	if eq {
		return
	}

	ts.Logf("%s", aText)

	var sb strings.Builder
	ts.Check(diff.Text(eFilePath, aFilePath, eText, aText, &sb))

	ts.Logf("%s", sb.String())
	ts.Fatalf("%s and %s differ", eFilePath, aFilePath)
}

// readFileFromISO returns the content of the ignition storage file `nodePath`
// embedded in the ISO `isoPath`.
func readFileFromISO(isoPath, nodePath string) ([]byte, error) {
	f, err := findIgnitionFile(isoPath, nodePath)
	if err != nil {
		return nil, err
	}

	actualData, err := dataurl.DecodeString(*f.Contents.Source)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode %s", nodePath)
	}
	return actualData.Data, nil
}

// findIgnitionFile returns the ignition storage file `nodePath` embedded in the ISO `isoPath`.
func findIgnitionFile(isoPath, nodePath string) (igntypes.File, error) {
	config, err := extractIgnition(isoPath)
	if err != nil {
		return igntypes.File{}, err
	}

	for _, f := range config.Storage.Files {
		if f.Path == nodePath {
			return f, nil
		}
	}

	return igntypes.File{}, errors.Errorf("file %s not found within the %s archive", nodePath, ignitionImgPath)
}

func extractIgnition(isoPath string) (igntypes.Config, error) {
	var config igntypes.Config

	rawContent, err := extractArchiveFile(isoPath)
	if err != nil {
		return config, err
	}

	if err := json.Unmarshal(rawContent, &config); err != nil {
		return config, errors.Wrap(err, "failed to parse ignition config")
	}

	return config, nil
}

// extractArchiveFile returns the ignition config stored in the ISO gzipped
// cpio ignition image. The image is zero-padded, so only the first gzip
// member is read.
func extractArchiveFile(isoPath string) ([]byte, error) {
	ignitionImg, err := isoeditor.ReadFileFromISO(isoPath, strings.TrimPrefix(ignitionImgPath, "/"))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read %s", ignitionImgPath)
	}

	gzipReader, err := gzip.NewReader(bytes.NewReader(ignitionImg))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open %s", ignitionImgPath)
	}
	gzipReader.Multistream(false)

	cpioReader := cpio.NewReader(gzipReader)
	for {
		header, err := cpioReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read %s", ignitionImgPath)
		}
		if header.Name == ignitionFile {
			return io.ReadAll(cpioReader)
		}
	}

	return nil, errors.Errorf("file %s not found within the %s archive", ignitionFile, ignitionImgPath)
}
