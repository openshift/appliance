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
	"github.com/sirupsen/logrus"
	"github.com/thedevsaddam/retry"
	"sigs.k8s.io/yaml"
)

const (
	// OcDefaultTries is the number of times to execute the oc command on failures
	OcDefaultTries = 5
	// OcDefaultRetryDelay is the time between retries
	OcDefaultRetryDelay = time.Second * 5
	// QueryPattern formats the image names for a given release
	QueryPattern = ".references.spec.tags[] | .name + \" \" + .from.name"
)

const (
	templateGetImage     = "oc adm release info --image-for=%s --insecure=%t %s"
	templateExtractCmd   = "oc adm release extract --command=%s --to=%s %s"
	templateImageExtract = "oc image extract --path %s:%s --confirm %s"
	ocMirror             = "oc mirror --v2 --config=%s docker://127.0.0.1:%d --workspace=file://%s --src-tls-verify=false --dest-tls-verify=false --parallel-images=4 --parallel-layers=4 --retry-times=5 --ignore-release-signature"
	// ocMirrorDryRun is the command template for running oc mirror in dry-run mode to generate mapping.txt
	ocMirrorDryRun = "oc mirror --v2 --config=%s docker://127.0.0.1:%d --workspace=file://%s --src-tls-verify=false --dest-tls-verify=false --ignore-release-signature --dry-run"
)

// Release is the interface to use the oc command to the get image info
//
//go:generate mockgen -source=release.go -package=release -destination=mock_release.go
type Release interface {
	ExtractFile(image, filename string) (string, error)
	MirrorInstallImages() error
	GetImageFromRelease(imageName string) (string, error)
	ExtractCommand(command string, dest string) (string, error)
	GetMappingFile() ([]byte, error)
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
	image, err := r.execute(cmd)
	if err != nil {
		return "", err
	}

	// Fix incomplete image references from local registries
	image, err = r.fixImageReference(image, swag.StringValue(r.ApplianceConfig.Config.OcpRelease.URL))
	if err != nil {
		return "", err
	}

	return image, nil
}

// fixImageReference repairs incomplete image references returned by oc adm release info
// when querying local registries with custom ports. The oc command may return references
// like "registry.example.com@sha256:..." which are missing the port and repository path.
// This function reconstructs the full reference from the release URL.
func (r *release) fixImageReference(imageRef, releaseURL string) (string, error) {
	// Check if this looks like an incomplete reference (has @ but no / after the hostname)
	// Example: "virthost.ostest.test.metalkube.org@sha256:abc123"
	if strings.Contains(imageRef, "@") && !strings.Contains(strings.Split(imageRef, "@")[0], "/") {
		logrus.Debugf("Detected incomplete image reference: %s", imageRef)

		// Extract digest from the incomplete reference
		parts := strings.SplitN(imageRef, "@", 2)
		if len(parts) != 2 {
			return imageRef, nil // Return as-is if we can't parse it
		}
		digest := parts[1]

		// Extract registry/port/repo from release URL
		// Example: "virthost.ostest.test.metalkube.org:5000/openshift/release-images:tag"
		// We want: "virthost.ostest.test.metalkube.org:5000/openshift/release-images"
		releaseRef := releaseURL

		// Remove tag or digest from release URL
		if idx := strings.LastIndex(releaseRef, ":"); idx > strings.LastIndex(releaseRef, "/") {
			releaseRef = releaseRef[:idx]
		}
		if idx := strings.Index(releaseRef, "@"); idx != -1 {
			releaseRef = releaseRef[:idx]
		}

		// Reconstruct full image reference
		fixedRef := fmt.Sprintf("%s@%s", releaseRef, digest)
		logrus.Debugf("Fixed image reference: %s -> %s", imageRef, fixedRef)
		return fixedRef, nil
	}

	return imageRef, nil
}

func (r *release) extractFileFromImage(image, file, outputDir string) (string, error) {
	cmd := fmt.Sprintf(templateImageExtract, file, outputDir, image)
	logrus.Debugf("extracting %s to %s, %s", file, outputDir, cmd)
	_, err := retry.Do(OcDefaultTries, OcDefaultRetryDelay, r.execute, cmd)
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
	stdout, err := r.execute(cmd)
	if err != nil {
		return "", err
	}
	return stdout, nil
}

func (r *release) execute(command string) (string, error) {
	stdout, err := r.Executer.Execute(command)
	if err == nil {
		return strings.TrimSpace(stdout), nil
	}
	return "", err
}

