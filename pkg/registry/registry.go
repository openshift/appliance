package registry

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/openshift/appliance/pkg/executer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	registryStartCmd = "podman run --net=host --privileged -d --name registry -v %s:/var/lib/registry --restart=always -e REGISTRY_HTTP_ADDR=0.0.0.0:%d %s"
	registryStopCmd  = "podman rm registry -f"

	registryAttempts             = 3
	registrySleepBetweenAttempts = 5
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
	_ = r.StopRegistry()

	if err = os.RemoveAll(r.DataDirPath); err != nil {
		return err
	}
	if err = os.MkdirAll(r.DataDirPath, os.ModePerm); err != nil {
		return err
	}

	_, err = r.Executer.Execute(fmt.Sprintf(registryStartCmd, r.DataDirPath, r.Port, r.URI))
	if err != nil {
		return errors.Wrapf(err, "registry start failure")
	}

	if err = r.verifyRegistryAvailability(r.registryURL); err != nil {
		return err
	}
	return nil
}

func (r *registry) StopRegistry() error {
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
	return filepath.Join(pwd, directory, subDirectory), nil
}
