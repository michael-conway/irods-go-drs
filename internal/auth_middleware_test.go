package internal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Nerzal/gocloak/v13"
	"github.com/gorilla/mux"
)

type mockRequestAuthenticator struct {
	result *AuthenticatedToken
	err    error
}

func (m *mockRequestAuthenticator) AuthenticateToken(ctx context.Context, accessToken string) (*AuthenticatedToken, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

func TestRouteAuthMiddlewareBearerUsesTrustedIdentity(t *testing.T) {
	active := true
	authenticator := &mockRequestAuthenticator{
		result: &AuthenticatedToken{
			Introspection: &gocloak.IntroSpectTokenResult{Active: &active},
			Username:      "test1",
		},
	}

	router := mux.NewRouter()
	router.Use(NewRouteAuthMiddleware(authenticator))
	router.HandleFunc("/secure", func(w http.ResponseWriter, r *http.Request) {
		username, ok := UsernameFromContext(r.Context())
		if !ok || username != "test1" {
			t.Fatalf("expected trusted username in context, got ok=%t username=%q", ok, username)
		}
		scheme, ok := AuthSchemeFromContext(r.Context())
		if !ok || scheme != "bearer" {
			t.Fatalf("expected bearer auth scheme in context, got ok=%t scheme=%q", ok, scheme)
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Bearer abc.def.ghi")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRouteAuthMiddlewareBearerRejectsMissingTrustedIdentity(t *testing.T) {
	active := true
	authenticator := &mockRequestAuthenticator{
		result: &AuthenticatedToken{
			Introspection: &gocloak.IntroSpectTokenResult{Active: &active},
			Username:      "",
		},
	}

	router := mux.NewRouter()
	router.Use(NewRouteAuthMiddleware(authenticator))
	router.HandleFunc("/secure", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// The payload contains preferred_username, but middleware must not parse/use it.
	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Bearer eyJhbGciOiJub25lIn0.eyJwcmVmZXJyZWRfdXNlcm5hbWUiOiJzaG91bGQtbm90LWJlLXVzZWQifQ.sig")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "authoritative user identity") {
		t.Fatalf("expected authoritative identity error, got %s", rec.Body.String())
	}
}
