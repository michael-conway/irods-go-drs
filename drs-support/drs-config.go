package drs_support

import (
	"fmt"
	"github.com/cyverse/go-irodsclient/irods/types"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

// DrsConfig Provides configuration for drs behaviors
type DrsConfig struct {
	DrsIdAvuValue            string
	DrsAvuUnit               string
	DrsLogLevel              string //info, debug
	IrodsHost                string
	IrodsPort                int
	IrodsZone                string
	IrodsDrsAdminUser        string
	IrodsDrsAdminPassword    string
	IrodsDrsAdminLoginType   string
	IrodsDefaultLoginType    string
	IrodsAuthScheme          string
	IrodsNegotiationPolicy   string
	IrodsNegotiationRequired bool
	IrodsDefaultResource     string
}

func (cfg *DrsConfig) ToIrodsAccount() types.IRODSAccount {
	authScheme := types.GetAuthScheme(cfg.IrodsAuthScheme)

	negotiationPolicy := types.GetCSNegotiationPolicyRequest(cfg.IrodsNegotiationPolicy)
	negotiation := types.GetCSNegotiation(cfg.IrodsNegotiationPolicy)

	account := types.IRODSAccount{
		AuthenticationScheme:    authScheme,
		ClientServerNegotiation: negotiation.IsNegotiationRequired(),
		CSNegotiationPolicy:     negotiationPolicy,
		Host:                    cfg.IrodsHost,
		Port:                    cfg.IrodsPort,
		ClientUser:              cfg.IrodsDrsAdminUser,
		ClientZone:              cfg.IrodsZone,
		ProxyUser:               cfg.IrodsDrsAdminUser,
		ProxyZone:               cfg.IrodsZone,
		Password:                cfg.IrodsDrsAdminPassword,
		DefaultResource:         cfg.IrodsDefaultResource,
	}

	account.FixAuthConfiguration()

	return account
}

const DefaultConfigName = "drs-config"
const DefaultConfigType = "yaml"

// ReadDrsConfig reads the configuration for DRS behaviors in irods
// can take a number of paths that will be prefixed in the searh path, or defaults
// may be accepted, blank params for name and type defaut to irods-drs.yaml
func ReadDrsConfig(configName string, configType string, configPaths []string) (*DrsConfig, error) {

	if configName == "" {
		viper.SetConfigName(DefaultConfigName) // name of config file (without extension)
	} else {
		viper.SetConfigName(configName)
	}

	if configType == "" {
		viper.SetConfigType(DefaultConfigType) // REQUIRED if the config file does not have the extension in the name
	} else {
		viper.SetConfigType(configType)
	}

	for _, path := range configPaths {
		viper.AddConfigPath(path)
	}

	viper.AddConfigPath("/etc/irods-ext/") // path to look for the config file in
	viper.AddConfigPath("$HOME/.appname")  // call multiple times to add many search paths
	viper.AddConfigPath(".")               // optionally look for config in the working directory
	err := viper.ReadInConfig()            // Find and read the config file
	if err != nil {                        // Handle errors reading the config file
		panic(fmt.Errorf("fatal error config file: %w", err))
	}
	var C DrsConfig

	err = viper.Unmarshal(&C)
	if err != nil {
		panic(fmt.Errorf("unable to decode into struct, %v", err))
	}

	return &C, nil
}

func (d *DrsConfig) InitializeLogging() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	switch d.DrsLogLevel {
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
