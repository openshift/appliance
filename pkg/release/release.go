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
	"github.com/openshift/appliance/pkg/consts"
	"github.com/openshift/appliance/pkg/executer"
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
	ocMirrorAndUpload    = "oc mirror --config=%s docker://127.0.0.1:%d --dest-skip-tls --dir %s"
	ocAdmReleaseInfo     = "oc adm release info quay.io/openshift-release-dev/ocp-release:%s-%s -o json"
)

// Release is the interface to use the oc command to the get image info
//
//go:generate mockgen -source=release.go -package=release -destination=mock_release.go
type Release interface {
	ExtractFile(image, filename string) (string, error)
	MirrorReleaseImages() error
	MirrorBootstrapImages() error
	GetImageFromRelease(imageName string) (string, error)
}

type ReleaseConfig struct {
	Executer        executer.Executer
	EnvConfig       *config.EnvConfig
	ApplianceConfig *config.ApplianceConfig
}

type release struct {
	ReleaseConfig
	blockedBootstrapImages    map[string]bool
	additionalBootstrapImages map[string]bool
}

// NewRelease is used to set up the executor to run oc commands
func NewRelease(config ReleaseConfig) Release {
	if config.Executer == nil {
		config.Executer = executer.NewExecuter()
	}

	return &release{
		ReleaseConfig:             config,
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
		"cluster-version-operator":                 true,
		"cluster-node-tuning-operator":             true,
	}
}

func initAdditionalImagesInfo() map[string]bool {
	return map[string]bool{
		"registry.redhat.io/ubi8/ubi:latest": true,
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
	image, err := r.execute(r.Executer, r.ApplianceConfig.Config.PullSecret, cmd)
	if err != nil {
		return "", err
	}

	return image, nil
}

func (r *release) extractFileFromImage(image, file, outputDir string) (string, error) {
	cmd := fmt.Sprintf(templateImageExtract, file, outputDir, image)

	logrus.Debugf("extracting %s to %s, %s", file, outputDir, cmd)
	_, err := retry.Do(OcDefaultTries, OcDefaultRetryDelay, r.execute, r.Executer, r.ApplianceConfig.Config.PullSecret, cmd)
	if err != nil {
		return "", err
	}
	// Make sure file exists after extraction
	p := filepath.Join(outputDir, path.Base(file))
	if _, err = os.Stat(p); err != nil {
		logrus.Debugf("File %s was not found, err %s", file, err.Error())
		return "", err
	}

	return p, nil
}

func (r *release) execute(executer executer.Executer, pullSecret, command string) (string, error) {
	executeCommand := command
	if pullSecret != "" {
		ps, err := executer.TempFile(r.EnvConfig.TempDir, "registry-config")
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

	stdout, err := executer.Execute(executeCommand)
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

	if err = os.WriteFile(configPath, []byte(r.ApplianceConfig.Config.PullSecret), os.ModePerm); err != nil {
		return errors.Wrap(err, "failed to write file")
	}
	return nil
}

func (r *release) mirrorImages(templateFile string, blockedImages string, additionalImages string) error {
	if err := templates.RenderTemplateFile(
		templateFile,
		templates.GetImageSetTemplateData(r.ApplianceConfig, blockedImages, additionalImages),
		r.EnvConfig.TempDir); err != nil {
		return err
	}

	absPath, err := filepath.Abs(templates.GetFilePathByTemplate(templateFile, r.EnvConfig.TempDir))
	if err != nil {
		return err
	}

	tempDir, err := os.MkdirTemp(r.EnvConfig.TempDir, "oc-mirror")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	cmd := fmt.Sprintf(ocMirrorAndUpload, absPath, swag.IntValue(r.ApplianceConfig.Config.ImageRegistry.Port), tempDir)
	logrus.Debugf("Fetching image from OCP release (%s)", cmd)

	if err = r.setDockerConfig(); err != nil {
		return err
	}

	result, err := r.execute(r.Executer, "", cmd)
	logrus.Debugf("mirroring result: %s", result)
	if err != nil {
		return err
	}

	return err
}

func (r *release) MirrorReleaseImages() error {
	return r.mirrorImages(consts.ImageSetTemplateFile, "", "")
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

func (r *release) generateBlockedImagesList() (string, error) {
	var releaseInfo map[string]any
	var result strings.Builder

	cmd := fmt.Sprintf(
		ocAdmReleaseInfo,
		r.ApplianceConfig.Config.OcpRelease.Version,
		swag.StringValue(r.ApplianceConfig.Config.OcpRelease.CpuArchitecture),
	)

	out, err := r.execute(r.Executer, r.ApplianceConfig.Config.PullSecret, cmd)
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
	additionalImages[consts.AssistedInstallerAgentImage] = true
	return additionalImages
}

func (r *release) MirrorBootstrapImages() error {
	blockedImages, err := r.generateBlockedImagesList()
	if err != nil {
		return err
	}

	return r.mirrorImages(
		consts.ImageSetTemplateFile,
		blockedImages,
		r.generateAdditionalImagesList(r.imagesListWithCustomImages()),
	)
}
