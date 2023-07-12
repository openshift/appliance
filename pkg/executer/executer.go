package executer

import (
	"bytes"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

//go:generate mockgen -source=executer.go -package=executer -destination=mock_executer.go
type Executer interface {
	Execute(command string, args ...string) (string, error)
	TempFile(dir, pattern string) (f *os.File, err error)
}

type executer struct {
}

func NewExecuter() Executer {
	return &executer{}
}

func (e *executer) Execute(command string, args ...string) (string, error) {
	var stdoutBytes, stderrBytes bytes.Buffer
	cmd := exec.Command(command, args...)
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

func FormatCommand(command string) (string, []string) {
	formattedCmd := strings.Split(command, " ")
	return formattedCmd[0], formattedCmd[1:]
}