func (r *release) mirrorImages(imageSetFile, blockedImages, additionalImages, operators string) error {
	var tempDir string

	// If a mirror path is provided in appliance-config, use it directly instead of running oc-mirror
	var mirrorPath string
	if r.ApplianceConfig.Config.MirrorPath != nil {
		mirrorPath = *r.ApplianceConfig.Config.MirrorPath
	}

	if mirrorPath != "" {
		logrus.Infof("Using pre-mirrored images from: %s", mirrorPath)
		tempDir = mirrorPath
	} else {
		// Normal mirroring flow - run oc-mirror
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

		tempDir = filepath.Join(r.EnvConfig.TempDir, "oc-mirror")
		registryPort := swag.IntValue(r.ApplianceConfig.Config.ImageRegistry.Port)
		cmd := fmt.Sprintf(ocMirror, imageSetFilePath, registryPort, tempDir)

		logrus.Debugf("Fetching image from OCP release (%s)", cmd)
		result, err := r.execute(cmd)
		logrus.Debugf("mirroring result: %s", result)
		if err != nil {
			return err
		}
	}

	// Copy generated yaml files to cache dir (works for both mirror path and oc-mirror output)
	if err := r.copyOutputYamls(tempDir); err != nil {
		return err
	}

	return nil
}

func (r *release) copyOutputYamls(ocMirrorDir string) error {
	yamlPaths, err := filepath.Glob(filepath.Join(ocMirrorDir, "working-dir", consts.OcMirrorResourcesDir, "*.yaml"))
	if err != nil {
		return err
	}
	for _, yamlPath := range yamlPaths {
		logrus.Debugf("Copying ymals from oc-mirror output: %s", yamlPath)
		yamlBytes, err := r.OSInterface.ReadFile(yamlPath)
		if err != nil {
			return err
		}

		// Replace localhost with internal registry URI
		buildRegistryURI := fmt.Sprintf("127.0.0.1:%d", swag.IntValue(r.ApplianceConfig.Config.ImageRegistry.Port))
		internalRegistryURI := fmt.Sprintf("%s:%d", registry.RegistryDomain, registry.RegistryPort)
		newYaml := strings.ReplaceAll(string(yamlBytes), buildRegistryURI, internalRegistryURI)

		// Write edited yamls to cache
		if err = r.OSInterface.MkdirAll(filepath.Join(r.EnvConfig.CacheDir, consts.OcMirrorResourcesDir), os.ModePerm); err != nil {
			return err
		}
		destYamlPath := filepath.Join(r.EnvConfig.CacheDir, consts.OcMirrorResourcesDir, filepath.Base(yamlPath))
		if err = r.OSInterface.WriteFile(destYamlPath, []byte(newYaml), os.ModePerm); err != nil {
			return err
		}
	}
	return nil
}

func (r *release) generateImagesList(images *[]types.Image) string {
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
		r.generateImagesList(r.ApplianceConfig.Config.BlockedImages),
		r.generateImagesList(r.ApplianceConfig.Config.AdditionalImages),
		r.generateOperatorsList(r.ApplianceConfig.Config.Operators),
	)
}

// GetMappingFile runs oc mirror in dry-run mode to generate and return the mapping.txt file content.
// mapping.txt is only generated with --dry-run flag.
func (r *release) GetMappingFile() ([]byte, error) {
	// Render the imageset template
	if err := templates.RenderTemplateFile(
		consts.ImageSetTemplateFile,
		templates.GetImageSetTemplateData(r.ApplianceConfig,
			r.generateImagesList(r.ApplianceConfig.Config.BlockedImages),
			r.generateImagesList(r.ApplianceConfig.Config.AdditionalImages),
			r.generateOperatorsList(r.ApplianceConfig.Config.Operators)),
		r.EnvConfig.TempDir); err != nil {
		return nil, err
	}

	imageSetFilePath, err := filepath.Abs(templates.GetFilePathByTemplate(consts.ImageSetTemplateFile, r.EnvConfig.TempDir))
	if err != nil {
		return nil, err
	}

	dryRunDir := filepath.Join(r.EnvConfig.TempDir, "oc-mirror-dry-run")
	registryPort := swag.IntValue(r.ApplianceConfig.Config.ImageRegistry.Port)
	dryRunCmd := fmt.Sprintf(ocMirrorDryRun, imageSetFilePath, registryPort, dryRunDir)

	logrus.Debugf("Running oc mirror dry-run to generate mapping file (%s)", dryRunCmd)
	result, err := r.execute(dryRunCmd)
	logrus.Debugf("dry-run result: %s", result)
	if err != nil {
		return nil, err
	}

	// Find and read the mapping file from dry-run output
	// In dry-run mode, oc-mirror puts the mapping file in working-dir/dry-run/mapping.txt
	mappingFilePath := filepath.Join(dryRunDir, "working-dir", "dry-run", consts.OcMirrorMappingFileName)

	return r.OSInterface.ReadFile(mappingFilePath)
}
