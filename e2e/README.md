# E2E Tests

This directory is reserved for end-to-end tests that run against the real DRS HTTP service and the docker compose test framework.

These tests are intended to exercise the full stack:

* HTTP routing and middleware
* authentication
* service context creation
* iRODS integration
* Keycloak-backed bearer token flows
* docker-compose-managed runtime dependencies

## Build Tag

End-to-end tests in this directory should use the `e2e` build tag:

```go
//go:build e2e
// +build e2e
```

Run them explicitly:

```bash
go test -tags=e2e ./e2e/...
```

## Environment

The current convention for E2E tests matches the shared live-test config style
used in `irods-go-rest`.

Use one shared config file and point the test helpers at it with:

* `DRS_E2E_CONFIG_FILE`

Optional overrides still exist for convenience, but the shared config file is
the intended source of truth.

The current E2E inputs are:

* `DRS_E2E_CONFIG_FILE` - required shared config file for both `test/` integration runs and `e2e/` runs
* `DRS_E2E_BASE_URL` - base URL of the running DRS service
* `DRS_TEST_BEARER_TOKEN` - bearer token for authenticated endpoint tests
* `DRS_E2E_SKIP_TLS_VERIFY` - optional, set to `true` when the docker test framework uses self-signed TLS

The shared config file may contain:

* normal top-level DRS runtime settings
* an `E2E` section for test-only values

For direct iRODS-backed test setup in integration helpers, keep these top-level
fields in the same file:

* `IrodsAdminUser`
* `IrodsAdminPassword`
* `IrodsPrimaryTestUser`
* `IrodsSecondaryTestUser`

The test helpers use proxy authentication through the admin account and default
the effective test user to `IrodsPrimaryTestUser`.

Current `E2E` fields:

* `E2E.BaseURL`
* `E2E.SkipTLSVerify`
* `E2E.BearerToken`

Example:

```bash
export DRS_E2E_CONFIG_FILE=./e2e/drs-config.e2e.sample.yaml
go test -tags=e2e ./e2e/...
go test -tags=integration ./test/...
```

Sample file:

* [e2e/drs-config.e2e.sample.yaml](/Users/conwaymc/Documents/workspace-gabble/irods-go-drs/e2e/drs-config.e2e.sample.yaml)

## Source of Truth

The docker-compose-backed test environment is under:

* `deployments/docker-test-framework/5-0`

Use `DEV_NOTES.md` for the higher-level testing taxonomy and environment setup guidance.
