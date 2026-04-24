package internal

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/Nerzal/gocloak/v13"
	drs_support "github.com/michael-conway/irods-go-drs/drs-support"
)

type tokenRetrospector interface {
	RetrospectToken(ctx context.Context, accessToken, clientID, clientSecret, realm string) (*gocloak.IntroSpectTokenResult, error)
}

type Keycloak struct {
	gocloak      tokenRetrospector // keycloak client
	clientId     string            // clientId specified in Keycloak
	clientSecret string            // client secret specified in Keycloak
	realm        string            // realm specified in Keycloak
}

func NewKeycloak(drsConfig *drs_support.DrsConfig) *Keycloak {
	client := gocloak.NewClient(drsConfig.OidcUrl)
	if drsConfig.OidcSkipTLSVerify {
		client.RestyClient().SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	}

	return &Keycloak{
		gocloak:      client,
		clientId:     drsConfig.OidcClientId,
		clientSecret: drsConfig.OidcClientSecret,
		realm:        drsConfig.OidcRealm,
	}
}

func (k *Keycloak) AuthenticateToken(ctx context.Context, accessToken string) (*gocloak.IntroSpectTokenResult, error) {
	if k == nil {
		return nil, fmt.Errorf("keycloak auth is not configured")
	}

	if accessToken == "" {
		return nil, fmt.Errorf("missing bearer token")
	}

	result, err := k.gocloak.RetrospectToken(ctx, accessToken, k.clientId, k.clientSecret, k.realm)
	if err != nil {
		return nil, err
	}

	if result == nil || result.Active == nil || !*result.Active {
		return nil, fmt.Errorf("token is not active")
	}

	return result, nil
}

func NewKeycloakFromConfigPaths(configPaths []string) (*Keycloak, error) {
	drsConfig, err := readDrsConfigNoPanic(drs_support.DefaultConfigName, drs_support.DefaultConfigType, configPaths)
	if err != nil {
		return nil, err
	}

	if drsConfig.OidcUrl == "" {
		return nil, fmt.Errorf("oidc auth is not configured")
	}

	if drsConfig.OidcClientId == "" || drsConfig.OidcClientSecret == "" || drsConfig.OidcRealm == "" {
		return nil, fmt.Errorf("oidc auth is missing required client configuration")
	}

	return NewKeycloak(drsConfig), nil
}

func readDrsConfigNoPanic(configName string, configType string, configPaths []string) (cfg *drs_support.DrsConfig, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("unable to read drs config: %v", recovered)
		}
	}()

	return drs_support.ReadDrsConfig(configName, configType, configPaths)
}
