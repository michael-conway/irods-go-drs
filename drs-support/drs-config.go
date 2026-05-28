package drs_support

import (
	"fmt"
	"log/slog"
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
	HTTPReadTimeoutSeconds           int
	HTTPReadHeaderTimeoutSeconds     int
	HTTPWriteTimeoutSeconds          int
	HTTPIdleTimeoutSeconds           int
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
	S3AccessMethodSupported          bool
	S3AccessMethodBaseURL            string
	S3ResourceAffinity               []ResourceAffinityEntry
	IrodsHost                        string
	IrodsPort                        int
	IrodsZone                        string
	IrodsAdminUser                   string
	IrodsAdminPassword               string
	IrodsAdminPasswordFile           string
	IrodsAdminLoginType              string
	IrodsPrimaryTestUser             string
	IrodsPrimaryTestPassword         string
	IrodsSecondaryTestUser           string
	IrodsSecondaryTestPassword       string
	IrodsAuthScheme                  string
	IrodsNegotiationPolicy           string
	IrodsSSLConfig                   IrodsSSLConfig
	IrodsDefaultResource             string
	HttpsResourceAffinity            []ResourceAffinityEntry
	OidcUrl                          string
	OidcClientId                     string
	OidcClientSecret                 string
	OidcClientSecretFile             string
	OidcRealm                        string
	OidcScope                        string
	OidcSkipTLSVerify                bool
	OidcInsecureSkipVerify           bool
}

type IrodsSSLConfig struct {
	CACertificateFile       string
	CACertificatePath       string
	EncryptionKeySize       int
	EncryptionAlgorithm     string
	EncryptionSaltSize      int
	EncryptionNumHashRounds int
	VerifyServer            string
	DHParamsFile            string
	ServerName              string
}

type ResourceAffinityEntry struct {
	Host      string   `mapstructure:"Host"`
	Resources []string `mapstructure:"Resources"`
}

func NormalizeIRODSNegotiationPolicy(policy string) string {
	policy = strings.TrimSpace(policy)
	switch policy {
	case string(types.CSNegotiationPolicyRequestTCP), string(types.CSNegotiationPolicyRequestSSL), string(types.CSNegotiationPolicyRequestDontCare):
		return policy
	default:
		slog.Warn(
			"invalid iRODS negotiation policy; defaulting to CS_NEG_DONT_CARE",
			"configured_policy", policy,
			"default_policy", string(types.CSNegotiationPolicyRequestDontCare),
		)
		return string(types.CSNegotiationPolicyRequestDontCare)
	}
}

func (cfg *DrsConfig) AdminAuthScheme() types.AuthScheme {
	adminLoginType := strings.TrimSpace(cfg.IrodsAdminLoginType)
	if adminLoginType == "" {
		adminLoginType = strings.TrimSpace(cfg.IrodsAuthScheme)
	}

	return types.GetAuthScheme(adminLoginType)
}

func (cfg *DrsConfig) RequestAuthScheme() types.AuthScheme {
	return types.GetAuthScheme(cfg.IrodsAuthScheme)
}

func (cfg *DrsConfig) ToIRODSSSLConfig() *types.IRODSSSLConfig {
	sslConfig := cfg.IrodsSSLConfig

	verifyServerName := strings.TrimSpace(sslConfig.VerifyServer)
	if verifyServerName == "" {
		verifyServerName = defaultIRODSSSLVerifyServer
	}

	verifyServer, err := types.GetSSLVerifyServer(verifyServerName)
	if err != nil {
		slog.Warn(
			"invalid iRODS SSL verify server; defaulting to go-irodsclient value",
			"configured_verify_server", verifyServerName,
			"default_verify_server", defaultIRODSSSLVerifyServer,
		)
		verifyServer, _ = types.GetSSLVerifyServer(defaultIRODSSSLVerifyServer)
	}

	encryptionKeySize := sslConfig.EncryptionKeySize
	if encryptionKeySize <= 0 {
		encryptionKeySize = defaultIRODSEncryptionKeySize
	}

	encryptionAlgorithm := strings.TrimSpace(sslConfig.EncryptionAlgorithm)
	if encryptionAlgorithm == "" {
		encryptionAlgorithm = defaultIRODSEncryptionAlgorithm
	}

	encryptionSaltSize := sslConfig.EncryptionSaltSize
	if encryptionSaltSize <= 0 {
		encryptionSaltSize = defaultIRODSEncryptionSaltSize
	}

	encryptionNumHashRounds := sslConfig.EncryptionNumHashRounds
	if encryptionNumHashRounds <= 0 {
		encryptionNumHashRounds = defaultIRODSEncryptionNumHashRounds
	}

	return &types.IRODSSSLConfig{
		CACertificateFile:       strings.TrimSpace(sslConfig.CACertificateFile),
		CACertificatePath:       strings.TrimSpace(sslConfig.CACertificatePath),
		EncryptionKeySize:       encryptionKeySize,
		EncryptionAlgorithm:     encryptionAlgorithm,
		EncryptionSaltSize:      encryptionSaltSize,
		EncryptionNumHashRounds: encryptionNumHashRounds,
		VerifyServer:            verifyServer,
		DHParamsFile:            strings.TrimSpace(sslConfig.DHParamsFile),
		ServerName:              strings.TrimSpace(sslConfig.ServerName),
	}
}

