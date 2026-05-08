package test

import (
	irodstypes "github.com/cyverse/go-irodsclient/irods/types"
	"github.com/michael-conway/irods-go-drs/drs-support"
	"testing"
)

func TestReadDrsConfig(t *testing.T) {
	var confs = [1]string{"./resources/"}
	actual, err := drs_support.ReadDrsConfig("drs-config1", "yaml", confs[:])
	if err != nil {
		t.Errorf("error reading drs config: %s", err)
	}
	if actual.DrsIdAvuValue != "drs-id" {
		t.Fail()
	}
	if actual.DrsListenPort != 8080 {
		t.Fatalf("expected default listen port from config to be 8080, got %d", actual.DrsListenPort)
	}
	if actual.ServiceInfoSampleIntervalMinutes != 7 {
		t.Fatalf("expected service info sample interval from config to be 7, got %d", actual.ServiceInfoSampleIntervalMinutes)
	}
	if !actual.HttpsAccessMethodSupported {
		t.Fatal("expected https access method to be enabled from config")
	}
	if actual.HttpsAccessImplementation != "irods-go-rest" {
		t.Fatalf("expected https access implementation from config, got %q", actual.HttpsAccessImplementation)
	}
	if actual.HttpsAccessMethodBaseURL != "/api/v1/path/contents?irods_path=" {
		t.Fatalf("expected https access method base URL from config, got %q", actual.HttpsAccessMethodBaseURL)
	}
	if !actual.HttpsAccessUseTicket {
		t.Fatal("expected https access ticket mode from config")
	}
	if !actual.S3AccessMethodSupported {
		t.Fatal("expected s3 access method to be enabled from config")
	}
	if actual.S3AccessMethodBaseURL != "s3://" {
		t.Fatalf("expected s3 access method base URL from config, got %q", actual.S3AccessMethodBaseURL)
	}
	if actual.DefaultTicketLifetimeMinutes != 720 {
		t.Fatalf("expected default ticket lifetime from config to be 720, got %d", actual.DefaultTicketLifetimeMinutes)
	}
	if actual.DefaultTicketUseLimit != 50 {
		t.Fatalf("expected default ticket use limit from config to be 50, got %d", actual.DefaultTicketUseLimit)
	}
	if len(actual.HttpsResourceAffinity) != 2 || actual.HttpsResourceAffinity[0].Host != "https://download.example.org" || actual.HttpsResourceAffinity[1].Host != "https://download-alt.example.org" {
		t.Fatalf("expected https resource affinity from config, got %+v", actual.HttpsResourceAffinity)
	}
	if len(actual.HttpsResourceAffinity[0].Resources) != 1 || actual.HttpsResourceAffinity[0].Resources[0] != "demoResc" {
		t.Fatalf("expected first https resource affinity resources from config, got %+v", actual.HttpsResourceAffinity)
	}
	if len(actual.HttpsResourceAffinity[1].Resources) != 0 {
		t.Fatalf("expected second https resource affinity resources from config, got %+v", actual.HttpsResourceAffinity)
	}
	if len(actual.S3ResourceAffinity) != 2 || actual.S3ResourceAffinity[0].Host != "s3://download.example.org" || actual.S3ResourceAffinity[1].Host != "s3://download-alt.example.org" {
		t.Fatalf("expected s3 resource affinity from config, got %+v", actual.S3ResourceAffinity)
	}
	if len(actual.S3ResourceAffinity[0].Resources) != 1 || actual.S3ResourceAffinity[0].Resources[0] != "demoResc" {
		t.Fatalf("expected first s3 resource affinity resources from config, got %+v", actual.S3ResourceAffinity)
	}
	if len(actual.S3ResourceAffinity[1].Resources) != 0 {
		t.Fatalf("expected second s3 resource affinity resources from config, got %+v", actual.S3ResourceAffinity)
	}
}

func TestSetLogLevel(t *testing.T) {
	var confs = [1]string{"./resources/"}
	config, err := drs_support.ReadDrsConfig("drs-config1", "yaml", confs[:])
	if err != nil {
		t.Errorf("error reading drs config: %s", err)
	}
	config.InitializeLogging()

}

func TestConfigToIrodsAccount(t *testing.T) {
	var confs = [1]string{"./resources/"}
	config, err := drs_support.ReadDrsConfig("drs-config1", "yaml", confs[:])
	if err != nil {
		t.Errorf("error reading drs config: %s", err)
	}
	actual := config.ToIrodsAccount()
	if actual.ClientUser != config.IrodsAdminUser {
		t.Fail()
	}

}

func TestConfigSeparatesAdminAndRequestAuthSchemes(t *testing.T) {
	cfg := &drs_support.DrsConfig{
		IrodsHost:              "irods.example.org",
		IrodsPort:              1247,
		IrodsZone:              "tempZone",
		IrodsAdminUser:         "rods",
		IrodsAdminPassword:     "secret",
		IrodsAdminLoginType:    "native",
		IrodsAuthScheme:        "pam",
		IrodsNegotiationPolicy: "CS_NEG_DONT_CARE",
		IrodsDefaultResource:   "demoResc",
	}

	adminAccount := cfg.ToIrodsAccount()
	if adminAccount.AuthenticationScheme != irodstypes.AuthSchemeNative {
		t.Fatalf("expected admin account to use native auth, got %q", adminAccount.AuthenticationScheme)
	}
	if adminAccount.ClientServerNegotiation {
		t.Fatal("expected native admin account not to require client-server negotiation with DONT_CARE policy")
	}
	if adminAccount.CSNegotiationPolicy != irodstypes.CSNegotiationPolicyRequestDontCare {
		t.Fatalf("expected admin account DONT_CARE policy, got %q", adminAccount.CSNegotiationPolicy)
	}

	requestAccount, err := irodstypes.CreateIRODSAccount(
		cfg.IrodsHost,
		cfg.IrodsPort,
		"alice",
		cfg.IrodsZone,
		cfg.RequestAuthScheme(),
		"secret",
		cfg.IrodsDefaultResource,
	)
	if err != nil {
		t.Fatalf("create request account: %v", err)
	}

	cfg.ApplyIRODSConnectionConfig(requestAccount)
	if requestAccount.AuthenticationScheme != irodstypes.AuthSchemePAM {
		t.Fatalf("expected basic request account to use PAM auth, got %q", requestAccount.AuthenticationScheme)
	}
	if !requestAccount.ClientServerNegotiation {
		t.Fatal("expected PAM request account to require client-server negotiation")
	}
	if requestAccount.CSNegotiationPolicy != irodstypes.CSNegotiationPolicyRequestSSL {
		t.Fatalf("expected PAM request account SSL policy, got %q", requestAccount.CSNegotiationPolicy)
	}
}
