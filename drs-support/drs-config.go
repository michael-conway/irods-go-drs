package drs_support

import (
	"fmt"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

// Provides configuration for drs behaviors
type DrsConfig struct {
	DrsIdAvuValue string
	DrsAvuUnit    string
	DrsLogLevel   string //info, debug
}

func ReadDrsConfig(configName string, configType string, configPaths []string) (*DrsConfig, error) {
	viper.SetConfigName("config")         // name of config file (without extension)
	viper.SetConfigType("yaml")           // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath("/etc/appname/")  // path to look for the config file in
	viper.AddConfigPath("$HOME/.appname") // call multiple times to add many search paths
	viper.AddConfigPath(".")              // optionally look for config in the working directory
	err := viper.ReadInConfig()           // Find and read the config file
	if err != nil {                       // Handle errors reading the config file
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

}

func InitializeLogging(drsConfig *DrsConfig) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	switch drsConfig.DrsLogLevel {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)

	}
}
