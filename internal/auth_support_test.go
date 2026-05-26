package internal

import (
	"context"
	"errors"
	"testing"

	"github.com/michael-conway/go-irodsclient-extensions/oidcverify"
	drs_support "github.com/michael-conway/irods-go-drs/drs-support"
)

func TestNewKeycloakBuildsVerifier(t *testing.T) {
	keycloak := NewKeycloak(&drs_support.DrsConfig{})
	if keycloak == nil || keycloak.verifier == nil {
		t.Fatal("expected keycloak verifier")
	}
}

type mockBearerVerifier struct {
	result *oidcverify.VerifiedToken
	err    error
}

func (m *mockBearerVerifier) VerifyToken(ctx context.Context, accessToken string) (*oidcverify.VerifiedToken, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

func TestAuthenticateTokenReturnsTrustedUsername(t *testing.T) {
	keycloak := &Keycloak{
		verifier: &mockBearerVerifier{
			result: &oidcverify.VerifiedToken{
				Introspection: oidcverify.Introspection{Active: true},
				Username:      "test1",
			},
		},
	}

	result, err := keycloak.AuthenticateToken(context.Background(), "token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected auth result")
	}
	if result.Username != "test1" {
		t.Fatalf("expected username test1, got %q", result.Username)
	}
	if result.Introspection == nil || result.Introspection.Active == nil || !*result.Introspection.Active {
		t.Fatalf("expected active introspection, got %+v", result.Introspection)
	}
}

func TestAuthenticateTokenPropagatesVerifierError(t *testing.T) {
	keycloak := &Keycloak{
		verifier: &mockBearerVerifier{
			err: errors.New("verify failed"),
		},
	}

	_, err := keycloak.AuthenticateToken(context.Background(), "token")
	if err == nil {
		t.Fatal("expected verifier error")
	}
}

func TestAuthenticateTokenRejectsMissingToken(t *testing.T) {
	keycloak := &Keycloak{verifier: &mockBearerVerifier{}}
	_, err := keycloak.AuthenticateToken(context.Background(), "")
	if err == nil {
		t.Fatal("expected missing token error")
	}
}

func TestAuthenticateTokenRejectsMissingUsername(t *testing.T) {
	keycloak := &Keycloak{
		verifier: &mockBearerVerifier{
			result: &oidcverify.VerifiedToken{
				Introspection: oidcverify.Introspection{Active: true},
			},
		},
	}

	_, err := keycloak.AuthenticateToken(context.Background(), "token")
	if err == nil {
		t.Fatal("expected missing username error")
	}
}
