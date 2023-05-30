package release

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-openapi/swag"
	"github.com/itchyny/gojq"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/executer"
	"github.com/openshift/appliance/pkg/registry"
	"github.com/openshift/appliance/pkg/templates"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/thedevsaddam/retry"
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
	ocMirrorAndUpload    = "oc mirror --config=%s docker://127.0.0.1:%s --dest-skip-tls --dir %s"
	ocAdmReleaseInfo     = "oc adm release info quay.io/openshift-release-dev/ocp-release:%s-%s -o json"
)

// Release is the interface to use the oc command to the get image info
type Release interface {
	ExtractFile(image string, filename string) (string, error)
	MirrorReleaseImages(envConfig *config.EnvConfig, applianceConfig *config.ApplianceConfig) error
	MirrorBootstrapImages(envConfig *config.EnvConfig, applianceConfig *config.ApplianceConfig) error
}

type release struct {
	executer                  executer.Executer
	envConfig                 *config.EnvConfig
	releaseImage              string
	pullSecret                string
	blockedBootstrapImages    map[string]bool
	additionalBootstrapImages map[string]bool
}

// NewRelease is used to set up the executor to run oc commands
func NewRelease(releaseImage, pullSecret string, envConfig *config.EnvConfig) Release {
	return &release{
		executer:                  executer.NewExecuter(),
		envConfig:                 envConfig,
		releaseImage:              releaseImage,
		pullSecret:                pullSecret,
		blockedBootstrapImages:    initBlockedBootstrapImagesInfo(),
		additionalBootstrapImages: initAdditionalImagesInfo(),
	}
}

func initBlockedBootstrapImagesInfo() map[string]bool {
	return map[string]bool{
		"agent-installer-api-server":               true,
		"agent-installer-csr-approver":             true,
		"agent-installer-orchestrator":             true,
		"agent-installer-node-agent":               true,
		"must-gather":                              true,
		"hyperkube":                                true,
		"cloud-credential-operator":                true,
		"cluster-policy-controller":                true,
		"pod":                                      true,
		"cluster-config-operator":                  true,
		"cluster-etcd-operator":                    true,
		"cluster-kube-scheduler-operator":          true,
		"machine-config-operator":                  true,
		"etcd":                                     true,
		"cluster-bootstrap":                        true,
		"cluster-ingress-operator":                 true,
		"cluster-kube-apiserver-operator":          true,
		"baremetal-installer":                      true,
		"keepalived-ipfailover":                    true,
		"baremetal-runtimecfg":                     true,
		"coredns":                                  true,
		"installer":                                true,
		"cluster-kube-controller-manager-operator": true,
	}
}

func initAdditionalImagesInfo() map[string]bool {
	return map[string]bool{
		"registry.redhat.io/ubi8/ubi:latest": true,
	}
}

// ExtractFile extracts the specified file from the given image name, and store it in the cache dir.
func (r *release) ExtractFile(image string, filename string) (string, error) {
	imagePullSpec, err := r.getImageFromRelease(image)
	if err != nil {
		return "", err
	}

	path, err := r.extractFileFromImage(imagePullSpec, filename, r.envConfig.CacheDir)
	if err != nil {
		return "", err
	}
	return path, err
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
	p := filepath.Join(cacheDir, path.Base(file))
	if _, err = os.Stat(p); err != nil {
		logrus.Debugf("File %s was not found, err %s", file, err.Error())
		return "", err
	}

	return p, nil
}

func (r *release) execute(executer executer.Executer, pullSecret, command string) (string, error) {
	executeCommand := command
	if pullSecret != "" {
		ps, err := executer.TempFile(r.envConfig.TempDir, "registry-config")
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
		executeCommand = command[:] + " --registry-config=" + ps.Name()
	}

	logrus.Debugf("Executing mirror command: %s", executeCommand)
	args := strings.Split(executeCommand, " ")

	stdout, err := executer.Execute(args[0], args[1:]...)
	if err == nil {
		return strings.TrimSpace(stdout), nil
	}

	return "", errors.Wrapf(err, "Failed to execute cmd (%s): %s", executeCommand, err)
}

func (r *release) setDockerConfig() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return errors.Wrapf(err, "Failed to get home directory")
	}

	configPath := filepath.Join(homeDir, ".docker", "config.json")
	if _, err := os.Stat(configPath); err == nil {
		logrus.Debugf("Using pull secret from: %s", configPath)
		return nil
	}

	if err = os.MkdirAll(filepath.Dir(configPath), os.ModePerm); err != nil {
		return err
	}

	if err = os.WriteFile(configPath, []byte(r.pullSecret), os.ModePerm); err != nil {
		return errors.Wrap(err, "failed to write file")
	}
	return nil
}

