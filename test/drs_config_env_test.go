package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/michael-conway/irods-go-drs/drs-support"
)

func TestReadDrsConfigEnvOverride(t *testing.T) {
	t.Setenv("DRS_IRODS_HOST", "env-host")
	t.Setenv("DRS_OIDC_CLIENT_SECRET", "env-secret")
	t.Setenv("DRS_DRS_LOG_LEVEL", "debug")
	t.Setenv("DRS_LISTEN_PORT", "9090")
	t.Setenv("DRS_SERVICE_INFO_SAMPLE_INTERVAL_MINUTES", "11")
	t.Setenv("DRS_SERVICE_INFO_FILE_PATH", "/tmp/service-info.json")
	t.Setenv("DRS_OIDC_SKIP_TLS_VERIFY", "true")
	t.Setenv("DRS_ACCESS_METHODS", "http,irods,local,s3")
	t.Setenv("DRS_HTTP_ACCESS_BASE_URL", "https://download.example.org")
	t.Setenv("DRS_IRODS_ACCESS_HOST", "irods-access.example.org")
	t.Setenv("DRS_IRODS_ACCESS_PORT", "2247")
	t.Setenv("DRS_LOCAL_ACCESS_ROOT_PATH", "/srv/irods-mount")
	t.Setenv("DRS_S3_ACCESS_ENDPOINT", "https://s3.example.org")

	var confs = [1]string{"./resources/"}
	config, err := drs_support.ReadDrsConfig("drs-config1", "yaml", confs[:])
	if err != nil {
		t.Fatalf("error reading drs config: %s", err)
	}

	if config.IrodsHost != "env-host" {
		t.Fatalf("expected env override for IrodsHost, got %q", config.IrodsHost)
	}

	if config.OidcClientSecret != "env-secret" {
		t.Fatalf("expected env override for OidcClientSecret, got %q", config.OidcClientSecret)
	}

	if config.DrsLogLevel != "debug" {
		t.Fatalf("expected env override for DrsLogLevel, got %q", config.DrsLogLevel)
	}

	if config.DrsListenPort != 9090 {
		t.Fatalf("expected env override for DrsListenPort, got %d", config.DrsListenPort)
	}

	if config.ServiceInfoSampleIntervalMinutes != 11 {
		t.Fatalf("expected env override for ServiceInfoSampleIntervalMinutes, got %d", config.ServiceInfoSampleIntervalMinutes)
	}

	if config.ServiceInfoFilePath != "/tmp/service-info.json" {
		t.Fatalf("expected env override for ServiceInfoFilePath, got %q", config.ServiceInfoFilePath)
	}

	if !config.OidcSkipTLSVerify {
		t.Fatal("expected env override for OidcSkipTLSVerify")
	}

	if len(config.AccessMethods) != 4 || config.AccessMethods[0] != "http" || config.AccessMethods[3] != "s3" {
		t.Fatalf("expected env override for AccessMethods, got %+v", config.AccessMethods)
	}

	if config.HTTPAccessBaseURL != "https://download.example.org" {
		t.Fatalf("expected env override for HTTPAccessBaseURL, got %q", config.HTTPAccessBaseURL)
	}

	if config.IRODSAccessHost != "irods-access.example.org" {
		t.Fatalf("expected env override for IRODSAccessHost, got %q", config.IRODSAccessHost)
	}

	if config.IRODSAccessPort != 2247 {
		t.Fatalf("expected env override for IRODSAccessPort, got %d", config.IRODSAccessPort)
	}

	if config.LocalAccessRootPath != "/srv/irods-mount" {
		t.Fatalf("expected env override for LocalAccessRootPath, got %q", config.LocalAccessRootPath)
	}

	if config.S3AccessEndpoint != "https://s3.example.org" {
		t.Fatalf("expected env override for S3AccessEndpoint, got %q", config.S3AccessEndpoint)
	}
}

func TestReadDrsConfigMissingFileReturnsError(t *testing.T) {
	_, err := drs_support.ReadDrsConfig("does-not-exist", "yaml", []string{"./resources/"})
	if err == nil {
		t.Fatal("expected error for missing config file")
	}
}

func TestReadDrsConfigSecretFileSupport(t *testing.T) {
	var confs = [1]string{"./resources/"}
	config, err := drs_support.ReadDrsConfig("drs-config-secret-files", "yaml", confs[:])
	if err != nil {
		t.Fatalf("error reading drs config: %s", err)
	}

	if config.IrodsAdminPassword != "rods" {
		t.Fatalf("expected secret file value for IrodsAdminPassword, got %q", config.IrodsAdminPassword)
	}

	if config.OidcClientSecret != "test-oidc-secret" {
		t.Fatalf("expected secret file value for OidcClientSecret, got %q", config.OidcClientSecret)
	}
}

