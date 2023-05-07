package registry

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/danielerez/openshift-appliance/pkg/executer"
	"github.com/danielerez/openshift-appliance/pkg/templates"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	registryStartCmd = "podman run --privileged -d --name registry -p 5000:5000 -v %s:/var/lib/registry --restart=always -e REGISTRY_HTTP_ADDR=0.0.0.0:5000 %s"
	registryStopCmd  = "podman rm registry -f"

	registryURL                  = "http://127.0.0.1:5000"
	registryAttempts             = 3
	registrySleepBetweenAttempts = 5
)

type Registry interface {
	StartRegistry(registryDataPath string) error
	StopRegistry() error
}

type registry struct {
	executer executer.Executer
}

func NewRegistry() Registry {
	return &registry{
		executer: executer.NewExecuter(),
	}
}

func (r *registry) verifyRegistryAvailability(registryURL string) error {
	client := http.Client{
		Timeout: 5 * time.Second,
	}

	for i := 0; i < registryAttempts; i++ {
		logrus.Debugf("image registry availability check attempts %d/%d", i+1, registryAttempts)
		resp, err := client.Get(registryURL)
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
func (r *registry) StartRegistry(registryDataPath string) error {
	_ = r.StopRegistry()
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	filePath := filepath.Join(pwd, registryDataPath)

	if err = os.RemoveAll(filePath); err != nil {
		return err
	}
	if err = os.MkdirAll(filePath, os.ModePerm); err != nil {
		return err
	}

	cmd := fmt.Sprintf(registryStartCmd, filePath, templates.RegistryImage)
	logrus.Debugf("Running registry cmd: %s", cmd)
	args := strings.Split(cmd, " ")
	_, err = r.executer.Execute(args[0], args[1:]...)
	if err != nil {
		return errors.Wrapf(err, "registry start failure")
	}

	if err = r.verifyRegistryAvailability(registryURL); err != nil {
		return err
	}
	return nil
}
func (r *registry) StopRegistry() error {
	logrus.Debug("Stopping registry script")
	args := strings.Split(registryStopCmd, " ")
	_, err := r.executer.Execute(args[0], args[1:]...)
	if err != nil {
		return errors.Wrapf(err, "registry stop failure")

	}
	return nil
}
