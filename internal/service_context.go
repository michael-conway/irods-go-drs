package internal

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/cyverse/go-irodsclient/irods/types"
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
	IrodsAccount *types.IRODSAccount
}

func NewRouteServiceContextMiddleware(drsConfig *drs_support.DrsConfig) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			serviceContext, err := NewDrsServiceContext(r.Context(), drsConfig)
			if err != nil {
				writeJSONError(w, http.StatusUnauthorized, fmt.Sprintf("failed to build service context: %v", err))
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

	logger.Info(fmt.Sprintf("userName: %s", userName))

	var (
		irodsAccount *types.IRODSAccount
		err          error
	)
	switch authScheme {
	case "bearer":
		irodsAccount, err = types.CreateIRODSProxyAccount(
			drsConfig.IrodsHost,
			drsConfig.IrodsPort,
			userName,
			drsConfig.IrodsZone,
			drsConfig.IrodsAdminUser,
			drsConfig.IrodsZone,
			types.GetAuthScheme(drsConfig.IrodsAuthScheme),
			drsConfig.IrodsAdminPassword,
			drsConfig.IrodsDefaultResource,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create iRODS proxy account: %w", err)
		}
	case "basic":
		password, ok := BasicPasswordFromContext(ctx)
		if !ok {
			return nil, fmt.Errorf("failed to determine basic password")
		}

		irodsAccount, err = types.CreateIRODSAccount(
			drsConfig.IrodsHost,
			drsConfig.IrodsPort,
			userName,
			drsConfig.IrodsZone,
			types.GetAuthScheme(drsConfig.IrodsAuthScheme),
			password,
			drsConfig.IrodsDefaultResource,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create iRODS account: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported auth scheme %q", authScheme)
	}

	return &DrsServiceContext{
		DrsConfig:    drsConfig,
		IrodsAccount: irodsAccount,
	}, nil
}
