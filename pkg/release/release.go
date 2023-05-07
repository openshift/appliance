package release

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/buger/jsonparser"
	"github.com/danielerez/openshift-appliance/pkg/asset/config"
	"github.com/danielerez/openshift-appliance/pkg/executer"
	"github.com/danielerez/openshift-appliance/pkg/templates"
	"github.com/go-openapi/swag"
	"github.com/itchyny/gojq"
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

	// QueryPattern formats the image names for a given release
	QueryPattern = ".references.spec.tags[] | .name + \" \" + .from.name"
)

// Release is the interface to use the oc command to the get image info
type Release interface {
	ExtractFile(image string, filename string) (string, error)
	GetReleaseArchitecture() (string, error)
	MirrorReleaseImages(envConfig *config.EnvConfig, applianceConfig *config.ApplianceConfig) error
	MirrorBootStrapImages(envConfig *config.EnvConfig, applianceConfig *config.ApplianceConfig) error
}

type release struct {
	executer                  executer.Executer
	releaseImage              string
	pullSecret                string
	cacheDir                  string
	assetsDir                 string
	tempDir                   string
	blockedBootstrapImages    map[string]bool
	additionalBootstrapImages map[string]bool
}

// NewRelease is used to set up the executor to run oc commands
func NewRelease(releaseImage, pullSecret string, envConfig *config.EnvConfig) Release {
	return &release{
		executer:                  executer.NewExecuter(),
		releaseImage:              releaseImage,
		pullSecret:                pullSecret,
		cacheDir:                  envConfig.CacheDir,
		assetsDir:                 envConfig.AssetsDir,
		tempDir:                   envConfig.TempDir,
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
		"cluster-kube-scheduler-operator ":         true,
		"machine-config-operator":                  true,
		"etcd":                                     true,
		"cluster-bootstrap":                        true,
		"cluster-ingress-operator":                 true,
		"cluster-kube-apiserver-operator":          true,
		"baremetal-installer":                      true,
		"keepalived-ipfailover":                    true,
		"baremetal-runtimecfg ":                    true,
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

const (
	templateGetImage     = "oc adm release info --image-for=%s --insecure=%t %s"
	templateImageExtract = "oc image extract --path %s:%s --confirm %s"
	templateImageInfo    = "oc image info --output json %s"
	ocMirrorAndUpload    = "oc mirror --config=%s docker://127.0.0.1:5000 --dest-skip-tls"
	ocAdmReleaseInfo     = "oc adm release info quay.io/openshift-release-dev/ocp-release:%s-%s -o json"
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

func (r *release) mirrorImages(envConfig *config.EnvConfig, applianceConfig *config.ApplianceConfig,
	templateFile string, blockedImages string, additionalImages string) error {
	if err := templates.RenderTemplateFile(
		templateFile,
		templates.GetImageSetTemplateData(applianceConfig, blockedImages, additionalImages),
		envConfig.TempDir); err != nil {
		return err
	}

	p, err := os.Getwd()
	if err != nil {
		return err
	}

	filePath := filepath.Join(p, templates.GetFilePathByTemplate(templateFile, envConfig.TempDir))
	cmd := fmt.Sprintf(ocMirrorAndUpload, filePath)
	logrus.Debugf("Fetching image from OCP release (%s)", cmd)

	if err = syscall.Chdir(filepath.Join(p, envConfig.TempDir)); err != nil {
		return err
	}
	result, err := r.execute(r.executer, "", cmd)
	logrus.Debugf("mirroring result: %s", result)

	// remove oc mirror leftovers
	if err = os.RemoveAll(filepath.Join(p, envConfig.TempDir, "oc-mirror-workspace")); err != nil {
		return err
	}
	if err = os.RemoveAll(filepath.Join(p, envConfig.TempDir, "metadata")); err != nil {
		return err
	}
	if err = syscall.Chdir(p); err != nil {
		return err
	}
	return err
}

func (r *release) MirrorReleaseImages(envConfig *config.EnvConfig, applianceConfig *config.ApplianceConfig) error {
	return r.mirrorImages(
		envConfig,
		applianceConfig,
		templates.ImageSetReleaseTemplateFile,
		"",
		r.generateAdditionalImagesList(r.additionalBootstrapImages),
	)
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
		r.normalizeCPUArchitecture(swag.StringValue(applianceConfig.Config.OcpRelease.CpuArchitecture)),
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
			result.WriteString(fmt.Sprintf("\n"))
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

func (r *release) MirrorBootStrapImages(envConfig *config.EnvConfig, applianceConfig *config.ApplianceConfig) error {
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
