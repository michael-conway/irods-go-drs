package internal

import (
	"github.com/Nerzal/gocloak/v13"
	drs_support "github.com/michael-conway/irods-go-drs/drs-support"
)

type keycloak struct {
	gocloak      *gocloak.GoCloak // keycloak client
	clientId     string           // clientId specified in Keycloak
	clientSecret string           // client secret specified in Keycloak
	realm        string           // realm specified in Keycloak
}

func newKeycloak(drsConfig drs_support.DrsConfig) *keycloak {
	return &keycloak{
		gocloak:      gocloak.NewClient(drsConfig.OidcUrl),
		clientId:     drsConfig.OidcClientId,
		clientSecret: drsConfig.OidcClientSecret,
		realm:        drsConfig.OidcRealm,
	}
}