func (cfg *DrsConfig) ApplyIRODSConnectionConfig(account *types.IRODSAccount) *types.IRODSAccount {
	if account == nil {
		return nil
	}

	normalizedPolicy := NormalizeIRODSNegotiationPolicy(cfg.IrodsNegotiationPolicy)
	negotiationPolicy := types.GetCSNegotiationPolicyRequest(normalizedPolicy)
	requireNegotiation := negotiationPolicy == types.CSNegotiationPolicyRequestSSL

	account.SetCSNegotiation(requireNegotiation, negotiationPolicy)
	account.SetSSLConfiguration(cfg.ToIRODSSSLConfig())
	account.FixAuthConfiguration()

	return account
}

func (cfg *DrsConfig) ToIrodsAccount() types.IRODSAccount {
	account := types.IRODSAccount{
		AuthenticationScheme: cfg.AdminAuthScheme(),
		Host:                 cfg.IrodsHost,
		Port:                 cfg.IrodsPort,
		ClientUser:           cfg.IrodsAdminUser,
		ClientZone:           cfg.IrodsZone,
		ProxyUser:            cfg.IrodsAdminUser,
		ProxyZone:            cfg.IrodsZone,
		Password:             cfg.IrodsAdminPassword,
		DefaultResource:      cfg.IrodsDefaultResource,
	}

	cfg.ApplyIRODSConnectionConfig(&account)

	return account
}

const DefaultConfigName = "drs-config"
const DefaultConfigType = "yaml"
const ConfigFileEnvVar = "DRS_CONFIG_FILE"
const defaultIRODSEncryptionAlgorithm = "AES-256-CBC"
const defaultIRODSEncryptionKeySize = 32
const defaultIRODSEncryptionSaltSize = 8
const defaultIRODSEncryptionNumHashRounds = 16
const defaultIRODSSSLVerifyServer = "hostname"

