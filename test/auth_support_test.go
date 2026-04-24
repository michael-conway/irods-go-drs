package test

import (
	"testing"

	drs_support "github.com/michael-conway/irods-go-drs/drs-support"
	"github.com/michael-conway/irods-go-drs/internal"
)

func TestNewKeycloak(t *testing.T) {
	var confs = [1]string{"./resources/"}
	drsConfig, err := drs_support.ReadDrsConfig("drs-config1", "yaml", confs[:])
	if err != nil {
		t.Errorf("error reading drs config: %s", err)
	}
	keycloak := internal.NewKeycloak(drsConfig)
	if keycloak == nil {
		t.Errorf("did not create keycloak")
	}
}
