package drs_support

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/cyverse/go-irodsclient/irods/types"
	"github.com/go-viper/mapstructure/v2"
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
	IrodsAccessMethodSupported       bool
	FileAccessMethodSupported        bool
	HttpsAccessMethodSupported       bool
	HttpsAccessImplementation        string
	HttpsAccessMethodBaseURL         string
	HttpsAccessUseTicket             bool
	DefaultTicketLifetimeMinutes     int
	DefaultTicketUseLimit            int
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
	ResourceAffinity                 []ResourceAffinityEntry
	OidcUrl                          string
	OidcClientId                     string
	OidcClientSecret                 string
	OidcClientSecretFile             string
	OidcRealm                        string
	OidcScope                        string
	OidcSkipTLSVerify                bool
	OidcInsecureSkipVerify           bool
}

type ResourceAffinityEntry struct {
	Host      string   `mapstructure:"Host"`
	Resources []string `mapstructure:"Resources"`
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
		"IrodsAccessMethodSupported":       {"DRS_IRODS_ACCESS_METHOD_SUPPORTED", "DRS_IRODSACCESSMETHODSUPPORTED"},
		"FileAccessMethodSupported":        {"DRS_FILE_ACCESS_METHOD_SUPPORTED", "DRS_FILEACCESSMETHODSUPPORTED"},
		"HttpsAccessMethodSupported":       {"DRS_HTTPS_ACCESS_METHOD_SUPPORTED", "DRS_HTTPSACCESSMETHODSUPPORTED"},
		"HttpsAccessImplementation":        {"DRS_HTTPS_ACCESS_IMPLEMENTATION", "DRS_HTTPSACCESSIMPLEMENTATION"},
		"HttpsAccessMethodBaseURL":         {"DRS_HTTPS_ACCESS_METHOD_BASE_URL", "DRS_HTTPSACCESSMETHODBASEURL"},
		"HttpsAccessUseTicket":             {"DRS_HTTPS_ACCESS_USE_TICKET", "DRS_HTTPSACCESSUSETICKET"},
		"DefaultTicketLifetimeMinutes":     {"DRS_DEFAULT_TICKET_LIFETIME_MINUTES", "DRS_DEFAULTTICKETLIFETIMEMINUTES"},
		"DefaultTicketUseLimit":            {"DRS_DEFAULT_TICKET_USE_LIMIT", "DRS_DEFAULTTICKETUSELIMIT"},
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

	err = v.Unmarshal(&C, func(decoderConfig *mapstructure.DecoderConfig) {
		decoderConfig.DecodeHook = mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToSliceHookFunc(","),
			func(from reflect.Type, to reflect.Type, data any) (any, error) {
				resourceAffinityType := reflect.TypeOf([]ResourceAffinityEntry{})
				if to != resourceAffinityType {
					return data, nil
				}

				switch value := data.(type) {
				case []string:
					normalized := normalizeStringSlice(value)
					if len(normalized) == 0 {
						return []ResourceAffinityEntry{}, nil
					}
					return []ResourceAffinityEntry{{
						Resources: normalized,
					}}, nil
				case string:
					normalized := normalizeStringSlice(strings.Split(value, ","))
					if len(normalized) == 0 {
						return []ResourceAffinityEntry{}, nil
					}
					return []ResourceAffinityEntry{{
						Resources: normalized,
					}}, nil
				case []any:
					legacyValues := make([]string, 0, len(value))
					for _, raw := range value {
						switch cast := raw.(type) {
						case string:
							legacyValues = append(legacyValues, cast)
						default:
							return data, nil
						}
					}

					normalized := normalizeStringSlice(legacyValues)
					if len(normalized) == 0 {
						return []ResourceAffinityEntry{}, nil
					}
					return []ResourceAffinityEntry{{
						Resources: normalized,
					}}, nil
				default:
					return data, nil
				}
			},
			func(from reflect.Type, to reflect.Type, data any) (any, error) {
				if to.Kind() != reflect.Slice || to.Elem().Kind() != reflect.String {
					return data, nil
				}

				values, ok := data.([]string)
				if !ok {
					return data, nil
				}

				normalized := make([]string, 0, len(values))
				normalized = append(normalized, normalizeStringSlice(values)...)
				return normalized, nil
			},
		)
	})
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
	C.HttpsAccessImplementation = strings.ToLower(strings.TrimSpace(C.HttpsAccessImplementation))
	C.HttpsAccessMethodBaseURL = strings.TrimSpace(C.HttpsAccessMethodBaseURL)
	C.IRODSAccessHost = strings.TrimSpace(C.IRODSAccessHost)
	C.LocalAccessRootPath = resolveConfigPath(C.LocalAccessRootPath, configDir)
	C.S3AccessEndpoint = strings.TrimSpace(C.S3AccessEndpoint)
	C.ResourceAffinity = normalizeResourceAffinities(C.ResourceAffinity)
	applyResourceAffinityEnvOverride(&C)
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

func normalizeStringSlice(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		normalized = append(normalized, value)
	}
	return normalized
}

func normalizeResourceAffinities(entries []ResourceAffinityEntry) []ResourceAffinityEntry {
	if len(entries) == 0 {
		return nil
	}

	normalized := make([]ResourceAffinityEntry, 0, len(entries))
	for _, entry := range entries {
		host := strings.TrimSpace(entry.Host)
		resources := normalizeStringSlice(entry.Resources)
		if host == "" && len(resources) == 0 {
			continue
		}

		normalized = append(normalized, ResourceAffinityEntry{
			Host:      host,
			Resources: resources,
		})
	}

	if len(normalized) == 0 {
		return nil
	}

	return normalized
}

func applyResourceAffinityEnvOverride(cfg *DrsConfig) {
	if cfg == nil {
		return
	}

	raw := strings.TrimSpace(os.Getenv("DRS_RESOURCE_AFFINITY"))
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv("DRS_RESOURCEAFFINITY"))
	}
	if raw == "" {
		return
	}

	resources := normalizeStringSlice(strings.Split(raw, ","))
	if len(resources) == 0 {
		cfg.ResourceAffinity = nil
		return
	}

	cfg.ResourceAffinity = []ResourceAffinityEntry{{
		Host:      strings.TrimSpace(cfg.HttpsAccessMethodBaseURL),
		Resources: resources,
	}}
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
