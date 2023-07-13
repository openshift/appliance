package executer

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

// Command contains the description of a command to be executed by an executor.
type Command struct {
	// Env is a set of additional environment variables. The keys of the map are the names of
	// the variables, and the values of the map their values. If the value of a variable is nil
	// then it will be removed from the environment. Otherwise it will converted to a string and
	// added to the environment, replacing any previous value.
	Env map[string]any

	// Args are the command line arguments.
	Args []string
}

//go:generate mockgen -source=executer.go -package=executer -destination=mock_executer.go
type Executer interface {
	Execute(command Command) (string, error)
	TempFile(dir, pattern string) (f *os.File, err error)
}

type executer struct {
}

func NewExecuter() Executer {
	return &executer{}
}

func (e *executer) Execute(command Command) (result string, err error) {
	if len(command.Args) < 1 {
		err = errors.New("at least one command line argument is required")
		return
	}
	bin := command.Args[0]
	args := command.Args[1:]
	cmd := exec.Command(bin, args...)
	if command.Env != nil {
		cmd.Env = e.mergeEnv(os.Environ(), command.Env)
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err = cmd.Run()
	if err != nil {
		err = errors.Wrapf(err, "failed to execute cmd (%s): %s", bin, stderr.String())
		return
	}
	result = strings.TrimSuffix(stdout.String(), "\n")
	return
}

// mergeEnv merges the given environment with the given variables. The environment is expected to be
// a list of strings in the the same format used by the os.Environ function. The variables is a map
// describing the changes to make to that environment. The keys of the map are the names of the
// variables and the values of the map the values. Variables with value nil will be removed from the
// environment. The rest will be added, replacing any previous value.
func (e *executer) mergeEnv(env []string, vars map[string]any) []string {
	values := e.parseEnv(env)
	for name, value := range vars {
		if value == nil {
			delete(values, name)
		} else {
			values[name] = fmt.Sprintf("%s", value)
		}
	}
	return e.renderEnv(values)
}

// parseEnv takes a set of environment variables in the format used by os.Environ and puts them in a
// map where the keys of the map are the names of the variables and the values of the map the values
// of the variables.
func (e *executer) parseEnv(env []string) map[string]string {
	values := map[string]string{}
	for _, pair := range env {
		name, value := e.parseVar(pair)
		values[name] = value
	}
	return values
}

// parseVar parses splits a name value pair into its name and value, using the equals sign as
// separator. If there is no equal sign in the input the output will be the name of the variable and
// an empty string as value.
func (e *executer) parseVar(pair string) (name, value string) {
	index := strings.Index(pair, "=")
	if index != -1 {
		name = pair[0:index]
		value = pair[index+1:]
	} else {
		name = pair
		value = ""
	}
	return
}

// renderEnv converts a map containing name value pairs into a slice of strings with the same format
// used by os.Environ.
func (e *executer) renderEnv(values map[string]string) []string {
	names := maps.Keys(values)
	slices.Sort(names)
	env := make([]string, len(names))
	for i, name := range names {
		value := values[name]
		env[i] = e.renderVar(name, value)
	}
	return env
}

// renderVar renders an environment variable using the equals sign to separate the name from the value.
func (e *executer) renderVar(name, value string) string {
	return fmt.Sprintf("%s=%s", name, value)
}

func (e *executer) TempFile(dir, pattern string) (f *os.File, err error) {
	return os.CreateTemp(dir, pattern)
}

func FormatCommand(command string) (string, []string) {
	formattedCmd := strings.Split(command, " ")
	return formattedCmd[0], formattedCmd[1:]
}
