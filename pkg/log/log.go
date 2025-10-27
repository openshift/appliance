package log

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/term"
)

type Filehook struct {
	file      io.Writer
	formatter logrus.Formatter
	level     logrus.Level

	truncateAtNewLine bool
}

func NewFileHook(file io.Writer, level logrus.Level, formatter logrus.Formatter) *Filehook {
	return &Filehook{
		file:      file,
		formatter: formatter,
		level:     level,
	}
}

func NewFileHookWithNewlineTruncate(file io.Writer, level logrus.Level, formatter logrus.Formatter) *Filehook {
	f := NewFileHook(file, level, formatter)
	f.truncateAtNewLine = true
	return f
}

func (h Filehook) Levels() []logrus.Level {
	var levels []logrus.Level
	for _, level := range logrus.AllLevels {
		if level <= h.level {
			levels = append(levels, level)
		}
	}

	return levels
}

func (h Filehook) Fire(entry *logrus.Entry) error {
	// logrus reuses the same entry for each invocation of hooks.
	// so we need to make sure we leave them message field as we received.
	orig := entry.Message
	defer func() { entry.Message = orig }()

	msgs := []string{orig}
	if h.truncateAtNewLine {
		msgs = strings.Split(orig, "\n")
	}

	for _, msg := range msgs {
		// this makes it easier to call format on entry
		// easy without creating a new one for each split message.
		entry.Message = msg
		line, err := h.formatter.Format(entry)
		if err != nil {
			return err
		}

		if _, err := h.file.Write(line); err != nil {
			return err
		}
	}

	return nil
}

func SetupFileHook(baseDir string) func() {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		logrus.Fatal(errors.Wrap(err, "failed to create base directory for logs"))
	}

	logfile, err := os.OpenFile(filepath.Join(baseDir, ".openshift_appliance.log"), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		logrus.Fatal(errors.Wrap(err, "failed to open log file"))
	}

	originalHooks := logrus.LevelHooks{}
	for k, v := range logrus.StandardLogger().Hooks {
		originalHooks[k] = v
	}
	logrus.AddHook(NewFileHook(logfile, logrus.TraceLevel, &logrus.TextFormatter{
		DisableColors:          true,
		DisableTimestamp:       false,
		FullTimestamp:          true,
		DisableLevelTruncation: false,
	}))

	return func() {
		if err = logfile.Close(); err != nil {
			logrus.Errorf("Failed to close log file: %s", err.Error())
		}
		logrus.StandardLogger().ReplaceHooks(originalHooks)
	}
}

func SetupOutputHook(logLevel string) {
	logrus.SetOutput(io.Discard)
	logrus.SetLevel(logrus.TraceLevel)

	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		level = logrus.InfoLevel
	}

	logrus.AddHook(NewFileHookWithNewlineTruncate(os.Stderr, level, &logrus.TextFormatter{
		// Setting ForceColors is necessary because logrus.TextFormatter determines
		// whether to enable colors by looking at the output of the logger.
		// In this case, the output is io.Discard, which is not a terminal.
		// Overriding it here allows the same check to be done, but against the
		// hook's output instead of the logger's output.
		ForceColors:            term.IsTerminal(int(os.Stderr.Fd())),
		DisableTimestamp:       true,
		DisableLevelTruncation: true,
		DisableQuote:           true,
	}))
}
