package drs_support

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cyverse/go-irodsclient/irods/types"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

// DrsConfig Provides configuration for drs behaviors
type DrsConfig struct {
	DrsIdAvuValue                    string
	DrsAvuUnit                       string
	DrsLogLevel                      string //info, debug
	DrsListenPort                    int
	ServiceInfoSampleIntervalMinutes int
	ServiceInfoFilePath              string
	AccessMethods                    []string
	HTTPAccessBaseURL                string
	IRODSAccessHost                  string
	IRODSAccessPort                  int
	LocalAccessRootPath              string
	S3AccessEndpoint                 string
	IrodsHost                        string
	IrodsPort                        int
	IrodsZone                        string
	IrodsAdminUser                   string
	IrodsAdminPassword               string
	IrodsAdminPasswordFile           string
	IrodsPrimaryTestUser             string
	IrodsPrimaryTestPassword         string
	IrodsSecondaryTestUser           string
	IrodsSecondaryTestPassword       string
	IrodsAuthScheme                  string
	IrodsNegotiationPolicy           string
	IrodsDefaultResource             string
	OidcUrl                          string
	OidcClientId                     string
	OidcClientSecret                 string
	OidcClientSecretFile             string
	OidcRealm                        string
	OidcScope                        string
	OidcSkipTLSVerify                bool
	OidcInsecureSkipVerify           bool
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
		ClientUser:              cfg.IrodsAdminUser,
		ClientZone:              cfg.IrodsZone,
		ProxyUser:               cfg.IrodsAdminUser,
		ProxyZone:               cfg.IrodsZone,
		Password:                cfg.IrodsAdminPassword,
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
		"DrsIdAvuValue":                    {"DRS_DRS_ID_AVU_VALUE", "DRS_DRSIDAVUVALUE"},
		"DrsAvuUnit":                       {"DRS_DRS_AVU_UNIT", "DRS_DRSAVUUNIT"},
		"DrsLogLevel":                      {"DRS_DRS_LOG_LEVEL", "DRS_DRSLOGLEVEL"},
		"DrsListenPort":                    {"DRS_LISTEN_PORT", "DRS_DRSLISTENPORT"},
		"ServiceInfoSampleIntervalMinutes": {"DRS_SERVICE_INFO_SAMPLE_INTERVAL_MINUTES", "DRS_SERVICEINFOSAMPLEINTERVALMINUTES"},
		"ServiceInfoFilePath":              {"DRS_SERVICE_INFO_FILE_PATH", "DRS_SERVICEINFOFILEPATH"},
		"AccessMethods":                    {"DRS_ACCESS_METHODS", "DRS_ACCESSMETHODS"},
		"HTTPAccessBaseURL":                {"DRS_HTTP_ACCESS_BASE_URL", "DRS_HTTPACCESSBASEURL"},
		"IRODSAccessHost":                  {"DRS_IRODS_ACCESS_HOST", "DRS_IRODSACCESSHOST"},
		"IRODSAccessPort":                  {"DRS_IRODS_ACCESS_PORT", "DRS_IRODSACCESSPORT"},
		"LocalAccessRootPath":              {"DRS_LOCAL_ACCESS_ROOT_PATH", "DRS_LOCALACCESSROOTPATH"},
		"S3AccessEndpoint":                 {"DRS_S3_ACCESS_ENDPOINT", "DRS_S3ACCESSENDPOINT"},
		"IrodsHost":                        {"DRS_IRODS_HOST", "DRS_IRODSHOST"},
		"IrodsPort":                        {"DRS_IRODS_PORT", "DRS_IRODSPORT"},
		"IrodsZone":                        {"DRS_IRODS_ZONE", "DRS_IRODSZONE"},
		"IrodsAdminUser":                   {"DRS_IRODS_ADMIN_USER", "DRS_IRODSADMINUSER", "DRS_IRODS_DRS_ADMIN_USER", "DRS_IRODSDRSADMINUSER"},
		"IrodsAdminPassword":               {"DRS_IRODS_ADMIN_PASSWORD", "DRS_IRODSADMINPASSWORD", "DRS_IRODS_DRS_ADMIN_PASSWORD", "DRS_IRODSDRSADMINPASSWORD"},
		"IrodsAdminPasswordFile":           {"DRS_IRODS_ADMIN_PASSWORD_FILE", "DRS_IRODSADMINPASSWORDFILE", "DRS_IRODS_DRS_ADMIN_PASSWORD_FILE", "DRS_IRODSDRSADMINPASSWORDFILE"},
		"IrodsPrimaryTestUser":             {"DRS_IRODS_PRIMARY_TEST_USER", "DRS_IRODSPRIMARYTESTUSER"},
		"IrodsPrimaryTestPassword":         {"DRS_IRODS_PRIMARY_TEST_PASSWORD", "DRS_IRODSPRIMARYTESTPASSWORD"},
		"IrodsSecondaryTestUser":           {"DRS_IRODS_SECONDARY_TEST_USER", "DRS_IRODSSECONDARYTESTUSER"},
		"IrodsSecondaryTestPassword":       {"DRS_IRODS_SECONDARY_TEST_PASSWORD", "DRS_IRODSSECONDARYTESTPASSWORD"},
		"IrodsAuthScheme":                  {"DRS_IRODS_AUTH_SCHEME", "DRS_IRODSAUTHSCHEME"},
		"IrodsNegotiationPolicy":           {"DRS_IRODS_NEGOTIATION_POLICY", "DRS_IRODSNEGOTIATIONPOLICY"},
		"IrodsDefaultResource":             {"DRS_IRODS_DEFAULT_RESOURCE", "DRS_IRODSDEFAULTRESOURCE"},
		"OidcUrl":                          {"DRS_OIDC_URL", "DRS_OIDCURL"},
		"OidcClientId":                     {"DRS_OIDC_CLIENT_ID", "DRS_OIDCCLIENTID"},
		"OidcClientSecret":                 {"DRS_OIDC_CLIENT_SECRET", "DRS_OIDCCLIENTSECRET"},
		"OidcClientSecretFile":             {"DRS_OIDC_CLIENT_SECRET_FILE", "DRS_OIDCCLIENTSECRETFILE"},
		"OidcRealm":                        {"DRS_OIDC_REALM", "DRS_OIDCREALM"},
		"OidcScope":                        {"DRS_OIDC_SCOPE", "DRS_OIDCSCOPE"},
		"OidcSkipTLSVerify":                {"DRS_OIDC_SKIP_TLS_VERIFY", "DRS_OIDCSKIPTLSVERIFY"},
		"OidcInsecureSkipVerify":           {"DRS_OIDC_INSECURE_SKIP_VERIFY", "DRS_OIDCINSECURESKIPVERIFY"},
	}

	for key, envNames := range envBindings {
		bindingArgs := append([]string{key}, envNames...)
		if err := v.BindEnv(bindingArgs...); err != nil {
			return fmt.Errorf("failed to bind env for %s: %w", key, err)
		}
	}

	return nil
}

