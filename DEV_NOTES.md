# Development Notes

Use this file for the main working rules in `irods-go-drs`.

## Service shape

`irods-go-drs` is the DRS-facing service for iRODS.

Keep the code split this way:

- `internal/` handles HTTP, request parsing, and response mapping
- `drs-support/` holds DRS and iRODS behavior

If you are adding logic, prefer to add it in `drs-support/` first and keep `internal/` thin.

## Core model

- A DRS object maps to an iRODS data object, not a collection.
- DRS metadata is stored as shallow AVUs on the data object.
- Compound objects are manifest-backed data objects.
- Compound membership comes from manifest content, not AVUs.
- Checksum and version should come from real iRODS state when possible.

Current AVUs:

- `iRODS:DRS:ID`
- `iRODS:DRS:VERSION`
- `iRODS:DRS:MIME_TYPE`
- `iRODS:DRS:DESCRIPTION`
- `iRODS:DRS:ALIAS`
- `iRODS:DRS:COMPOUND_MANIFEST`

Metadata unit:

- `iRODS:DRS`

## Compound objects

A compound object is a JSON manifest stored as an iRODS data object.

Keep these rules:

- the manifest file is the DRS object
- the manifest points to child DRS IDs
- a child can be another manifest-backed object
- manifests do not point back to parents

## Access methods

Access method generation belongs in `drs-support`.

Current direction:

- `http` should resolve later through `/access`
- `irods` should resolve later through `/access`
- `local` may return a direct mapped path
- `s3` is still a stub

## Local docs

When the service is running:

- Swagger UI: `http://localhost:8080/swagger`
- OpenAPI spec: `http://localhost:8080/openapi.yaml`

## Testing

Use three layers:

- unit tests next to the package, run with `go test ./...`
- integration tests under `test/`, run with `go test -tags=integration ./test/...`
- end-to-end tests under `e2e/`, run with `go test -tags=e2e ./e2e/...`

## Live-test configuration

Use a single shared config file for both integration and E2E tests:

- `DRS_E2E_CONFIG_FILE`

Configuration strategy:

- keep normal service settings at the top level
- keep test-only values in an `E2E` section
- use the same file for both `go test -tags=integration ./test/...` and `go test -tags=e2e ./e2e/...`
- allow direct env vars only as optional overrides

Current shared live-test inputs:

- `E2E.BaseURL`
- `E2E.SkipTLSVerify`
- `E2E.BearerToken`
- `IrodsAdminUser`
- `IrodsAdminPassword`
- `IrodsPrimaryTestUser`
- `IrodsSecondaryTestUser`
- `OidcInsecureSkipVerify`

Current test-user strategy:

- use iRODS proxy authentication for direct iRODS test setup
- use `IrodsAdminUser` and `IrodsAdminPassword` as the proxy credentials
- default the effective test user to `IrodsPrimaryTestUser`
- reserve `IrodsSecondaryTestUser` for multi-user or permission-oriented tests

Current live-test variables:

- `DRS_E2E_CONFIG_FILE`
- `DRS_E2E_BASE_URL`
- `DRS_TEST_BEARER_TOKEN`
- `DRS_E2E_SKIP_TLS_VERIFY`

Current OIDC TLS setting names:

- prefer `OidcInsecureSkipVerify` in YAML
- prefer `DRS_OIDC_INSECURE_SKIP_VERIFY` in env
- keep `OidcSkipTLSVerify` and `DRS_OIDC_SKIP_TLS_VERIFY` only as compatibility aliases

For console and CLI-oriented workflows, assume `gocmd` is available on `PATH`.
