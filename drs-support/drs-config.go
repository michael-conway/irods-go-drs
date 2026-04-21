package drs_support

import (
	"fmt"
	"os"
	"strings"

	"github.com/cyverse/go-irodsclient/irods/types"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

// DrsConfig Provides configuration for drs behaviors
type DrsConfig struct {
	DrsIdAvuValue          string
	DrsAvuUnit             string
	DrsLogLevel            string //info, debug
	IrodsHost              string
	IrodsPort              int
	IrodsZone              string
	IrodsDrsAdminUser      string
	IrodsDrsAdminPassword  string
	IrodsDrsAdminPasswordFile string
	IrodsAuthScheme        string
	IrodsNegotiationPolicy string
	IrodsDefaultResource   string
	OidcUrl                string
	OidcClientId           string
	OidcClientSecret       string
	OidcClientSecretFile   string
	OidcRealm              string
	OidcScope              string
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
const ConfigFileEnvVar = "DRS_CONFIG_FILE"

func bindEnvVars(v *viper.Viper) error {
	envBindings := map[string][]string{
		"DrsIdAvuValue":          {"DRS_DRS_ID_AVU_VALUE", "DRS_DRSIDAVUVALUE"},
		"DrsAvuUnit":             {"DRS_DRS_AVU_UNIT", "DRS_DRSAVUUNIT"},
		"DrsLogLevel":            {"DRS_DRS_LOG_LEVEL", "DRS_DRSLOGLEVEL"},
		"IrodsHost":              {"DRS_IRODS_HOST", "DRS_IRODSHOST"},
		"IrodsPort":              {"DRS_IRODS_PORT", "DRS_IRODSPORT"},
		"IrodsZone":              {"DRS_IRODS_ZONE", "DRS_IRODSZONE"},
		"IrodsDrsAdminUser":      {"DRS_IRODS_DRS_ADMIN_USER", "DRS_IRODSDRSADMINUSER"},
		"IrodsDrsAdminPassword":  {"DRS_IRODS_DRS_ADMIN_PASSWORD", "DRS_IRODSDRSADMINPASSWORD"},
		"IrodsDrsAdminPasswordFile": {"DRS_IRODS_DRS_ADMIN_PASSWORD_FILE", "DRS_IRODSDRSADMINPASSWORDFILE"},
		"IrodsAuthScheme":        {"DRS_IRODS_AUTH_SCHEME", "DRS_IRODSAUTHSCHEME"},
		"IrodsNegotiationPolicy": {"DRS_IRODS_NEGOTIATION_POLICY", "DRS_IRODSNEGOTIATIONPOLICY"},
		"IrodsDefaultResource":   {"DRS_IRODS_DEFAULT_RESOURCE", "DRS_IRODSDEFAULTRESOURCE"},
		"OidcUrl":                {"DRS_OIDC_URL", "DRS_OIDCURL"},
		"OidcClientId":           {"DRS_OIDC_CLIENT_ID", "DRS_OIDCCLIENTID"},
		"OidcClientSecret":       {"DRS_OIDC_CLIENT_SECRET", "DRS_OIDCCLIENTSECRET"},
		"OidcClientSecretFile":   {"DRS_OIDC_CLIENT_SECRET_FILE", "DRS_OIDCCLIENTSECRETFILE"},
		"OidcRealm":              {"DRS_OIDC_REALM", "DRS_OIDCREALM"},
		"OidcScope":              {"DRS_OIDC_SCOPE", "DRS_OIDCSCOPE"},
	}

	for key, envNames := range envBindings {
		bindingArgs := append([]string{key}, envNames...)
		if err := v.BindEnv(bindingArgs...); err != nil {
			return fmt.Errorf("failed to bind env for %s: %w", key, err)
		}
	}

	return nil
}

func resolveSecret(secret string, secretFile string, secretName string) (string, error) {
	if secret != "" {
		return secret, nil
	}

	if secretFile == "" {
		return "", nil
	}

	secretBytes, err := os.ReadFile(secretFile)
	if err != nil {
		return "", fmt.Errorf("failed to read %s file %q: %w", secretName, secretFile, err)
	}

	return strings.TrimSpace(string(secretBytes)), nil
}

// ReadDrsConfig reads the configuration for DRS behaviors in irods
// can take a number of paths that will be prefixed in the search path, or defaults
// may be accepted, blank params for name and type default to irods-drs.yaml
func ReadDrsConfig(configName string, configType string, configPaths []string) (*DrsConfig, error) {
	v := viper.New()

	if configFilePath := os.Getenv(ConfigFileEnvVar); configFilePath != "" {
		v.SetConfigFile(configFilePath)
	} else {
		if configName == "" {
			v.SetConfigName(DefaultConfigName) // name of config file (without extension)
		} else {
			v.SetConfigName(configName)
		}

		if configType == "" {
			v.SetConfigType(DefaultConfigType) // REQUIRED if the config file does not have the extension in the name
		} else {
			v.SetConfigType(configType)
		}

		for _, path := range configPaths {
			v.AddConfigPath(path)
		}

		v.AddConfigPath("/etc/irods-ext/")  // path to look for the config file in
		v.AddConfigPath("$HOME/.irods-drs") // call multiple times to add many search paths
		v.AddConfigPath(".")                // optionally look for config in the working directory
	}

	if err := bindEnvVars(v); err != nil {
		return nil, err
	}

	err := v.ReadInConfig() // Find and read the config file
	if err != nil {         // Handle errors reading the config file
		return nil, fmt.Errorf("fatal error config file: %w", err)
	}
	var C DrsConfig

	err = v.Unmarshal(&C)
	if err != nil {
		return nil, fmt.Errorf("unable to decode into struct: %w", err)
	}

	C.IrodsDrsAdminPassword, err = resolveSecret(C.IrodsDrsAdminPassword, C.IrodsDrsAdminPasswordFile, "iRODS admin password")
	if err != nil {
		return nil, err
	}

	C.OidcClientSecret, err = resolveSecret(C.OidcClientSecret, C.OidcClientSecretFile, "OIDC client secret")
	if err != nil {
		return nil, err
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
