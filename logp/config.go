package logp

// Config contains the configuration options for the logger.
type Config struct {
	AppName   string   `json:"-"`         // Name of the App (for default file name).
	JSON      bool     `json:"json"`      // Write logs as JSON.
	Level     Level    `json:"level"`     // Logging level (error, warning, info, debug).
	Selectors []string `json:"selectors"` // Selectors for debug level logging.

	toObserver bool `json:"to_observer"`
	ToStderr   bool `json:"to_stderr"`
	ToFiles    bool `json:"to_files"`

	Files FileConfig `json:"files"`

	addCaller   bool `json:"add_caller"`  // Adds package and line number info to messages.
	development bool `json:"development"` // Controls how DPanic behaves.
}

// FileConfig contains the configuration options for the file output.
type FileConfig struct {
	Path       string `json:"path"`
	Name       string `json:"name"`
	MaxSize    int    `json:"maxsize"`
	MaxBackups int    `json:"maxbackups"`
	MaxAge     int    `json:"maxage"`
	Compress   bool   `json:"compress"`
}

var defaultConfig = Config{
	JSON:    true,
	Level:   InfoLevel,
	ToFiles: true,
	Files: FileConfig{
		Path:       "logs",
		MaxSize:    10,
		MaxBackups: 20,
		MaxAge:     10,
	},
	addCaller: true,
}

// DefaultConfig returns the default config options.
func DefaultConfig() Config {
	return defaultConfig
}
