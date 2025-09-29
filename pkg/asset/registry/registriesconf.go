package registry

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/containers/image/pkg/sysregistriesv2"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/consts"
	"github.com/openshift/installer/pkg/asset"
	agentManifests "github.com/openshift/installer/pkg/asset/agent/manifests"
	"github.com/pelletier/go-toml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	RegistryDomain      = "registry.appliance.openshift.com"
	RegistryPort        = 5000
	RegistryPortUpgrade = 5001
)

var (
	registriesConfFilename = filepath.Join("mirror", "registries.conf")
	idmsFileName           = filepath.Join(consts.OcMirrorResourcesDir, "idms-oc-mirror.yaml")
)

type ImageDigestMirrorSet struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Spec struct {
		ImageDigestMirrors []ImageDigestMirror `yaml:"imageDigestMirrors"`
	} `yaml:"spec"`
	Status struct {
		// You can define specific fields for Status if needed. Here it is an empty struct for now.
	} `yaml:"status"`
}
type ImageDigestMirror struct {
	Mirrors []string `yaml:"mirrors"`
	Source  string   `yaml:"source"`
}

// RegistriesConf generates the registries.conf file.
type RegistriesConf struct {
	File   *asset.File
	Config *sysregistriesv2.V2RegistriesConf

	fSys fs.FS
}

var _ asset.Asset = (*RegistriesConf)(nil)

// Name returns a human friendly name for the asset.
func (*RegistriesConf) Name() string {
	return "Mirror Registries Config"
}

// Dependencies returns all the dependencies directly needed to generate
// the asset.
func (*RegistriesConf) Dependencies() []asset.Asset {
	return []asset.Asset{
		&config.EnvConfig{},
		&config.ApplianceConfig{},
	}
}

// Generate generates the registries.conf file from install-config.
func (i *RegistriesConf) Generate(_ context.Context, dependencies asset.Parents) error {
	envConfig := &config.EnvConfig{}
	applianceConfig := &config.ApplianceConfig{}
	dependencies.Get(envConfig, applianceConfig)

	if i.fSys == nil {
		i.fSys = os.DirFS(envConfig.CacheDir)
	}

	releaseImagesLocation, releaseLocation := i.getEndpointLocations()
	registries := &sysregistriesv2.V2RegistriesConf{
		Registries: []sysregistriesv2.Registry{
			{
				Endpoint: sysregistriesv2.Endpoint{
					Location: releaseImagesLocation,
				},
				Mirrors: []sysregistriesv2.Endpoint{
					{
						Location: fmt.Sprintf("%s:%d/openshift/release-images", RegistryDomain, RegistryPort),
					},
					// Mirror for the upgrade registry
					{
						Location: fmt.Sprintf("%s:%d/openshift/release-images", RegistryDomain, RegistryPortUpgrade),
					},
				},
			},
			{
				Endpoint: sysregistriesv2.Endpoint{
					Location: releaseLocation,
				},
				Mirrors: []sysregistriesv2.Endpoint{
					{
						Location: fmt.Sprintf("%s:%d/openshift/release", RegistryDomain, RegistryPort),
					},
					// Mirror for the upgrade registry
					{
						Location: fmt.Sprintf("%s:%d/openshift/release", RegistryDomain, RegistryPortUpgrade),
					},
				},
			},
		},
	}

	registriesData, err := toml.Marshal(registries)
	if err != nil {
		return err
	}

	i.File = &asset.File{
		Filename: registriesConfFilename,
		Data:     registriesData,
	}

	return nil
}

func (i *RegistriesConf) getEndpointLocations() (string, string) {
	releaseImagesLocation := "quay.io/openshift-release-dev/ocp-release"
	releaseLocation := "quay.io/openshift-release-dev/ocp-v4.0-art-dev"

	idmsFile, err := fs.ReadFile(i.fSys, idmsFileName)
	if err != nil {
		logrus.Debugf("missing IDMS yaml (%v), fallback to defaults.", err)
		return releaseImagesLocation, releaseLocation
	}

	idmsManifests, err := agentManifests.GetMultipleYamls[ImageDigestMirrorSet](idmsFile)
	if err != nil {
		logrus.Debugf("could not decode YAML for %s (%v), fallback to defaults.", idmsFileName, err)
		return releaseImagesLocation, releaseLocation
	}

	for _, idms := range idmsManifests {
		for _, digestMirrors := range idms.Spec.ImageDigestMirrors {
			if len(digestMirrors.Mirrors) == 0 {
				continue
			}
			location := digestMirrors.Mirrors[0]
			if strings.HasSuffix(location, "release-images") {
				releaseImagesLocation = digestMirrors.Source
			} else if strings.HasSuffix(location, "release") {
				releaseLocation = digestMirrors.Source
			}
		}
	}
	logrus.Debugf("endpoints locations: %s, %s", releaseImagesLocation, releaseLocation)
	return releaseImagesLocation, releaseLocation
}

func (i *RegistriesConf) Load(f asset.FileFetcher) (bool, error) {
	file, err := f.FetchByName(registriesConfFilename)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, errors.Wrap(err, fmt.Sprintf("failed to load %s file", registriesConfFilename))
	}

	registriesConf := &sysregistriesv2.V2RegistriesConf{}
	if err := toml.Unmarshal(file.Data, registriesConf); err != nil {
		return false, errors.Wrapf(err, "failed to unmarshal %s", registriesConfFilename)
	}

	i.File, i.Config = file, registriesConf

	return true, nil
}

// Files returns the files generated by the asset.
func (i *RegistriesConf) Files() []*asset.File {
	if i.File != nil {
		return []*asset.File{i.File}
	}
	return []*asset.File{}
}
