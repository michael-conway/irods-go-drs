package test

import (
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
	if actual.HttpsAccessMethodBaseURL != "https://download.example.org/api/v1/path/contents?irods_path=" {
		t.Fatalf("expected https access method base URL from config, got %q", actual.HttpsAccessMethodBaseURL)
	}
	if !actual.HttpsAccessUseTicket {
		t.Fatal("expected https access ticket mode from config")
	}
	if actual.DefaultTicketLifetimeMinutes != 720 {
		t.Fatalf("expected default ticket lifetime from config to be 720, got %d", actual.DefaultTicketLifetimeMinutes)
	}
	if actual.DefaultTicketUseLimit != 50 {
		t.Fatalf("expected default ticket use limit from config to be 50, got %d", actual.DefaultTicketUseLimit)
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
