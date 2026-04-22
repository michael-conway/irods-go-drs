package internal

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/Nerzal/gocloak/v13"
	"github.com/gorilla/mux"
)

type contextKey string

const (
	tokenIntrospectionContextKey contextKey = "tokenIntrospection"
	authSchemeContextKey         contextKey = "authScheme"
	usernameContextKey           contextKey = "username"
	basicPasswordContextKey      contextKey = "basicPassword"
)

type RequestAuthenticator interface {
	AuthenticateToken(ctx context.Context, accessToken string) (*gocloak.IntroSpectTokenResult, error)
}

func NewRouteAuthMiddleware(authenticator RequestAuthenticator) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			scheme, token, username, basicPassword, err := authorizationTokenFromRequest(r)
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
			ctx = context.WithValue(ctx, authSchemeContextKey, scheme)
			ctx = context.WithValue(ctx, usernameContextKey, username)
			ctx = context.WithValue(ctx, basicPasswordContextKey, basicPassword)
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

func authorizationTokenFromRequest(r *http.Request) (string, string, string, string, error) {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if authHeader == "" {
		return "", "", "", "", fmt.Errorf("missing Authorization header")
	}

	authType, authValue, found := strings.Cut(authHeader, " ")
	if !found || strings.TrimSpace(authValue) == "" {
		return "", "", "", "", fmt.Errorf("expected Authorization: Bearer <token> or Basic <token>")
	}

	authValue = strings.TrimSpace(authValue)

	switch {
	case strings.EqualFold(authType, "Bearer"):
		return "bearer", authValue, "", "", nil
	case strings.EqualFold(authType, "Basic"):
		token, username, password, err := tokenUsernameAndPasswordFromBasicAuthValue(authValue)
		return "basic", token, username, password, err
	default:
		return "", "", "", "", fmt.Errorf("expected Authorization: Bearer <token> or Basic <token>")
	}
}

func tokenUsernameAndPasswordFromBasicAuthValue(authValue string) (string, string, string, error) {
	decoded, err := base64.StdEncoding.DecodeString(authValue)
	if err != nil {
		return "", "", "", fmt.Errorf("invalid Basic authorization value")
	}

	credentials := strings.TrimSpace(string(decoded))
	if credentials == "" {
		return "", "", "", fmt.Errorf("invalid Basic authorization value")
	}

	username, password, found := strings.Cut(credentials, ":")
	if found {
		username = strings.TrimSpace(username)
		password = strings.TrimSpace(password)
		if password != "" {
			return password, username, password, nil
		}

		if token := strings.TrimSpace(username); token != "" {
			return token, "", "", nil
		}
	}

	return credentials, "", "", nil
}

func TokenIntrospectionFromContext(ctx context.Context) (*gocloak.IntroSpectTokenResult, bool) {
	introspection, ok := ctx.Value(tokenIntrospectionContextKey).(*gocloak.IntroSpectTokenResult)
	return introspection, ok
}

func AuthSchemeFromContext(ctx context.Context) (string, bool) {
	scheme, ok := ctx.Value(authSchemeContextKey).(string)
	return scheme, ok
}

func UsernameFromContext(ctx context.Context) (string, bool) {
	username, ok := ctx.Value(usernameContextKey).(string)
	return username, ok
}

func BasicPasswordFromContext(ctx context.Context) (string, bool) {
	password, ok := ctx.Value(basicPasswordContextKey).(string)
	return password, ok
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": message})
}