func bindEnvVars(v *viper.Viper) error {
	envBindings := map[string][]string{
		"DrsIdAvuValue":                          {"DRS_DRS_ID_AVU_VALUE", "DRS_DRSIDAVUVALUE"},
		"DrsAvuUnit":                             {"DRS_DRS_AVU_UNIT", "DRS_DRSAVUUNIT"},
		"DrsLogLevel":                            {"DRS_DRS_LOG_LEVEL", "DRS_DRSLOGLEVEL"},
		"DrsListenPort":                          {"DRS_LISTEN_PORT", "DRS_DRSLISTENPORT"},
		"HTTPReadTimeoutSeconds":                 {"DRS_HTTP_READ_TIMEOUT_SECONDS", "DRS_HTTPREADTIMEOUTSECONDS"},
		"HTTPReadHeaderTimeoutSeconds":           {"DRS_HTTP_READ_HEADER_TIMEOUT_SECONDS", "DRS_HTTPREADHEADERTIMEOUTSECONDS"},
		"HTTPWriteTimeoutSeconds":                {"DRS_HTTP_WRITE_TIMEOUT_SECONDS", "DRS_HTTPWRITETIMEOUTSECONDS"},
		"HTTPIdleTimeoutSeconds":                 {"DRS_HTTP_IDLE_TIMEOUT_SECONDS", "DRS_HTTPIDLETIMEOUTSECONDS"},
		"ServiceInfoSampleIntervalMinutes":       {"DRS_SERVICE_INFO_SAMPLE_INTERVAL_MINUTES", "DRS_SERVICEINFOSAMPLEINTERVALMINUTES"},
		"ServiceInfoFilePath":                    {"DRS_SERVICE_INFO_FILE_PATH", "DRS_SERVICEINFOFILEPATH"},
		"IrodsAccessMethodSupported":             {"DRS_IRODS_ACCESS_METHOD_SUPPORTED", "DRS_IRODSACCESSMETHODSUPPORTED"},
		"FileAccessMethodSupported":              {"DRS_FILE_ACCESS_METHOD_SUPPORTED", "DRS_FILEACCESSMETHODSUPPORTED"},
		"HttpsAccessMethodSupported":             {"DRS_HTTPS_ACCESS_METHOD_SUPPORTED", "DRS_HTTPSACCESSMETHODSUPPORTED"},
		"HttpsAccessImplementation":              {"DRS_HTTPS_ACCESS_IMPLEMENTATION", "DRS_HTTPSACCESSIMPLEMENTATION"},
		"HttpsAccessMethodBaseURL":               {"DRS_HTTPS_ACCESS_METHOD_BASE_URL", "DRS_HTTPSACCESSMETHODBASEURL"},
		"HttpsAccessUseTicket":                   {"DRS_HTTPS_ACCESS_USE_TICKET", "DRS_HTTPSACCESSUSETICKET"},
		"DefaultTicketLifetimeMinutes":           {"DRS_DEFAULT_TICKET_LIFETIME_MINUTES", "DRS_DEFAULTTICKETLIFETIMEMINUTES"},
		"DefaultTicketUseLimit":                  {"DRS_DEFAULT_TICKET_USE_LIMIT", "DRS_DEFAULTTICKETUSELIMIT"},
		"IRODSAccessHost":                        {"DRS_IRODS_ACCESS_HOST", "DRS_IRODSACCESSHOST"},
		"IRODSAccessPort":                        {"DRS_IRODS_ACCESS_PORT", "DRS_IRODSACCESSPORT"},
		"LocalAccessRootPath":                    {"DRS_LOCAL_ACCESS_ROOT_PATH", "DRS_LOCALACCESSROOTPATH"},
		"S3AccessMethodSupported":                {"DRS_S3_ACCESS_METHOD_SUPPORTED", "DRS_S3ACCESSMETHODSUPPORTED"},
		"S3AccessMethodBaseURL":                  {"DRS_S3_ACCESS_METHOD_BASE_URL", "DRS_S3ACCESSMETHODBASEURL"},
		"IrodsHost":                              {"DRS_IRODS_HOST", "DRS_IRODSHOST"},
		"IrodsPort":                              {"DRS_IRODS_PORT", "DRS_IRODSPORT"},
		"IrodsZone":                              {"DRS_IRODS_ZONE", "DRS_IRODSZONE"},
		"IrodsAdminUser":                         {"DRS_IRODS_ADMIN_USER", "DRS_IRODSADMINUSER", "DRS_IRODS_DRS_ADMIN_USER", "DRS_IRODSDRSADMINUSER"},
		"IrodsAdminPassword":                     {"DRS_IRODS_ADMIN_PASSWORD", "DRS_IRODSADMINPASSWORD", "DRS_IRODS_DRS_ADMIN_PASSWORD", "DRS_IRODSDRSADMINPASSWORD"},
		"IrodsAdminPasswordFile":                 {"DRS_IRODS_ADMIN_PASSWORD_FILE", "DRS_IRODSADMINPASSWORDFILE", "DRS_IRODS_DRS_ADMIN_PASSWORD_FILE", "DRS_IRODSDRSADMINPASSWORDFILE"},
		"IrodsAdminLoginType":                    {"DRS_IRODS_ADMIN_LOGIN_TYPE", "DRS_IRODS_ADMIN_AUTH_SCHEME", "DRS_IRODSADMINLOGINTYPE"},
		"IrodsPrimaryTestUser":                   {"DRS_IRODS_PRIMARY_TEST_USER", "DRS_IRODSPRIMARYTESTUSER"},
		"IrodsPrimaryTestPassword":               {"DRS_IRODS_PRIMARY_TEST_PASSWORD", "DRS_IRODSPRIMARYTESTPASSWORD"},
		"IrodsSecondaryTestUser":                 {"DRS_IRODS_SECONDARY_TEST_USER", "DRS_IRODSSECONDARYTESTUSER"},
		"IrodsSecondaryTestPassword":             {"DRS_IRODS_SECONDARY_TEST_PASSWORD", "DRS_IRODSSECONDARYTESTPASSWORD"},
		"IrodsAuthScheme":                        {"DRS_IRODS_AUTH_SCHEME", "DRS_IRODSAUTHSCHEME"},
		"IrodsNegotiationPolicy":                 {"DRS_IRODS_NEGOTIATION_POLICY", "DRS_IRODSNEGOTIATIONPOLICY"},
		"IrodsSSLConfig.CACertificateFile":       {"DRS_IRODS_SSL_CA_CERTIFICATE_FILE", "DRS_IRODSSSLCACERTIFICATEFILE"},
		"IrodsSSLConfig.CACertificatePath":       {"DRS_IRODS_SSL_CA_CERTIFICATE_PATH", "DRS_IRODSSSLCACERTIFICATEPATH"},
		"IrodsSSLConfig.EncryptionKeySize":       {"DRS_IRODS_ENCRYPTION_KEY_SIZE", "DRS_IRODSENCRYPTIONKEYSIZE"},
		"IrodsSSLConfig.EncryptionAlgorithm":     {"DRS_IRODS_ENCRYPTION_ALGORITHM", "DRS_IRODSENCRYPTIONALGORITHM"},
		"IrodsSSLConfig.EncryptionSaltSize":      {"DRS_IRODS_ENCRYPTION_SALT_SIZE", "DRS_IRODSENCRYPTIONSALTSIZE"},
		"IrodsSSLConfig.EncryptionNumHashRounds": {"DRS_IRODS_ENCRYPTION_NUM_HASH_ROUNDS", "DRS_IRODSENCRYPTIONNUMHASHROUNDS"},
		"IrodsSSLConfig.VerifyServer":            {"DRS_IRODS_SSL_VERIFY_SERVER", "DRS_IRODSSSLVERIFYSERVER"},
		"IrodsSSLConfig.DHParamsFile":            {"DRS_IRODS_SSL_DH_PARAMS_FILE", "DRS_IRODSSSLDHPARAMSFILE"},
		"IrodsSSLConfig.ServerName":              {"DRS_IRODS_SSL_SERVER_NAME", "DRS_IRODSSSLSERVERNAME"},
		"IrodsDefaultResource":                   {"DRS_IRODS_DEFAULT_RESOURCE", "DRS_IRODSDEFAULTRESOURCE"},
		"OidcUrl":                                {"DRS_OIDC_URL", "DRS_OIDCURL"},
		"OidcClientId":                           {"DRS_OIDC_CLIENT_ID", "DRS_OIDCCLIENTID"},
		"OidcClientSecret":                       {"DRS_OIDC_CLIENT_SECRET", "DRS_OIDCCLIENTSECRET"},
		"OidcClientSecretFile":                   {"DRS_OIDC_CLIENT_SECRET_FILE", "DRS_OIDCCLIENTSECRETFILE"},
		"OidcRealm":                              {"DRS_OIDC_REALM", "DRS_OIDCREALM"},
		"OidcScope":                              {"DRS_OIDC_SCOPE", "DRS_OIDCSCOPE"},
		"OidcSkipTLSVerify":                      {"DRS_OIDC_SKIP_TLS_VERIFY", "DRS_OIDCSKIPTLSVERIFY"},
		"OidcInsecureSkipVerify":                 {"DRS_OIDC_INSECURE_SKIP_VERIFY", "DRS_OIDCINSECURESKIPVERIFY"},
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

	C.IrodsNegotiationPolicy = NormalizeIRODSNegotiationPolicy(C.IrodsNegotiationPolicy)
	C.ServiceInfoFilePath = resolveConfigPath(C.ServiceInfoFilePath, configDir)
	C.HttpsAccessImplementation = strings.ToLower(strings.TrimSpace(C.HttpsAccessImplementation))
	C.HttpsAccessMethodBaseURL = strings.TrimSpace(C.HttpsAccessMethodBaseURL)
	C.IRODSAccessHost = strings.TrimSpace(C.IRODSAccessHost)
	C.LocalAccessRootPath = resolveConfigPath(C.LocalAccessRootPath, configDir)
	C.S3AccessMethodBaseURL = strings.TrimSpace(C.S3AccessMethodBaseURL)
	C.S3ResourceAffinity = normalizeResourceAffinities(C.S3ResourceAffinity)
	C.HttpsResourceAffinity = normalizeResourceAffinities(C.HttpsResourceAffinity)
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
