package internal

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	drs_support "github.com/michael-conway/irods-go-drs/drs-support"
)

func TestGetOpenAPISpecUsesRequestHost(t *testing.T) {
	oldConfigReader := readRouteDrsConfig
	readRouteDrsConfig = func() (*drs_support.DrsConfig, error) {
		return &drs_support.DrsConfig{DrsListenPort: 9443}, nil
	}
	defer func() { readRouteDrsConfig = oldConfigReader }()

	req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	req.Host = "drs.example.org:1234"
	rec := httptest.NewRecorder()

	GetOpenAPISpec(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "default: drs.example.org:1234") {
		t.Fatalf("expected request host and port in spec, got %q", body)
	}
}

func TestGetSwaggerUIUsesSameOriginOpenAPISpec(t *testing.T) {
	oldConfigReader := readRouteDrsConfig
	readRouteDrsConfig = func() (*drs_support.DrsConfig, error) {
		return &drs_support.DrsConfig{DrsListenPort: 9443}, nil
	}
	defer func() { readRouteDrsConfig = oldConfigReader }()

	req := httptest.NewRequest(http.MethodGet, "/swagger", nil)
	req.Host = "drs.example.org:1234"
	rec := httptest.NewRecorder()

	GetSwaggerUI(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `url: "/openapi.yaml"`) {
		t.Fatalf("expected same-origin openapi spec url in swagger ui, got %q", body)
	}

	if strings.Contains(body, `http://drs.example.org`) {
		t.Fatalf("expected swagger ui not to use an absolute cross-origin spec url, got %q", body)
	}
}
