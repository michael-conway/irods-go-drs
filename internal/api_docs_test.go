package internal

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	drs_support "github.com/michael-conway/irods-go-drs/drs-support"
)

func TestGetOpenAPISpecUsesConfiguredListenPort(t *testing.T) {
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
	if !strings.Contains(body, "default: drs.example.org:9443") {
		t.Fatalf("expected configured host and port in spec, got %q", body)
	}
}

func TestGetSwaggerUIUsesConfiguredListenPort(t *testing.T) {
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
	if !strings.Contains(body, `url: "http://drs.example.org:9443/openapi.yaml"`) {
		t.Fatalf("expected configured host and port in swagger ui, got %q", body)
	}
}
