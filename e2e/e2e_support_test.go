//go:build e2e
// +build e2e

package e2e

import (
	"crypto/tls"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func requireE2EBaseURL(t *testing.T) string {
	t.Helper()

	baseURL := strings.TrimSpace(os.Getenv("DRS_E2E_BASE_URL"))
	if baseURL == "" {
		t.Skip("DRS_E2E_BASE_URL is not set")
	}

	return baseURL
}

func requireE2EBearerToken(t *testing.T) string {
	t.Helper()

	token := strings.TrimSpace(os.Getenv("DRS_TEST_BEARER_TOKEN"))
	if token == "" {
		t.Skip("DRS_TEST_BEARER_TOKEN is not set")
	}

	return token
}

func newE2EHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if strings.EqualFold(strings.TrimSpace(os.Getenv("DRS_E2E_SKIP_TLS_VERIFY")), "true") {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}
}
