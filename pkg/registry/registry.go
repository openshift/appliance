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
	registrySaveCmd      = "podman push %s dir:%s/registry"
	registryLoadCmd      = "skopeo copy dir:%s/registry containers-storage:localhost/registry:latest"
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
		cmd = fmt.Sprintf(registryStartCmdOcp, r.DataDirPath, r.Port, r.URI)
		logrus.Debugf("Running OCP docker-registry image with distribution entrypoint: %s", cmd)
	} else {
		cmd = fmt.Sprintf(registryStartCmd, r.DataDirPath, r.Port, r.URI)
		logrus.Debugf("Running registry image: %s", cmd)
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
	// Store image in dir format
	_, err = exec.Execute(fmt.Sprintf(registrySaveCmd, consts.RegistryImage, destDir))
	return err
}

func LoadRegistryImage(cacheDir string) error {
	exec := executer.NewExecuter()
	// Load image
	_, err := exec.Execute(fmt.Sprintf(registryLoadCmd, cacheDir))
	return err
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

// isVersionAtLeast checks if the given version string is at least the minimum version.
// It strips pre-release and build metadata (everything after and including the first '-')
// before comparison. E.g., "4.21.0-0.ci-2025-11-17-124207" becomes "4.21.0".
func isVersionAtLeast(versionStr, minVersionStr string) bool {
	// Strip everything after and including the first '-' character
	if idx := strings.Index(versionStr, "-"); idx != -1 {
		versionStr = versionStr[:idx]
	}

	ocpVer, err := version.NewVersion(versionStr)
	if err != nil {
		return false
	}

	minOcpVer, err := version.NewVersion(minVersionStr)
	if err != nil {
		return false
	}

	return ocpVer.GreaterThanOrEqual(minOcpVer)
}

func ocpVersionContainsDistributionRegistry(applianceConfig *config.ApplianceConfig) bool {
	versionStr := applianceConfig.Config.OcpRelease.Version
	result := isVersionAtLeast(versionStr, consts.MinOcpVersionContainingDistributionRegistry)
	logrus.Debugf("ocpVersionContainsDistributionRegistry: OCP version %s >= minimum version %s: %v",
		versionStr, consts.MinOcpVersionContainingDistributionRegistry, result)
	return result
}

// GetRegistryImageURI returns the registry image URI to use based on configuration priority
func GetRegistryImageURI(envConfig *config.EnvConfig, applianceConfig *config.ApplianceConfig) string {
	// First priority: appliance config imageRegistry.uri (user-specified)
	sourceRegistryUri := swag.StringValue(applianceConfig.Config.ImageRegistry.URI)
	if sourceRegistryUri != "" {
		return sourceRegistryUri
	}

	// Second priority: docker-registry from OCP release
	if ShouldUseOcpRegistry(envConfig, applianceConfig) {
		releaseConfig := release.ReleaseConfig{
			ApplianceConfig: applianceConfig,
			EnvConfig:       envConfig,
		}
		r := release.NewRelease(releaseConfig)
		imageRegistryUri, err := r.GetImageFromRelease("docker-registry")
		if err == nil && imageRegistryUri != "" {
			return imageRegistryUri
		}
	}

	// Last resort: Use an internally built registry image
	return consts.RegistryImage
}

func CopyRegistryImageIfNeeded(envConfig *config.EnvConfig, applianceConfig *config.ApplianceConfig) (string, error) {
	registryFilename := filepath.Base(consts.RegistryFilePath)
	fileInCachePath := filepath.Join(envConfig.CacheDir, registryFilename)

	// Determine source registry URI using the helper function
	sourceRegistryUri := GetRegistryImageURI(envConfig, applianceConfig)
	logrus.Infof("Registry image URI: %s", sourceRegistryUri)

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
			// and copy it to dir format to preserve digests
			logrus.Infof("Copying registry image from %s to %s", sourceRegistryUri, consts.RegistryImage)
			if err := skopeo.NewSkopeo(nil).CopyToFile(
				sourceRegistryUri,
				consts.RegistryImage,
				fileInCachePath); err != nil {
				return "", err
			}
		}
	} else {
		logrus.Debug("Reusing registry from cache")
	}

	// Load the registry image into podman storage
	if err := LoadRegistryImage(envConfig.CacheDir); err != nil {
		return "", err
	}

	// Copy registry from cache to data dir staging area
	// This staging area gets packaged into the data ISO (agentdata partition),
	// which is mounted at /mnt/agentdata/ in the appliance for disconnected installation
	fileInDataDir := filepath.Join(envConfig.TempDir, dataDir, imagesDir, consts.RegistryFilePath)
	exec := executer.NewExecuter()
	if err := os.MkdirAll(filepath.Dir(fileInDataDir), os.ModePerm); err != nil {
		logrus.Error(err)
		return "", err
	}
	if _, err := exec.Execute(fmt.Sprintf("cp -r %s %s", fileInCachePath, fileInDataDir)); err != nil {
		logrus.Error(err)
		return "", err
	}

	// Return localhost/registry:latest for use in podman run during the build process
	// LoadRegistryImage tagged the image as localhost/registry:latest in local storage
	// Note: For REGISTRY_IMAGE env var in the appliance, use GetRegistryImageURI() instead
	return consts.RegistryImage, nil
}
