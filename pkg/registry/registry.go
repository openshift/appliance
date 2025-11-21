package registry

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-openapi/swag"
	"github.com/hashicorp/go-version"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/consts"
	"github.com/openshift/appliance/pkg/executer"
	"github.com/openshift/appliance/pkg/fileutil"
	"github.com/openshift/appliance/pkg/release"
	"github.com/openshift/appliance/pkg/skopeo"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	registryStartCmd     = "podman run --net=host --privileged -d --name registry -v %s:/var/lib/registry --restart=always -e REGISTRY_HTTP_ADDR=0.0.0.0:%d %s"
	registryStartCmdOcp  = "podman run --net=host --privileged -d --name registry -v %s:/var/lib/registry --restart=always -e REGISTRY_HTTP_ADDR=0.0.0.0:%d -u 0 --entrypoint=/usr/bin/distribution containers-storage:%s serve /etc/registry/config.yaml"
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
	Executer       executer.Executer
	HTTPClient     HTTPClient
	Port           int
	URI            string
	DataDirPath    string
	UseBinary      bool
	UseOcpRegistry bool
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

	var cmd string
	if r.UseOcpRegistry {
		logrus.Debug("Running OCP docker-registry image with distribution entrypoint")
		cmd = fmt.Sprintf(registryStartCmdOcp, r.DataDirPath, r.Port, r.URI)
	} else {
		logrus.Debug("Running registry image")
		cmd = fmt.Sprintf(registryStartCmd, r.DataDirPath, r.Port, r.URI)
	}

	_, err := r.Executer.Execute(cmd)
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

// IsUsingOcpRegistry returns true if we're using the docker-registry from OCP release
// (i.e., user has not specified a custom imageRegistry.uri)
func IsUsingOcpRegistry(applianceConfig *config.ApplianceConfig) bool {
	return swag.StringValue(applianceConfig.Config.ImageRegistry.URI) == ""
}

// ShouldUseOcpRegistry determines if the OCP docker-registry image should be used
// based on the priority: user config > OCP release > internally built
// Returns true only if no user config is set AND OCP version >= 4.21 AND OCP release has docker-registry available
func ShouldUseOcpRegistry(envConfig *config.EnvConfig, applianceConfig *config.ApplianceConfig) bool {
	// Only use OCP registry if user hasn't configured their own imageRegistry.uri
	if swag.StringValue(applianceConfig.Config.ImageRegistry.URI) != "" {
		logrus.Debug("User-configured registry detected, not using OCP docker-registry")
		return false
	}

	// Check if OCP version supports docker-registry with distribution binary (>= 4.21)
	if !ocpVersionContainsDistributionRegistry(applianceConfig) {
		logrus.Debug("OCP version < 4.21, docker-registry does not contain distribution binary")
		return false
	}

	// No user-configured registry, check if OCP release has docker-registry
	releaseConfig := release.ReleaseConfig{
		ApplianceConfig: applianceConfig,
		EnvConfig:       envConfig,
	}
	r := release.NewRelease(releaseConfig)
	imageRegistryUri, err := r.GetImageFromRelease("docker-registry")
	if err == nil && imageRegistryUri != "" {
		logrus.Debug("OCP docker-registry available and will be used")
		return true
	}

	logrus.Debug("No OCP docker-registry available, using internally built registry")
	return false
}

func ocpVersionContainsDistributionRegistry(applianceConfig *config.ApplianceConfig) bool {
	minOcpVer, _ := version.NewVersion(consts.MinOcpVersionContainingDistributionRegistry)

	// Strip everything after and including the first '-' character
	// E.g., "4.21.0-0.ci-2025-11-17-124207" becomes "4.21.0"
	versionStr := applianceConfig.Config.OcpRelease.Version
	if idx := strings.Index(versionStr, "-"); idx != -1 {
		versionStr = versionStr[:idx]
	}

	ocpVer, _ := version.NewVersion(versionStr)
	result := ocpVer.GreaterThanOrEqual(minOcpVer)
	logrus.Debugf("ocpVersionContainsDistributionRegistry: OCP version %s >= minimum version %s: %v",
		ocpVer.String(), minOcpVer.String(), result)
	return result
}

func CopyRegistryImageIfNeeded(envConfig *config.EnvConfig, applianceConfig *config.ApplianceConfig) (string, error) {
	registryFilename := filepath.Base(consts.RegistryFilePath)
	fileInCachePath := filepath.Join(envConfig.CacheDir, registryFilename)

	// Determine source registry URI with priority: appliance config, docker-registry from OCP release, or internally built
	var sourceRegistryUri string

	// First priority: appliance config imageRegistry.uri (user-specified)
	sourceRegistryUri = swag.StringValue(applianceConfig.Config.ImageRegistry.URI)
	if sourceRegistryUri != "" {
		logrus.Infof("Using registry from appliance config: %s", sourceRegistryUri)
	} else if ShouldUseOcpRegistry(envConfig, applianceConfig) {
		// Second priority: docker-registry from OCP release
		releaseConfig := release.ReleaseConfig{
			ApplianceConfig: applianceConfig,
			EnvConfig:       envConfig,
		}
		r := release.NewRelease(releaseConfig)
		imageRegistryUri, _ := r.GetImageFromRelease("docker-registry")
		sourceRegistryUri = imageRegistryUri
		logrus.Infof("Using docker-registry from OCP release: %s", sourceRegistryUri)
	} else {
		// Last resort: Use an internally built registry image
		sourceRegistryUri = consts.RegistryImage
		logrus.Infof("Using internally built registry image: %s", sourceRegistryUri)
	}

	// Search for registry image in cache dir
	if fileName := envConfig.FindInCache(registryFilename); fileName == "" {
		// Not in cache, need to create it
		if sourceRegistryUri == consts.RegistryImage {
			// Build the registry image internally
			if err := BuildRegistryImage(envConfig.CacheDir); err != nil {
				return "", err
			}
		} else {
			// Pull the source registry image (docker-registry from OCP release or from appliance config)
			// and copy/tag it to localhost/registry:latest, then save to registry.tar
			logrus.Infof("Copying registry image from %s to %s", sourceRegistryUri, consts.RegistryImage)
			if err := skopeo.NewSkopeo(nil).CopyToFile(
				sourceRegistryUri,
				consts.RegistryImage,
				fileInCachePath); err != nil {
				return "", err
			}
		}
	} else {
		logrus.Debug("Reusing registry.tar from cache")
	}

	// Load the registry image into podman storage
	if err := LoadRegistryImage(envConfig.CacheDir); err != nil {
		return "", err
	}

	// Copy to data dir in temp
	fileInDataDir := filepath.Join(envConfig.TempDir, dataDir, imagesDir, consts.RegistryFilePath)
	if err := fileutil.CopyFile(fileInCachePath, fileInDataDir); err != nil {
		logrus.Error(err)
		return "", err
	}

	// Always return localhost/registry:latest as this is what will be loaded in the disconnected environment
	return consts.RegistryImage, nil
}
