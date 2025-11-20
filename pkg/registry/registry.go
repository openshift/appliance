package registry

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-openapi/swag"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/consts"
	"github.com/openshift/appliance/pkg/executer"
	"github.com/openshift/appliance/pkg/fileutil"
	"github.com/openshift/appliance/pkg/skopeo"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	registryStartCmd     = "podman run --net=host --privileged -d --name registry -v %s:/var/lib/registry --restart=always -e REGISTRY_HTTP_ADDR=0.0.0.0:%d %s"
	registryStopCmd      = "podman rm registry -f"
	registryBuildCmd     = "podman build -f Dockerfile.registry -t registry ."
	registrySaveCmd      = "podman save -o %s/registry.tar %s"
	registryLoadCmd      = "podman load -q -i %s/registry.tar"
	registryRunBinaryCmd = "/registry serve config.yml"

	registryAttempts             = 3
	registrySleepBetweenAttempts = 5

	dataDir   = "data"
	imagesDir = "images"
)

type Registry interface {
	StartRegistry() error
	StopRegistry() error
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type RegistryConfig struct {
	Executer    executer.Executer
	HTTPClient  HTTPClient
	Port        int
	URI         string
	DataDirPath string
	UseBinary   bool
}

type registry struct {
	RegistryConfig
	registryURL string
}

func NewRegistry(config RegistryConfig) Registry {
	if config.Executer == nil {
		config.Executer = executer.NewExecuter()
	}

	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{
			Timeout: 5 * time.Second,
		}
	}
	return &registry{
		RegistryConfig: config,
		registryURL:    fmt.Sprintf("http://127.0.0.1:%d", config.Port),
	}
}

func (r *registry) verifyRegistryAvailability(registryURL string) error {
	for i := 0; i < registryAttempts; i++ {
		logrus.Debugf("image registry availability check attempts %d/%d", i+1, registryAttempts)
		req, _ := http.NewRequest("GET", registryURL, nil)
		resp, err := r.HTTPClient.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			time.Sleep(registrySleepBetweenAttempts * time.Second)
			continue
		}
		if resp.StatusCode == http.StatusOK {
			return nil
		}
	}
	return errors.Errorf("image registry at %s was not available after %d attempts", registryURL, registryAttempts)
}

func (r *registry) StartRegistry() error {
	var err error
	if err = os.MkdirAll(r.DataDirPath, os.ModePerm); err != nil {
		return err
	}

	if r.UseBinary {
		err = r.runRegistryBinary()
	} else {
		err = r.runRegistryImage()
	}
	if err != nil {
		return err
	}

	if err = r.verifyRegistryAvailability(r.registryURL); err != nil {
		return err
	}

	return nil
}

func (r *registry) runRegistryImage() error {
	_ = r.StopRegistry()

	logrus.Debug("Running registry image")
	_, err := r.Executer.Execute(fmt.Sprintf(registryStartCmd, r.DataDirPath, r.Port, r.URI))
	if err != nil {
		return errors.Wrapf(err, "registry start failure")
	}
	return nil
}

func (r *registry) runRegistryBinary() error {
	// Add env vars
	envVars := os.Environ()
	envVars = append(envVars, fmt.Sprintf("REGISTRY_STORAGE_FILESYSTEM_ROOTDIRECTORY=%s", r.DataDirPath))
	envVars = append(envVars, fmt.Sprintf("REGISTRY_HTTP_ADDR=127.0.0.1:%d", r.Port))

	// Run the registry binary
	logrus.Debug("Running registry binary")
	err := r.Executer.ExecuteBackground(registryRunBinaryCmd, envVars)
	if err != nil {
		return errors.Wrapf(err, "registry binary run failure")
	}
	return nil
}

func (r *registry) StopRegistry() error {
	if r.UseBinary {
		return nil
	}
	logrus.Debug("Stopping registry container")
	_, err := r.Executer.Execute(registryStopCmd)
	if err != nil {
		return errors.Wrapf(err, "registry stop failure")
	}
	return nil
}

func GetRegistryDataPath(directory, subDirectory string) (string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(directory, pwd) {
		// Removes the working directory if already exists
		// (happens when using the openshift-appliance binary flow)
		directory = strings.ReplaceAll(directory, pwd, "")
	}
	return filepath.Join(pwd, directory, subDirectory), nil
}

func BuildRegistryImage(destDir string) error {
	exec := executer.NewExecuter()
	// Build image
	_, err := exec.Execute(registryBuildCmd)
	if err != nil {
		return err
	}
	// Store image
	_, err = exec.Execute(fmt.Sprintf(registrySaveCmd, destDir, consts.RegistryImage))
	return err
}

func LoadRegistryImage(cacheDir string) error {
	exec := executer.NewExecuter()
	// Load image
	_, err := exec.Execute(fmt.Sprintf(registryLoadCmd, cacheDir))
	return err
}

func CopyRegistryImageIfNeeded(envConfig *config.EnvConfig, applianceConfig *config.ApplianceConfig) (string, error) {
	registryFilename := filepath.Base(consts.RegistryFilePath)
	fileInCachePath := filepath.Join(envConfig.CacheDir, registryFilename)
	registryUri := swag.StringValue(applianceConfig.Config.ImageRegistry.URI)

	if registryUri == "" {
		// Use an internally built registry image
		registryUri = consts.RegistryImage
	}

	// Search for registry image in cache dir
	if fileName := envConfig.FindInCache(registryFilename); fileName != "" {
		logrus.Debug("Reusing registry.tar from cache")
		if err := LoadRegistryImage(envConfig.CacheDir); err != nil {
			return "", err
		}
	} else if registryUri == consts.RegistryImage {
		// Build the registry image internally
		if err := BuildRegistryImage(envConfig.CacheDir); err != nil {
			return "", err
		}
	} else {
		// Pulling the registry image and copying to cache
		if err := skopeo.NewSkopeo(nil).CopyToFile(
			registryUri,
			consts.RegistryImage,
			fileInCachePath); err != nil {
			return registryUri, err
		}
	}

	// Copy to data dir in temp
	fileInDataDir := filepath.Join(envConfig.TempDir, dataDir, imagesDir, consts.RegistryFilePath)
	if err := fileutil.CopyFile(fileInCachePath, fileInDataDir); err != nil {
		logrus.Error(err)
		return "", err
	}

	return registryUri, nil
}
