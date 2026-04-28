package test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/michael-conway/irods-go-drs/internal"
)

func TestOpenAPISpecRoute(t *testing.T) {
	router := internal.NewRouter()

	req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "application/yaml") {
		t.Fatalf("expected yaml content type, got %q", got)
	}

	body := rec.Body.String()
	if !containsAll(body, "openapi: 3.0.3", "title: Data Repository Service") {
		t.Fatalf("unexpected response body: %q", body)
	}
}

func TestSwaggerUIRoute(t *testing.T) {
	router := internal.NewRouter()

	req := httptest.NewRequest(http.MethodGet, "/swagger", nil)
	req.Host = "drs.example.org:1234"
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Fatalf("expected html content type, got %q", got)
	}

	body := rec.Body.String()
	if !containsAll(body, "SwaggerUIBundle", "openapi.yaml") {
		t.Fatalf("unexpected response body: %q", body)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(s, part) {
			return false
		}
	}
	return true
}
