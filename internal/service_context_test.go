package internal

import (
	"context"
	"strings"
	"testing"

	drs_support "github.com/michael-conway/irods-go-drs/drs-support"
)

func TestNewDrsServiceContextRejectsBearerWithoutTrustedUsername(t *testing.T) {
	ctx := context.WithValue(context.Background(), authSchemeContextKey, "bearer")
	ctx = context.WithValue(ctx, usernameContextKey, "")

	_, err := NewDrsServiceContext(ctx, &drs_support.DrsConfig{})
	if err == nil {
		t.Fatalf("expected error for missing trusted bearer username")
	}
	if !strings.Contains(err.Error(), "trusted bearer user identity") {
		t.Fatalf("expected trusted bearer identity error, got %v", err)
	}
}
