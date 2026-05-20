# E2E Tests

This directory is reserved for end-to-end tests that run against the real DRS
HTTP service and a reachable iRODS test grid.

The preferred local grid is now `irods-grid-stack`. The older
`deployments/docker-test-framework/` stack in this repository is deprecated and
kept only as a compatibility fixture while DRS and REST development workflows
move to the shared grid stack.

These tests are intended to exercise the full stack:

* HTTP routing and middleware
* authentication
* service context creation
* iRODS integration
* docker-compose-managed runtime dependencies

Current route coverage includes:

* `GET /ga4gh/drs/v1/objects/{object_id}` authentication checks
* `GET /ga4gh/drs/v1/objects/{object_id}` for an existing DRS object
* `GET /ga4gh/drs/v1/objects/{object_id}` for a missing DRS object
* `GET /ga4gh/drs/v1/objects/{object_id}` for a compound DRS object with direct `access_url`
* `GET /ga4gh/drs/v1/objects/{object_id}` for an object under an `iRODS:S3:Bucket` AVU-mapped collection
* `GET /ga4gh/drs/v1/ext/compound/{object_id}` runtime manifest retrieval
* compound workflow checks for `.drsignore` exclusion behavior
* compound strip/remove semantics using `drs-support` metadata stripping

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

The shared config file is the only source of runtime settings for both
`test/` integration runs and `e2e/` runs.

For HTTP route tests, the E2E client base URL is derived from:

* `DrsListenPort` as `http://localhost:<DrsListenPort>`

For direct iRODS-backed test setup in integration helpers, keep these top-level
fields in the same file:

* `IrodsAdminUser`
* `IrodsAdminPassword`
* `IrodsPrimaryTestUser`
* `IrodsPrimaryTestPassword`
* `IrodsSecondaryTestUser`
* `IrodsSecondaryTestPassword`

Do not use the old YAML keys `IrodsDrsAdminUser`,
`IrodsDrsAdminPassword`, or `IrodsDrsAdminPasswordFile`.

The test helpers use proxy authentication through the admin account and default
the effective test user to `IrodsPrimaryTestUser`.

For Basic-authenticated E2E coverage, use `IrodsPrimaryTestPassword` or
`IrodsSecondaryTestPassword` from that same shared config file when building the
Authorization header.

Bearer-authenticated E2E/integration route tests are currently skipped by
default in this harness because bearer token injection is not sourced from the
shared config schema.

For OIDC TLS settings in the shared config file, prefer:

* `OidcInsecureSkipVerify`

The older `OidcSkipTLSVerify` name is still accepted as a compatibility alias.

Example:

```bash
export DRS_E2E_CONFIG_FILE=./e2e/drs-config.e2e.sample.yaml
go test -tags=e2e ./e2e/...
go test -tags=integration ./test/...
```

Sample file:

* [e2e/drs-config.e2e.sample.yaml](/Users/conwaymc/Documents/workspace-gabble/irods-go-drs/e2e/drs-config.e2e.sample.yaml)

## Source of Truth

The docker-compose-backed test environment is `irods-grid-stack`:

```bash
cd ../irods-grid-stack
cp .env.example .env

# Backend-only grid for local DRS/REST development from source.
docker compose up -d --build

# Full demo stack including REST, DRS, and Starbase containers.
docker compose --profile frontend up -d --build
```

Use the backend-only mode when you want to run `irods-go-drs` locally from this
repository while reusing grid-stack iRODS, Keycloak, and S3 services. Use the
`frontend` profile when E2E tests should target the containerized DRS service
on the configured host port.

The legacy in-repository compose stack remains under:

* `deployments/docker-test-framework/5-0`

It should not receive new feature work unless a short-term compatibility fix is
needed.

Use [`DEVELOPER_NOTES.md`](../docs/DEVELOPER_NOTES.md) for the higher-level testing taxonomy and environment setup guidance.
