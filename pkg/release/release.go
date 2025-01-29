package release

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-openapi/swag"
	"github.com/openconfig/goyang/pkg/indent"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/asset/registry"
	"github.com/openshift/appliance/pkg/consts"
	"github.com/openshift/appliance/pkg/executer"
	"github.com/openshift/appliance/pkg/fileutil"
	"github.com/openshift/appliance/pkg/templates"
	"github.com/openshift/appliance/pkg/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/thedevsaddam/retry"
	"sigs.k8s.io/yaml"
)

const (
	//OcDefaultTries is the number of times to execute the oc command on failures
	OcDefaultTries = 5
	// OcDefaultRetryDelay is the time between retries
	OcDefaultRetryDelay = time.Second * 5
	// QueryPattern formats the image names for a given release
	QueryPattern = ".references.spec.tags[] | .name + \" \" + .from.name"
)

const (
	templateGetImage     = "oc adm release info --image-for=%s --insecure=%t %s"
	templateImageExtract = "oc image extract --path %s:%s --confirm %s"
	ocMirrorToDir        = "oc mirror --config=%s docker://127.0.0.1:%d --dir %s --dest-use-http"
	ocAdmReleaseInfo     = "oc adm release info quay.io/openshift-release-dev/ocp-release:%s-%s -o json"
	templateExtractCmd   = "oc adm release extract --command=%s --to=%s %s"
)

// Release is the interface to use the oc command to the get image info
//
//go:generate mockgen -source=release.go -package=release -destination=mock_release.go
type Release interface {
	ExtractFile(image, filename string) (string, error)
	MirrorInstallImages() error
	GetImageFromRelease(imageName string) (string, error)
	ExtractCommand(command string, dest string) (string, error)
}

type ReleaseConfig struct {
	Executer        executer.Executer
	EnvConfig       *config.EnvConfig
	ApplianceConfig *config.ApplianceConfig
	OSInterface     fileutil.OSInterface
}

type release struct {
	ReleaseConfig
}

// NewRelease is used to set up the executor to run oc commands
func NewRelease(config ReleaseConfig) Release {
	if config.Executer == nil {
		config.Executer = executer.NewExecuter()
	}
	if config.OSInterface == nil {
		config.OSInterface = &fileutil.OSFS{}
	}

	return &release{
		ReleaseConfig: config,
	}
}

// ExtractFile extracts the specified file from the given image name, and store it in the cache dir.
func (r *release) ExtractFile(image string, filename string) (string, error) {
	imagePullSpec, err := r.GetImageFromRelease(image)
	if err != nil {
		return "", err
	}

	path, err := r.extractFileFromImage(imagePullSpec, filename, r.EnvConfig.CacheDir)
	if err != nil {
		return "", err
	}
	return path, err
}

func (r *release) GetImageFromRelease(imageName string) (string, error) {
	cmd := fmt.Sprintf(templateGetImage, imageName, true, swag.StringValue(r.ApplianceConfig.Config.OcpRelease.URL))

	logrus.Debugf("Fetching image from OCP release (%s)", cmd)
	image, err := r.execute(r.ApplianceConfig.Config.PullSecret, cmd)
	if err != nil {
		return "", err
	}

	return image, nil
}

func (r *release) extractFileFromImage(image, file, outputDir string) (string, error) {
	cmd := fmt.Sprintf(templateImageExtract, file, outputDir, image)

	logrus.Debugf("extracting %s to %s, %s", file, outputDir, cmd)
	_, err := retry.Do(OcDefaultTries, OcDefaultRetryDelay, r.execute, r.ApplianceConfig.Config.PullSecret, cmd)
	if err != nil {
		return "", err
	}
	// Make sure file exists after extraction
	p := filepath.Join(outputDir, path.Base(file))
	if _, err = r.OSInterface.Stat(p); err != nil {
		logrus.Debugf("File %s was not found, err %s", file, err.Error())
		return "", err
	}

	return p, nil
}

func (r *release) ExtractCommand(command string, dest string) (string, error) {
	cmd := fmt.Sprintf(templateExtractCmd, command, dest, *r.ApplianceConfig.Config.OcpRelease.URL)
	logrus.Debugf("extracting %s to %s, %s", command, dest, cmd)
	stdout, err := r.execute(r.ApplianceConfig.Config.PullSecret, cmd)
	if err != nil {
		return "", err
	}
	return stdout, nil
}

func (r *release) execute(pullSecret, command string) (string, error) {
	executeCommand := command
	if pullSecret != "" {
		ps, err := r.Executer.TempFile(r.EnvConfig.TempDir, "registry-config")
		if err != nil {
			return "", err
		}
		logrus.Debugf("Created a temporary file: %s", ps.Name())
		defer func() {
			ps.Close()
			_ = r.OSInterface.Remove(ps.Name())
		}()
		_, err = ps.Write([]byte(r.ApplianceConfig.Config.PullSecret))
		if err != nil {
			logrus.Errorf("Failed to write pull-secret data into %s: %s", ps.Name(), err.Error())
			return "", err
		}
		// flush the buffer to ensure the file can be read
		ps.Close()
		executeCommand = command[:] + " --registry-config=" + ps.Name()
	}

	stdout, err := r.Executer.Execute(executeCommand)

	if err == nil {
		return strings.TrimSpace(stdout), nil
	}

	return "", errors.Wrapf(err, "Failed to execute cmd (%s): %s", executeCommand, err)
}

