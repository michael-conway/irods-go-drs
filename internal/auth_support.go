package internal

import (
	"context"
	"fmt"

	"github.com/Nerzal/gocloak/v13"
	"github.com/michael-conway/go-irodsclient-extensions/oidcverify"
	drs_support "github.com/michael-conway/irods-go-drs/drs-support"
)

type bearerTokenVerifier interface {
	VerifyToken(ctx context.Context, accessToken string) (*oidcverify.VerifiedToken, error)
}

type Keycloak struct {
	verifier bearerTokenVerifier
}

func NewKeycloak(drsConfig *drs_support.DrsConfig) *Keycloak {
	if drsConfig == nil {
		drsConfig = &drs_support.DrsConfig{}
	}

	return &Keycloak{
		verifier: oidcverify.NewVerifier(oidcverify.Config{
			BaseURL:            drsConfig.OidcUrl,
			Realm:              drsConfig.OidcRealm,
			ClientID:           drsConfig.OidcClientId,
			ClientSecret:       drsConfig.OidcClientSecret,
			InsecureSkipVerify: drsConfig.OidcSkipTLSVerify,
		}),
	}
}

func (k *Keycloak) AuthenticateToken(ctx context.Context, accessToken string) (*AuthenticatedToken, error) {
	if k == nil {
		return nil, fmt.Errorf("keycloak auth is not configured")
	}
	if k.verifier == nil {
		return nil, fmt.Errorf("keycloak auth verifier is not configured")
	}

	if accessToken == "" {
		return nil, fmt.Errorf("missing bearer token")
	}

	result, err := k.verifier.VerifyToken(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("missing token verification result")
	}
	if result.Username == "" {
		return nil, fmt.Errorf("token is missing preferred_username identity claim")
	}
	active := result.Introspection.Active

	return &AuthenticatedToken{
		Introspection: &gocloak.IntroSpectTokenResult{Active: &active},
		Username:      result.Username,
	}, nil
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