func (r *release) mirrorImages(envConfig *config.EnvConfig, applianceConfig *config.ApplianceConfig,
	templateFile string, blockedImages string, additionalImages string) error {
	if err := templates.RenderTemplateFile(
		templateFile,
		templates.GetImageSetTemplateData(applianceConfig, blockedImages, additionalImages),
		envConfig.TempDir); err != nil {
		return err
	}

	absPath, err := filepath.Abs(templates.GetFilePathByTemplate(templateFile, envConfig.TempDir))
	if err != nil {
		return err
	}

	tempDir, err := os.MkdirTemp(envConfig.TempDir, "oc-mirror")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	cmd := fmt.Sprintf(ocMirrorAndUpload, absPath, registry.RegistryPort, tempDir)
	logrus.Debugf("Fetching image from OCP release (%s)", cmd)

	if err = r.setDockerConfig(); err != nil {
		return err
	}

	result, err := r.execute(r.executer, "", cmd)
	logrus.Debugf("mirroring result: %s", result)
	if err != nil {
		return err
	}

	return err
}
func (r *release) MirrorReleaseImages(envConfig *config.EnvConfig, applianceConfig *config.ApplianceConfig) error {
	return r.mirrorImages(envConfig, applianceConfig, templates.ImageSetReleaseTemplateFile, "", "")
}

func (r *release) shouldBlockImage(imageName string) bool {
	action := "not blocking"
	_, found := r.blockedBootstrapImages[imageName]
	if !found {
		action = "blocking"
	}
	logrus.Debugf("%s image: %s", action, imageName)
	return !found
}

func (r *release) getImageInfo(v any) (string, string) {
	rawImageInfo := strings.Split(fmt.Sprintf("%#v", v), " ")
	imageName := strings.Replace(rawImageInfo[0], "\"", "", -1)
	return imageName, rawImageInfo[1]
}

func (r *release) generateBlockedImagesList(applianceConfig *config.ApplianceConfig) (string, error) {
	var releaseInfo map[string]any
	var result strings.Builder

	cmd := fmt.Sprintf(
		ocAdmReleaseInfo,
		applianceConfig.Config.OcpRelease.Version,
		swag.StringValue(applianceConfig.Config.OcpRelease.CpuArchitecture),
	)

	out, err := r.execute(r.executer, r.pullSecret, cmd)
	if err != nil {
		return "", err
	}
	if err = json.Unmarshal([]byte(out), &releaseInfo); err != nil {
		return "", err
	}

	query, err := gojq.Parse(QueryPattern)
	if err != nil {
		return "", err
	}
	iter := query.Run(releaseInfo)

	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok = v.(error); ok {
			return "", err
		}

		imageName, imageURL := r.getImageInfo(v)
		if r.shouldBlockImage(imageName) {
			result.WriteString(fmt.Sprintf("    - name: \"%s\n", imageURL))
		}
	}
	return result.String(), nil
}

func (r *release) generateAdditionalImagesList(imagesMap map[string]bool) string {
	var result strings.Builder
	var i int

	for imageURL := range imagesMap {
		result.WriteString(fmt.Sprintf("    - name: \"%s\"", imageURL))
		if i != len(r.additionalBootstrapImages) {
			result.WriteString("\n")
		}
		i++
	}
	return result.String()
}

func (r *release) imagesListWithCustomImages() map[string]bool {
	// TODO(MGMT-14548): Remove when no longer needed to use unofficial images.
	additionalImages := make(map[string]bool)
	for key, value := range r.additionalBootstrapImages {
		additionalImages[key] = value
	}
	additionalImages[templates.AssistedInstallerAgentImage] = true
	return additionalImages
}

func (r *release) MirrorBootstrapImages(envConfig *config.EnvConfig, applianceConfig *config.ApplianceConfig) error {
	blockedImages, err := r.generateBlockedImagesList(applianceConfig)
	if err != nil {
		return err
	}

	return r.mirrorImages(
		envConfig,
		applianceConfig,
		templates.ImageSetBootstrapTemplateFile,
		blockedImages,
		r.generateAdditionalImagesList(r.imagesListWithCustomImages()),
	)
}
