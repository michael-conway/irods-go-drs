package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestLoggerEmitsStructuredRequestFields(t *testing.T) {
	originalLogger := requestLogger
	defer func() { requestLogger = originalLogger }()

	var buffer bytes.Buffer
	requestLogger = slog.New(slog.NewJSONHandler(&buffer, nil))

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
	})

	req := httptest.NewRequest(http.MethodGet, "/ga4gh/drs/v1/objects/object-123/access/s3-user", nil)
	req.Header.Set("Authorization", "Basic dGVzdDE6cGFzc3dvcmQ=")
	req = mux.SetURLVars(req, map[string]string{
		"object_id": "object-123",
		"access_id": "s3-user",
	})

	rec := httptest.NewRecorder()
	Logger(inner, "GetAccessURL").ServeHTTP(rec, req)

	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d", rec.Code)
	}

	var entry map[string]any
	if err := json.Unmarshal(buffer.Bytes(), &entry); err != nil {
		t.Fatalf("unmarshal structured log entry: %v; log=%q", err, buffer.String())
	}

	if entry["route"] != "GetAccessURL" {
		t.Fatalf("expected route GetAccessURL, got %+v", entry["route"])
	}
	if entry["object_id"] != "object-123" {
		t.Fatalf("expected object_id object-123, got %+v", entry["object_id"])
	}
	if entry["access_id"] != "s3-user" {
		t.Fatalf("expected access_id s3-user, got %+v", entry["access_id"])
	}
	if entry["auth_mode"] != "basic" {
		t.Fatalf("expected auth_mode basic, got %+v", entry["auth_mode"])
	}
	if entry["error_class"] != "not_implemented" {
		t.Fatalf("expected error_class not_implemented, got %+v", entry["error_class"])
	}
}

func TestRequestAuthModeContextPrecedence(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ga4gh/drs/v1/objects/object-123", nil)
	req.Header.Set("Authorization", "Basic dGVzdDE6cGFzc3dvcmQ=")
	req = req.WithContext(context.WithValue(req.Context(), authSchemeContextKey, "bearer"))

	mode := requestAuthMode(req)
	if mode != "bearer" {
		t.Fatalf("expected bearer from context, got %q", mode)
	}
}

func TestRequestErrorClassMapping(t *testing.T) {
	cases := []struct {
		statusCode int
		expected   string
	}{
		{statusCode: http.StatusOK, expected: "none"},
		{statusCode: http.StatusBadRequest, expected: "bad_request"},
		{statusCode: http.StatusUnauthorized, expected: "auth_error"},
		{statusCode: http.StatusNotFound, expected: "not_found"},
		{statusCode: http.StatusNotImplemented, expected: "not_implemented"},
		{statusCode: http.StatusInternalServerError, expected: "server_error"},
	}

	for _, tc := range cases {
		actual := requestErrorClass(tc.statusCode)
		if actual != tc.expected {
			t.Fatalf("status %d expected %q, got %q", tc.statusCode, tc.expected, actual)
		}
	}
}
