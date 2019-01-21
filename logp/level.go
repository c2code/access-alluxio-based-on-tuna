package logp

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"go.uber.org/zap/zapcore"
)

// Level is a logging priority. Higher levels are more important.
type Level int8

// Logging levels.
const (
	DebugLevel Level = iota - 1
	InfoLevel
	WarnLevel
	ErrorLevel
)

var levelStrings = map[Level]string{
	DebugLevel: "debug",
	InfoLevel:  "info",
	WarnLevel:  "warning",
	ErrorLevel: "error",
}

var zapLevels = map[Level]zapcore.Level{
	DebugLevel: zapcore.DebugLevel,
	InfoLevel:  zapcore.InfoLevel,
	WarnLevel:  zapcore.WarnLevel,
	ErrorLevel: zapcore.ErrorLevel,
}

func convLevel(lvl string) (zapcore.Level, error) {
	for k, v := range levelStrings {
		if strings.EqualFold(lvl, v) {
			return zapLevels[k], nil
		}
	}

	return zapcore.InfoLevel, errors.New(fmt.Sprintf("unknown level %s", lvl))
}

// String returns the name of the logging level.
func (l Level) String() string {
	s, found := levelStrings[l]
	if found {
		return s
	}
	return fmt.Sprintf("Level(%d)", l)
}

// Enabled returns true if given level is enabled.
func (l Level) Enabled(level Level) bool {
	return level >= l
}

// Unpack unmarshals a level string to a Level. This implements
// ucfg.StringUnpacker.
func (l *Level) Unpack(str string) error {
	str = strings.ToLower(str)
	for level, name := range levelStrings {
		if name == str {
			*l = level
			return nil
		}
	}

	return errors.Errorf("invalid level '%v'", str)
}

func (l Level) zapLevel() zapcore.Level {
	z, found := zapLevels[l]
	if found {
		return z
	}
	return zapcore.InfoLevel
}
