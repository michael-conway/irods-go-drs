package test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Nerzal/gocloak/v13"
	"github.com/gorilla/mux"
	"github.com/michael-conway/irods-go-drs/internal"
)

type fakeAuthenticator struct {
	called     bool
	result     *gocloak.IntroSpectTokenResult
	err        error
	lastAccess string
}

func (f *fakeAuthenticator) AuthenticateToken(_ context.Context, accessToken string) (*gocloak.IntroSpectTokenResult, error) {
	f.called = true
	f.lastAccess = accessToken
	return f.result, f.err
}

func TestRouteAuthMiddlewareAllowsUnwrappedPublicRoutes(t *testing.T) {
	authenticator := &fakeAuthenticator{}

	router := mux.NewRouter()
	router.HandleFunc("/ga4gh/drs/v1/service-info", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Methods(http.MethodGet).Name("GetServiceInfo")

	req := httptest.NewRequest(http.MethodGet, "/ga4gh/drs/v1/service-info", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200 for public route, got %d", resp.Code)
	}

	if authenticator.called {
		t.Fatal("expected public route to skip auth")
	}
}

func TestRouteAuthMiddlewareRejectsMissingBearerToken(t *testing.T) {
	router := mux.NewRouter()
	router.Handle("/ga4gh/drs/v1/objects/{object_id}", internal.NewRouteAuthMiddleware(&fakeAuthenticator{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))).Methods(http.MethodGet).Name("GetObject")

	req := httptest.NewRequest(http.MethodGet, "/ga4gh/drs/v1/objects/example", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing token, got %d", resp.Code)
	}
}

func TestRouteAuthMiddlewareCallsAuthenticator(t *testing.T) {
	active := true
	authenticator := &fakeAuthenticator{
		result: &gocloak.IntroSpectTokenResult{Active: &active},
	}

	router := mux.NewRouter()
	router.Handle("/ga4gh/drs/v1/objects/{object_id}", internal.NewRouteAuthMiddleware(authenticator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := internal.TokenIntrospectionFromContext(r.Context()); !ok {
			t.Fatal("expected introspection details on request context")
		}

		w.WriteHeader(http.StatusOK)
	}))).Methods(http.MethodGet).Name("GetObject")

	req := httptest.NewRequest(http.MethodGet, "/ga4gh/drs/v1/objects/example", nil)
	req.Header.Set("Authorization", "Bearer token-123")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200 for authenticated request, got %d", resp.Code)
	}

	if !authenticator.called {
		t.Fatal("expected protected route to call authenticator")
	}

	if authenticator.lastAccess != "token-123" {
		t.Fatalf("expected bearer token to be passed to authenticator, got %q", authenticator.lastAccess)
	}
}

func TestRouteAuthMiddlewareReturnsServerErrorWithoutAuthenticator(t *testing.T) {
	router := mux.NewRouter()
	router.Handle("/ga4gh/drs/v1/objects/{object_id}", internal.NewRouteAuthMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))).Methods(http.MethodGet).Name("GetObject")

	req := httptest.NewRequest(http.MethodGet, "/ga4gh/drs/v1/objects/example", nil)
	req.Header.Set("Authorization", "Bearer token-123")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when auth is required but not configured, got %d", resp.Code)
	}
}
