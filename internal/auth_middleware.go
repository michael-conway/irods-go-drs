package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/Nerzal/gocloak/v13"
	"github.com/gorilla/mux"
)

type contextKey string

const tokenIntrospectionContextKey contextKey = "tokenIntrospection"

type RequestAuthenticator interface {
	AuthenticateToken(ctx context.Context, accessToken string) (*gocloak.IntroSpectTokenResult, error)
}

func NewRouteAuthMiddleware(authenticator RequestAuthenticator) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := bearerTokenFromRequest(r)
			if err != nil {
				writeJSONError(w, http.StatusUnauthorized, err.Error())
				return
			}

			if authenticator == nil {
				writeJSONError(w, http.StatusInternalServerError, "auth middleware is enabled but no authenticator is configured")
				return
			}

			introspection, err := authenticator.AuthenticateToken(r.Context(), token)
			if err != nil {
				writeJSONError(w, http.StatusUnauthorized, fmt.Sprintf("authentication failed: %v", err))
				return
			}

			ctx := context.WithValue(r.Context(), tokenIntrospectionContextKey, introspection)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func NewDefaultRouteAuthMiddleware() mux.MiddlewareFunc {
	keycloak, err := NewKeycloakFromConfigPaths(nil)
	if err != nil {
		log.Printf("auth middleware disabled: %v", err)
		return nil
	}

	return NewRouteAuthMiddleware(keycloak)
}

func bearerTokenFromRequest(r *http.Request) (string, error) {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if authHeader == "" {
		return "", fmt.Errorf("missing Authorization header")
	}

	tokenType, token, found := strings.Cut(authHeader, " ")
	if !found || !strings.EqualFold(tokenType, "Bearer") || strings.TrimSpace(token) == "" {
		return "", fmt.Errorf("expected Authorization: Bearer <token>")
	}

	return strings.TrimSpace(token), nil
}

func TokenIntrospectionFromContext(ctx context.Context) (*gocloak.IntroSpectTokenResult, bool) {
	introspection, ok := ctx.Value(tokenIntrospectionContextKey).(*gocloak.IntroSpectTokenResult)
	return introspection, ok
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": message})
}
