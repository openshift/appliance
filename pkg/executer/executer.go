package executer

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

//go:generate mockgen -source=executer.go -package=executer -destination=mock_executer.go
type Executer interface {
	Execute(command string) (string, error)
	ExecuteBackground(command string, envVars []string) error
	TempFile(dir, pattern string) (f *os.File, err error)
}

type executer struct {
}

func NewExecuter() Executer {
	return &executer{}
}

func (e *executer) Execute(command string) (string, error) {
	formattedCmd, args := e.formatCommand(command)
	logrus.Debugf("Running cmd: %s %s", formattedCmd, strings.Join(args[:], " "))
	cmd := exec.Command(formattedCmd, args...)
	combinedOutput, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.New(fmt.Sprintf("Failed to execute cmd (%s): %s", cmd, string(combinedOutput)))
	}
	return strings.TrimSuffix(string(combinedOutput), "\n"), nil
}

// Execute command in background
func (e *executer) ExecuteBackground(command string, envVars []string) error {
	formattedCmd, args := e.formatCommand(command)
	logrus.Debugf("Running cmd: %s %s", formattedCmd, strings.Join(args[:], " "))
	cmd := exec.Command(formattedCmd, args...)
	cmd.Env = envVars
	return cmd.Start()
}

func (e *executer) TempFile(dir, pattern string) (f *os.File, err error) {
	return os.CreateTemp(dir, pattern)
}

func (e *executer) formatCommand(command string) (string, []string) {
	formattedCmd := strings.Split(command, " ")
	return formattedCmd[0], formattedCmd[1:]
}
