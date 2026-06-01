package internal

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"

	irodstypes "github.com/cyverse/go-irodsclient/irods/types"
	irodsauth "github.com/michael-conway/go-irodsclient-extensions/irodsauth"
	drs_support "github.com/michael-conway/irods-go-drs/drs-support"
)

/***
ServiceContext is a shared object that holds service-related information and configurations.
This contains the hooks to talk to underlying iRODS services, authenticate requests
to the underlying iRODS server, and to manage configuration.
*/

var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

const drsServiceContextKey contextKey = "drsServiceContext"

type DrsServiceContext struct {
	DrsConfig    *drs_support.DrsConfig
	IrodsAccount *irodstypes.IRODSAccount
}

func NewRouteServiceContextMiddleware(drsConfig *drs_support.DrsConfig) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			serviceContext, err := NewDrsServiceContext(r.Context(), drsConfig)
			if err != nil {
				requestLogger.Warn(
					"failed to build DRS service context",
					"method", r.Method,
					"path", r.URL.Path,
					"error", err.Error(),
				)
				writeJSONError(w, http.StatusUnauthorized, "request authentication failed")
				return
			}

			ctx := context.WithValue(r.Context(), drsServiceContextKey, serviceContext)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func NewDefaultRouteServiceContextMiddleware() func(next http.Handler) http.Handler {
	drsConfig, err := drs_support.ReadDrsConfig("", "", nil)
	if err != nil {
		log.Printf("service context middleware disabled: %v", err)
		return nil
	}

	return NewRouteServiceContextMiddleware(drsConfig)
}

func DrsServiceContextFromContext(ctx context.Context) (*DrsServiceContext, bool) {
	serviceContext, ok := ctx.Value(drsServiceContextKey).(*DrsServiceContext)
	return serviceContext, ok
}

func NewDrsServiceContext(ctx context.Context, drsConfig *drs_support.DrsConfig) (*DrsServiceContext, error) {

	if ctx == nil {
		return nil, fmt.Errorf("no context provided")
	}

	if drsConfig == nil {
		return nil, fmt.Errorf("no drs config provided")
	}

	authScheme, ok := AuthSchemeFromContext(ctx)
	if !ok || authScheme == "" {
		return nil, fmt.Errorf("failed to determine auth scheme")
	}

	logger.Info(fmt.Sprintf("auth mode: %s", authScheme))

	userName, ok := UsernameFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("failed to determine user name")
	}
	userName = strings.TrimSpace(userName)
	if authScheme == "bearer" && userName == "" {
		return nil, fmt.Errorf("failed to determine trusted bearer user identity")
	}

	logger.Info(fmt.Sprintf("userName: %s", userName))

	password := ""
	if authScheme == irodsauth.AuthSchemeBasic {
		var ok bool
		password, ok = BasicPasswordFromContext(ctx)
		if !ok {
			return nil, fmt.Errorf("failed to determine basic password")
		}
	}

	irodsAccount, err := irodsauth.CreateAccount(irodsauth.Request{
		AuthScheme:    authScheme,
		Username:      userName,
		BasicPassword: password,
	}, irodsauth.Config{
		Host:                  drsConfig.IrodsHost,
		Port:                  drsConfig.IrodsPort,
		Zone:                  drsConfig.IrodsZone,
		DefaultResource:       drsConfig.IrodsDefaultResource,
		AdminUser:             drsConfig.IrodsAdminUser,
		AdminPassword:         drsConfig.IrodsAdminPassword,
		RequestAuthScheme:     drsConfig.RequestAuthScheme(),
		AdminAuthScheme:       drsConfig.AdminAuthScheme(),
		ApplyConnectionConfig: drsConfig.ApplyIRODSConnectionConfig,
	})
	if err != nil {
		return nil, err
	}

	return &DrsServiceContext{
		DrsConfig:    drsConfig,
		IrodsAccount: irodsAccount,
	}, nil
}
