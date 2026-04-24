package internal

import (
	"net/http"
	"testing"

	"github.com/Nerzal/gocloak/v13"
	drs_support "github.com/michael-conway/irods-go-drs/drs-support"
)

func TestNewKeycloakConfiguresTLSVerificationSkip(t *testing.T) {
	keycloak := NewKeycloak(&drs_support.DrsConfig{
		OidcUrl:           "https://localhost:8443",
		OidcSkipTLSVerify: true,
	})

	if keycloak == nil {
		t.Fatal("expected keycloak client")
	}

	client, ok := keycloak.gocloak.(*gocloak.GoCloak)
	if !ok {
		t.Fatal("expected gocloak client implementation")
	}

	transport, ok := client.RestyClient().GetClient().Transport.(*http.Transport)
	if !ok {
		t.Fatal("expected http transport")
	}

	if transport.TLSClientConfig == nil || !transport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("expected Resty client TLS config to skip verification")
	}
}
