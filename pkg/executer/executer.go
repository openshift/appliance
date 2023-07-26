package executer

import (
	"bytes"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

//go:generate mockgen -source=executer.go -package=executer -destination=mock_executer.go
type Executer interface {
	Execute(command string) (string, error)
	TempFile(dir, pattern string) (f *os.File, err error)
}

type executer struct {
}

func NewExecuter() Executer {
	return &executer{}
}

func (e *executer) Execute(command string) (string, error) {
	var stdoutBytes, stderrBytes bytes.Buffer

	formattedCmd, args := e.formatCommand(command)
	logrus.Debugf("Running cmd: %s %s", formattedCmd, strings.Join(args[:], " "))
	cmd := exec.Command(formattedCmd, args...)
	cmd.Stdout = &stdoutBytes
	cmd.Stderr = &stderrBytes
	err := cmd.Run()
	if err != nil {
		return "", errors.Wrapf(err, "Failed to execute cmd (%s): %s", cmd, stderrBytes.String())
	}

	return strings.TrimSuffix(stdoutBytes.String(), "\n"), nil
}

func (e *executer) TempFile(dir, pattern string) (f *os.File, err error) {
	return os.CreateTemp(dir, pattern)
}

func (e *executer) formatCommand(command string) (string, []string) {
	formattedCmd := strings.Split(command, " ")
	return formattedCmd[0], formattedCmd[1:]
}
