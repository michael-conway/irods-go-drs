package test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Nerzal/gocloak/v13"
	"github.com/gorilla/mux"
	drs_support "github.com/michael-conway/irods-go-drs/drs-support"
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

func TestRouteAuthMiddlewareRejectsMissingAuthorization(t *testing.T) {
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

func TestRouteAuthMiddlewareAcceptsBasicTokenInPassword(t *testing.T) {
	authenticator := &fakeAuthenticator{}

	router := mux.NewRouter()
	router.Handle("/ga4gh/drs/v1/objects/{object_id}", internal.NewRouteAuthMiddleware(authenticator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := internal.TokenIntrospectionFromContext(r.Context()); ok {
			t.Fatal("did not expect token introspection details for basic auth")
		}
		if scheme, ok := internal.AuthSchemeFromContext(r.Context()); !ok || scheme != "basic" {
			t.Fatalf("expected auth scheme basic in context, got %q, ok=%t", scheme, ok)
		}
		if user, ok := internal.UsernameFromContext(r.Context()); !ok || user != "drs-client" {
			t.Fatalf("expected username in context, got %q, ok=%t", user, ok)
		}
		if password, ok := internal.BasicPasswordFromContext(r.Context()); !ok || password != "token-456" {
			t.Fatalf("expected basic password in context, got %q, ok=%t", password, ok)
		}

		w.WriteHeader(http.StatusOK)
	}))).Methods(http.MethodGet).Name("GetObject")

	encoded := base64.StdEncoding.EncodeToString([]byte("drs-client:token-456"))
	req := httptest.NewRequest(http.MethodGet, "/ga4gh/drs/v1/objects/example", nil)
	req.Header.Set("Authorization", "Basic "+encoded)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200 for authenticated request, got %d", resp.Code)
	}

	if authenticator.called {
		t.Fatal("did not expect basic auth to call bearer token authenticator")
	}
}

func TestRouteAuthMiddlewareRejectsInvalidBasicAuthorization(t *testing.T) {
	router := mux.NewRouter()
	router.Handle("/ga4gh/drs/v1/objects/{object_id}", internal.NewRouteAuthMiddleware(&fakeAuthenticator{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))).Methods(http.MethodGet).Name("GetObject")

	req := httptest.NewRequest(http.MethodGet, "/ga4gh/drs/v1/objects/example", nil)
	req.Header.Set("Authorization", "Basic not-base64")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid Basic authorization, got %d", resp.Code)
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
		if scheme, ok := internal.AuthSchemeFromContext(r.Context()); !ok || scheme != "bearer" {
			t.Fatalf("expected auth scheme bearer in context, got %q, ok=%t", scheme, ok)
		}
		if user, ok := internal.UsernameFromContext(r.Context()); !ok || user != "" {
			t.Fatalf("expected username in context, got %q, ok=%t", user, ok)
		}
		if password, ok := internal.BasicPasswordFromContext(r.Context()); !ok || password != "" {
			t.Fatalf("expected empty basic password in context for bearer auth, got %q, ok=%t", password, ok)
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

func TestRouteAuthMiddlewareStoresEmptyUsernameWhenMissing(t *testing.T) {
	active := true
	authenticator := &fakeAuthenticator{
		result: &gocloak.IntroSpectTokenResult{Active: &active},
	}

	router := mux.NewRouter()
	router.Handle("/ga4gh/drs/v1/objects/{object_id}", internal.NewRouteAuthMiddleware(authenticator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := internal.UsernameFromContext(r.Context())
		if !ok {
			t.Fatal("expected username key in context")
		}
		if user != "" {
			t.Fatalf("expected empty username when introspection has none, got %q", user)
		}
		password, ok := internal.BasicPasswordFromContext(r.Context())
		if !ok {
			t.Fatal("expected basic password key in context")
		}
		if password != "" {
			t.Fatalf("expected empty basic password for bearer auth, got %q", password)
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
}

func TestRouteAuthMiddlewareExtractsBearerUsernameFromJWTClaims(t *testing.T) {
	active := true
	authenticator := &fakeAuthenticator{
		result: &gocloak.IntroSpectTokenResult{Active: &active},
	}

	router := mux.NewRouter()
	router.Handle("/ga4gh/drs/v1/objects/{object_id}", internal.NewRouteAuthMiddleware(authenticator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := internal.UsernameFromContext(r.Context())
		if !ok {
			t.Fatal("expected username key in context")
		}
		if user != "test1" {
			t.Fatalf("expected bearer username from jwt claims, got %q", user)
		}

		w.WriteHeader(http.StatusOK)
	}))).Methods(http.MethodGet).Name("GetObject")

	req := httptest.NewRequest(http.MethodGet, "/ga4gh/drs/v1/objects/example", nil)
	req.Header.Set("Authorization", "Bearer "+unsignedJWT(t, map[string]any{
		"preferred_username": "test1",
		"sub":                "subject-123",
	}))
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200 for authenticated request, got %d", resp.Code)
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

func TestRouteAuthMiddlewareAndServiceContextMiddlewarePopulateContext(t *testing.T) {
	drsConfig := &drs_support.DrsConfig{
		IrodsHost:          "localhost",
		IrodsPort:          1247,
		IrodsZone:          "tempZone",
		IrodsAdminUser:     "rods",
		IrodsAdminPassword: "rods",
		IrodsAuthScheme:    "native",
	}

	router := mux.NewRouter()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serviceContext, ok := internal.DrsServiceContextFromContext(r.Context())
		if !ok || serviceContext == nil {
			t.Fatal("expected service context on request context")
		}
		if serviceContext.IrodsAccount == nil {
			t.Fatal("expected iRODS account in service context")
		}
		if serviceContext.IrodsAccount.ClientUser != "basic-user" {
			t.Fatalf("expected account for basic user, got %q", serviceContext.IrodsAccount.ClientUser)
		}
		w.WriteHeader(http.StatusOK)
	})
	handlerWithContext := internal.NewRouteServiceContextMiddleware(drsConfig)(handler)
	handlerWithAuth := internal.NewRouteAuthMiddleware(nil)(handlerWithContext)

	router.Handle("/ga4gh/drs/v1/objects/{object_id}", handlerWithAuth).Methods(http.MethodGet).Name("GetObject")

	req := httptest.NewRequest(http.MethodGet, "/ga4gh/drs/v1/objects/example", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("basic-user:password-123")))
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200 for authenticated request, got %d", resp.Code)
	}
}

func unsignedJWT(t *testing.T, claims map[string]any) string {
	t.Helper()

	headerBytes, err := json.Marshal(map[string]any{
		"alg": "none",
		"typ": "JWT",
	})
	if err != nil {
		t.Fatalf("marshal jwt header: %v", err)
	}

	claimBytes, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal jwt claims: %v", err)
	}

	return base64.RawURLEncoding.EncodeToString(headerBytes) + "." + base64.RawURLEncoding.EncodeToString(claimBytes) + "."
}