func (r *release) setDockerConfig() error {
	homeDir, err := r.OSInterface.UserHomeDir()
	if err != nil {
		return errors.Wrapf(err, "Failed to get home directory")
	}

	configPath := filepath.Join(homeDir, ".docker", "config.json")
	if _, err = r.OSInterface.Stat(configPath); err == nil {
		logrus.Debugf("Using pull secret from: %s", configPath)
		return nil
	}

	if err = r.OSInterface.MkdirAll(filepath.Dir(configPath), os.ModePerm); err != nil {
		return err
	}

	if err = r.OSInterface.WriteFile(configPath, []byte(r.ApplianceConfig.Config.PullSecret), os.ModePerm); err != nil {
		return errors.Wrap(err, "failed to write file")
	}
	return nil
}

func (r *release) mirrorImages(imageSetFile, blockedImages, additionalImages, operators string) error {
	if err := templates.RenderTemplateFile(
		imageSetFile,
		templates.GetImageSetTemplateData(r.ApplianceConfig, blockedImages, additionalImages, operators),
		r.EnvConfig.TempDir); err != nil {
		return err
	}

	imageSetFilePath, err := filepath.Abs(templates.GetFilePathByTemplate(imageSetFile, r.EnvConfig.TempDir))
	if err != nil {
		return err
	}

	tempDir, err := r.OSInterface.MkdirTemp(r.EnvConfig.TempDir, "oc-mirror")
	if err != nil {
		return err
	}

	registryPort := swag.IntValue(r.ApplianceConfig.Config.ImageRegistry.Port)
	cmd := fmt.Sprintf(ocMirrorToDir, imageSetFilePath, registryPort, tempDir)
	logrus.Debugf("Fetching image from OCP release (%s)", cmd)

	if err = r.setDockerConfig(); err != nil {
		return err
	}

	result, err := r.execute("", cmd)
	logrus.Debugf("mirroring result: %s", result)
	if err != nil {
		return err
	}

	// Copy yaml files (imageContentSourcePolicy and catalogSource) to cache dir
	if err = r.copyOutputYamls(tempDir); err != nil {
		return err
	}

	if err = r.copyMappingFile(tempDir); err != nil {
		return err
	}

	return err
}

func (r *release) copyMappingFile(ocMirrorDir string) error {
	mappingFiles, err := filepath.Glob(filepath.Join(ocMirrorDir, fmt.Sprintf("results-*/%s", consts.OcMirrorMappingFileName)))
	if err != nil {
		return err
	}

	// The slice returned from Glob will have a single filename when running the application, but it will be empty when running the unit-tests since they don't create the files "oc mirror" generates
	for _, mappingFile := range mappingFiles {
		if err := fileutil.CopyFile(mappingFile, filepath.Join(r.EnvConfig.CacheDir, consts.OcMirrorMappingFileName)); err != nil {
			return err
		}
	}

	return nil
}

func (r *release) copyOutputYamls(ocMirrorDir string) error {
	yamlPaths, err := filepath.Glob(filepath.Join(ocMirrorDir, "results-*/*.yaml"))
	if err != nil {
		return err
	}
	for _, yamlPath := range yamlPaths {
		logrus.Debugf("Copying ymals from oc-mirror output: %s", yamlPath)
		yamlBytes, err := r.OSInterface.ReadFile(yamlPath)
		if err != nil {
			return err
		}

		// Replace registry URI with "registry.appliance.com:5000"
		buildRegistryURI := fmt.Sprintf("127.0.0.1:%d", swag.IntValue(r.ApplianceConfig.Config.ImageRegistry.Port))
		internalRegistryURI := fmt.Sprintf("%s:%d", registry.RegistryDomain, registry.RegistryPort)
		newYaml := strings.ReplaceAll(string(yamlBytes), buildRegistryURI, internalRegistryURI)

		// Write edited yamls to cache
		destYamlPath := filepath.Join(r.EnvConfig.CacheDir, filepath.Base(yamlPath))
		if err = r.OSInterface.WriteFile(destYamlPath, []byte(newYaml), os.ModePerm); err != nil {
			return err
		}
	}
	return nil
}

func (r *release) generateAdditionalImagesList(images *[]types.Image) string {
	if images == nil {
		return ""
	}

	var result strings.Builder
	obj, err := yaml.Marshal(images)
	if err != nil {
		return ""
	}
	result.WriteString(indent.String("    ", string(obj)))
	return result.String()
}

func (r *release) generateOperatorsList(operators *[]types.Operator) string {
	if operators == nil {
		return ""
	}
	var result strings.Builder
	obj, err := yaml.Marshal(operators)
	if err != nil {
		return ""
	}
	result.WriteString(indent.String("    ", string(obj)))
	return result.String()
}

func (r *release) MirrorInstallImages() error {
	return r.mirrorImages(
		consts.ImageSetTemplateFile,
		"",
		r.generateAdditionalImagesList(r.ApplianceConfig.Config.AdditionalImages),
		r.generateOperatorsList(r.ApplianceConfig.Config.Operators),
	)
}
