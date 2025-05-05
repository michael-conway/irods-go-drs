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
	if actual.ClientUser != config.IrodsDrsAdminUser {
		t.Fail()
	}

}