func resolveSecret(secret string, secretFile string, secretName string, configDir string) (string, error) {
	if secret != "" {
		return secret, nil
	}

	if secretFile == "" {
		return "", nil
	}

	if !filepath.IsAbs(secretFile) && configDir != "" {
		secretFile = filepath.Join(configDir, secretFile)
	}

	secretBytes, err := os.ReadFile(secretFile)
	if err != nil {
		return "", fmt.Errorf("failed to read %s file %q: %w", secretName, secretFile, err)
	}

	return strings.TrimSpace(string(secretBytes)), nil
}

func resolveConfigPath(configPath string, configDir string) string {
	configPath = strings.TrimSpace(configPath)

	if configPath == "" {
		return ""
	}

	if filepath.IsAbs(configPath) || configDir == "" {
		return configPath
	}

	return filepath.Join(configDir, configPath)
}

func resolveStringSliceConfig(v *viper.Viper, key string) []string {
	values := v.GetStringSlice(key)
	if len(values) == 0 {
		raw := strings.TrimSpace(v.GetString(key))
		if raw == "" {
			return nil
		}
		values = []string{raw}
	}

	resolved := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		for _, candidate := range strings.Split(value, ",") {
			normalized := strings.ToLower(strings.TrimSpace(candidate))
			if normalized == "" {
				continue
			}
			if _, ok := seen[normalized]; ok {
				continue
			}
			seen[normalized] = struct{}{}
			resolved = append(resolved, normalized)
		}
	}
	return resolved
}

// ReadDrsConfig reads the configuration for DRS behaviors in irods
// can take a number of paths that will be prefixed in the search path, or defaults
// may be accepted, blank params for name and type default to irods-drs.yaml
func ReadDrsConfig(configName string, configType string, configPaths []string) (*DrsConfig, error) {
	v := viper.New()

	configName = strings.TrimSpace(configName)
	configType = strings.TrimSpace(configType)

	if configFilePath := strings.TrimSpace(os.Getenv(ConfigFileEnvVar)); configFilePath != "" {
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
			path = strings.TrimSpace(path)
			if path == "" {
				continue
			}

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

	configDir := filepath.Dir(v.ConfigFileUsed())
	var C DrsConfig

	err = v.Unmarshal(&C)
	if err != nil {
		return nil, fmt.Errorf("unable to decode into struct: %w", err)
	}

	C.IrodsAdminPassword, err = resolveSecret(C.IrodsAdminPassword, C.IrodsAdminPasswordFile, "iRODS admin password", configDir)
	if err != nil {
		return nil, err
	}

	C.OidcClientSecret, err = resolveSecret(C.OidcClientSecret, C.OidcClientSecretFile, "OIDC client secret", configDir)
	if err != nil {
		return nil, err
	}

	C.ServiceInfoFilePath = resolveConfigPath(C.ServiceInfoFilePath, configDir)
	C.AccessMethods = resolveStringSliceConfig(v, "AccessMethods")
	C.HTTPAccessBaseURL = strings.TrimSpace(C.HTTPAccessBaseURL)
	C.IRODSAccessHost = strings.TrimSpace(C.IRODSAccessHost)
	C.LocalAccessRootPath = resolveConfigPath(C.LocalAccessRootPath, configDir)
	C.S3AccessEndpoint = strings.TrimSpace(C.S3AccessEndpoint)
	C.OidcSkipTLSVerify = C.OidcSkipTLSVerify || C.OidcInsecureSkipVerify
	C.OidcInsecureSkipVerify = C.OidcSkipTLSVerify

	if C.DrsListenPort == 0 {
		C.DrsListenPort = 8080
	}

	if C.ServiceInfoSampleIntervalMinutes <= 0 {
		C.ServiceInfoSampleIntervalMinutes = 5
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
