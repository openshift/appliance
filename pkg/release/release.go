package release

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/buger/jsonparser"
	"github.com/danielerez/openshift-appliance/pkg/asset/config"
	"github.com/danielerez/openshift-appliance/pkg/executer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/thedevsaddam/retry"
)

const (
	//OcDefaultTries is the number of times to execute the oc command on failures
	OcDefaultTries = 5
	// OcDefaultRetryDelay is the time between retries
	OcDefaultRetryDelay = time.Second * 5

	// CPU architectures
	CPUArchitectureAMD64   = "amd64"
	CPUArchitectureX86     = "x86_64"
	CPUArchitectureARM64   = "arm64"
	CPUArchitectureAARCH64 = "aarch64"
)

// Release is the interface to use the oc command to the get image info
type Release interface {
	ExtractFile(image string, filename string) (string, error)
	GetReleaseArchitecture() (string, error)
	MirrorRelease(ocpReleaseImage *string) error
}

type release struct {
	executer     executer.Executer
	releaseImage string
	pullSecret   string
	cacheDir     string
	assetsDir    string
	tempDir      string
}

// NewRelease is used to set up the executor to run oc commands
func NewRelease(releaseImage, pullSecret string, envConfig *config.EnvConfig) Release {
	return &release{
		executer:     executer.NewExecuter(),
		releaseImage: releaseImage,
		pullSecret:   pullSecret,
		cacheDir:     envConfig.CacheDir,
		assetsDir:    envConfig.AssetsDir,
		tempDir:      envConfig.TempDir,
	}
}

const (
	templateGetImage      = "oc adm release info --image-for=%s --insecure=%t %s"
	templateImageExtract  = "oc image extract --path %s:%s --confirm %s"
	templateImageInfo     = "oc image info --output json %s"
	ocAdmMirrorCmd        = "oc adm release mirror --from=%s --to-dir %s"
	mirroredImagesDirName = "oc-mirror"
)

// ExtractFile extracts the specified file from the given image name, and store it in the cache dir.
func (r *release) ExtractFile(image string, filename string) (string, error) {
	imagePullSpec, err := r.getImageFromRelease(image)
	if err != nil {
		return "", err
	}

	path, err := r.extractFileFromImage(imagePullSpec, filename, r.cacheDir)
	if err != nil {
		return "", err
	}
	return path, err
}

func (r *release) GetReleaseArchitecture() (string, error) {
	cmd := fmt.Sprintf(templateImageInfo, r.releaseImage)
	imageInfoStr, err := r.execute(r.executer, r.pullSecret, cmd)
	if err != nil {
		return "", err
	}

	architecture, err := jsonparser.GetString([]byte(imageInfoStr), "config", "architecture")
	if err != nil {
		return "", err
	}

	// Convert architecture naming to supported values
	return r.normalizeCPUArchitecture(architecture), nil
}

func (r *release) getImageFromRelease(imageName string) (string, error) {
	cmd := fmt.Sprintf(templateGetImage, imageName, true, r.releaseImage)

	logrus.Debugf("Fetching image from OCP release (%s)", cmd)
	image, err := r.execute(r.executer, r.pullSecret, cmd)
	if err != nil {
		return "", err
	}

	return image, nil
}

func (r *release) extractFileFromImage(image, file, cacheDir string) (string, error) {
	cmd := fmt.Sprintf(templateImageExtract, file, cacheDir, image)

	logrus.Debugf("extracting %s to %s, %s", file, cacheDir, cmd)
	_, err := retry.Do(OcDefaultTries, OcDefaultRetryDelay, r.execute, r.executer, r.pullSecret, cmd)
	if err != nil {
		return "", err
	}
	// Make sure file exists after extraction
	path := filepath.Join(cacheDir, path.Base(file))
	if _, err := os.Stat(path); err != nil {
		logrus.Debugf("File %s was not found, err %s", file, err.Error())
		return "", err
	}

	return path, nil
}

func (r *release) execute(executer executer.Executer, pullSecret, command string) (string, error) {
	ps, err := executer.TempFile(r.tempDir, "registry-config")
	if err != nil {
		return "", err
	}
	logrus.Debugf("Created a temporary file: %s", ps.Name())
	defer func() {
		ps.Close()
		os.Remove(ps.Name())
	}()
	_, err = ps.Write([]byte(pullSecret))
	if err != nil {
		logrus.Errorf("Failed to write pull-secret data into %s: %s", ps.Name(), err.Error())
		return "", err
	}
	// flush the buffer to ensure the file can be read
	ps.Close()
	executeCommand := command[:] + " --registry-config=" + ps.Name()
	logrus.Debugf("Executing mirror command: %s", executeCommand)
	args := strings.Split(executeCommand, " ")

	stdout, err := executer.Execute(args[0], args[1:]...)
	if err == nil {
		return strings.TrimSpace(stdout), nil
	}

	return "", errors.Wrapf(err, "Failed to execute cmd (%s): %s", executeCommand, err)
}

func (r *release) MirrorRelease(ocpReleaseImage *string) error {
	if ocpReleaseImage == nil {
		return errors.New("OCP release image URL must be specified")
	}
	mirroredImagesPath := filepath.Join(r.assetsDir, mirroredImagesDirName)
	cmd := fmt.Sprintf(ocAdmMirrorCmd, *ocpReleaseImage, mirroredImagesPath)

	if err := os.MkdirAll(mirroredImagesPath, os.ModePerm); err != nil {
		logrus.Errorf("Failed to create dir: %s: %s", mirroredImagesPath, err.Error())
		return err
	}

	_, err := r.execute(r.executer, r.pullSecret, cmd)
	if err != nil {
		return err
	}

	return nil
}

func (r *release) normalizeCPUArchitecture(arch string) string {
	switch arch {
	case CPUArchitectureAMD64:
		return CPUArchitectureX86
	case CPUArchitectureARM64:
		return CPUArchitectureAARCH64
	default:
		return arch
	}
}
