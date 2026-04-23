//go:build integration
// +build integration

package test

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

const testBearerTokenEnvVar = "DRS_TEST_BEARER_TOKEN"

func optionalBearerToken() string {
	return strings.TrimSpace(os.Getenv(testBearerTokenEnvVar))
}

func requireBearerToken(t *testing.T) string {
	t.Helper()

	token := optionalBearerToken()
	if token == "" {
		t.Skipf("%s is not set", testBearerTokenEnvVar)
	}

	return token
}

func newIntegrationRequest(t *testing.T, method string, url string, body io.Reader) *http.Request {
	t.Helper()

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	if token := optionalBearerToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return req
}
