package internal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIndexReturnsNotImplemented(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ga4gh/drs/v1/", nil)
	rec := httptest.NewRecorder()

	Index(rec, req)

	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d", rec.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if !strings.Contains(response["message"], "not supported in this deployment") {
		t.Fatalf("expected explicit unsupported-operation message, got %+v", response)
	}
}
