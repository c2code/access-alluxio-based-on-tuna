package configure

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"strings"

	"github.com/spf13/viper"

	//flag "github.com/spf13/pflag"

	"hexmeet.com/haishen/tuna/logp"
)

func init() {
	/*flag.BoolP("verbose", "v", false, "Log at INFO level")
	flag.Bool("toStderr", false, "Log to stderr and disable file output")
	flag.StringSliceP("debug", "d", nil, "Enable certain debug selectors")
	flag.String("log_config", "log.json", "Configure log")*/
}

// Logging builds a logp.Config based on configs.
func Logging(appName string) error {
	config := logp.DefaultConfig()
	config.AppName = appName

	if viper.IsSet("log_config") {
		logConfig := viper.GetString("log_config")

		content, err := ioutil.ReadFile(logConfig)
		if err != nil {
			log.Printf("logging: read log_config %s failed, %s", logConfig, err)
		} else {
			err = json.Unmarshal(content, &config)
			if err != nil {
				log.Panicf("logging: parse log_config %s failed, %s", logConfig, err)
			}
		}
	}

	applyFlags(&config)
	return logp.Configure(config)
}

func applyFlags(cfg *logp.Config) {
	verbose := viper.GetBool("verbose")
	debugSelectors := viper.GetStringSlice("debug")

	if viper.GetBool("toStderr") {
		cfg.ToStderr = true
	}
	if cfg.Level > logp.InfoLevel && verbose {
		cfg.Level = logp.InfoLevel
	}
	for _, selectors := range debugSelectors {
		cfg.Selectors = append(cfg.Selectors, strings.Split(selectors, ",")...)
	}
}