func TestReadDrsConfigConfigFileEnvOverride(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "custom-drs-config.yaml")
	configBody := "" +
		"DrsIdAvuValue: env-config\n" +
		"DrsListenPort: 9191\n" +
		"ServiceInfoSampleIntervalMinutes: 13\n" +
		"ServiceInfoFilePath: service-info.json\n" +
		"AccessMethods:\n" +
		"  - http\n" +
		"  - local\n" +
		"HTTPAccessBaseURL: https://download.example.org\n" +
		"LocalAccessRootPath: local-root\n" +
		"IrodsHost: env-file-host\n" +
		"IrodsPort: 1247\n" +
		"IrodsZone: tempZone\n" +
		"IrodsAdminUser: rods\n" +
		"IrodsAuthScheme: native\n" +
		"IrodsNegotiationPolicy: native\n"

	if err := os.WriteFile(configPath, []byte(configBody), 0600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv(drs_support.ConfigFileEnvVar, configPath)

	config, err := drs_support.ReadDrsConfig("does-not-exist", "yaml", []string{"./resources/"})
	if err != nil {
		t.Fatalf("error reading drs config with %s override: %s", drs_support.ConfigFileEnvVar, err)
	}

	if config.DrsIdAvuValue != "env-config" {
		t.Fatalf("expected config from %s override, got %q", drs_support.ConfigFileEnvVar, config.DrsIdAvuValue)
	}

	if config.IrodsHost != "env-file-host" {
		t.Fatalf("expected host from %s override, got %q", drs_support.ConfigFileEnvVar, config.IrodsHost)
	}

	if config.DrsListenPort != 9191 {
		t.Fatalf("expected listen port from %s override, got %d", drs_support.ConfigFileEnvVar, config.DrsListenPort)
	}

	if config.ServiceInfoSampleIntervalMinutes != 13 {
		t.Fatalf("expected service info sample interval from %s override, got %d", drs_support.ConfigFileEnvVar, config.ServiceInfoSampleIntervalMinutes)
	}

	expectedServiceInfoPath := filepath.Join(dir, "service-info.json")
	if config.ServiceInfoFilePath != expectedServiceInfoPath {
		t.Fatalf("expected service info path from %s override to resolve to %q, got %q", drs_support.ConfigFileEnvVar, expectedServiceInfoPath, config.ServiceInfoFilePath)
	}

	if len(config.AccessMethods) != 2 || config.AccessMethods[0] != "http" || config.AccessMethods[1] != "local" {
		t.Fatalf("expected configured access methods from %s override, got %+v", drs_support.ConfigFileEnvVar, config.AccessMethods)
	}

	expectedLocalRoot := filepath.Join(dir, "local-root")
	if config.LocalAccessRootPath != expectedLocalRoot {
		t.Fatalf("expected local access root path from %s override to resolve to %q, got %q", drs_support.ConfigFileEnvVar, expectedLocalRoot, config.LocalAccessRootPath)
	}
}

func TestReadDrsConfigTrimsWhitespaceFromInputs(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "custom-drs-config.yaml")
	configBody := "" +
		"DrsIdAvuValue: trimmed-config\n" +
		"DrsListenPort: 8181\n" +
		"ServiceInfoFilePath: service-info.json\n" +
		"IrodsHost: trimmed-host\n" +
		"IrodsPort: 1247\n" +
		"IrodsZone: tempZone\n" +
		"IrodsAdminUser: rods\n" +
		"IrodsAuthScheme: native\n" +
		"IrodsNegotiationPolicy: native\n"

	if err := os.WriteFile(configPath, []byte(configBody), 0600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv(drs_support.ConfigFileEnvVar, "  "+configPath+"  ")

	config, err := drs_support.ReadDrsConfig(" drs-config1 ", " yaml ", []string{"  " + dir + "  ", "   "})
	if err != nil {
		t.Fatalf("error reading drs config with whitespace-padded inputs: %s", err)
	}

	if config.DrsIdAvuValue != "trimmed-config" {
		t.Fatalf("expected config file override after trimming, got %q", config.DrsIdAvuValue)
	}

	if config.IrodsHost != "trimmed-host" {
		t.Fatalf("expected trimmed host from config file override, got %q", config.IrodsHost)
	}

	expectedServiceInfoPath := filepath.Join(dir, "service-info.json")
	if config.ServiceInfoFilePath != expectedServiceInfoPath {
		t.Fatalf("expected trimmed service info path to resolve to %q, got %q", expectedServiceInfoPath, config.ServiceInfoFilePath)
	}
}
