# Development Notes

Use this file for the main working rules in `irods-go-drs`.

## Alpha Release Gate (Open Issues)


- [x] Normalize error exposure for security-sensitive paths.
  - Replace raw upstream error bodies with sanitized client-safe messages.
  - Keep detailed causes in structured logs only.

- [x] Harden HTTP server transport defaults.
  - Replace bare `http.ListenAndServe` with an `http.Server` that sets read, header, write, and idle timeouts.

- [x] Prevent response URL poisoning.
  - Do not build `self_uri`/extension `access_url` directly from untrusted `Host` and `X-Forwarded-Proto`.
  - Add explicit trusted external base URL config or trusted-proxy validation.

- [x] Align auth failure status semantics.
  - Return consistent `401/403` for authentication/authorization failures across basic and bearer flows.
  - Avoid returning `500` for expected auth failures.

- [x] Tighten production config posture and docs.
  - Ensure production docs default to TLS verification enabled and strict iRODS negotiation policy.
  - Keep insecure development examples explicitly marked as local-only.

## Service shape

`irods-go-drs` is the DRS-facing service for iRODS.

Monitor shared higher-level iRODS client logic against `go-irodsclient-extensions`.

If functionality here is also needed by `irods-go-rest` or other clients, consider refactoring it into `go-irodsclient-extensions` instead of duplicating it across service repositories.

Keep the code split this way:

- `internal/` handles HTTP, request parsing, and response mapping
- `drs-support/` holds DRS and iRODS behavior

If you are adding logic, prefer to add it in `drs-support/` first and keep `internal/` thin.

## Core model

- Atomic DRS objects map to iRODS data objects.
- Compound DRS objects map to iRODS collections marked with DRS AVUs.
- DRS metadata is stored as AVUs on iRODS collections/data objects.
- Compound manifest payloads are generated at request time from collection/data-object AVUs.
- Checksum and version for data objects should come from real iRODS state when possible.

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

A compound object is a collection-backed DRS object.

Keep these rules:

- the root collection carries `iRODS:DRS:ID` and `iRODS:DRS:COMPOUND_MANIFEST`
- included descendant data objects become DRS objects unless excluded by `.drsignore`
- intermediate subcollections are represented in runtime manifest output and carry alias/description AVUs
- runtime manifest JSON is served by `GET /ga4gh/drs/v1/ext/compound/{object_id}`
- `.drsignore` is a creation/preflight control file and is excluded from the compound bundle

## Access methods

Access method generation belongs in `drs-support`.

Ticket parsing, ticket creation helpers, and other reusable client workflows should be considered for extraction into `go-irodsclient-extensions` when they are not DRS-specific.

Current behavior:

- Atomic object `https` access methods are returned with `access_id`; clients call `/objects/{object_id}/access/{access_id}`.
- Atomic object `irods` access method is returned with `access_id=irods`; clients call `/objects/{object_id}/access/irods`.
- Compound object `https` access method is returned with a direct `access_url` to `/ga4gh/drs/v1/ext/compound/{object_id}` (no compound `access_id` hop).
- `local` access method (when enabled) returns a direct mapped `local://` URL.
- `s3` access method generation is active for objects under collections marked with `iRODS:S3:Bucket` AVUs and returns inline S3 URL data; affinity tuning remains an open TODO.

API status notes:

- Passport/bulk POST endpoints are intentionally unsupported in alpha and return `501 Not Implemented`.

## Local docs

When the service is running:

- Swagger UI: `http://localhost:8080/swagger`
- OpenAPI spec: `http://localhost:8080/openapi.yaml`

## Testing

Use three layers:

- unit tests next to the package, run with `go test ./...`
- integration tests under `test/`, run with `go test -tags=integration ./test/...`
- end-to-end tests under `e2e/`, run with `go test -tags=e2e ./e2e/...`

Shared live-test variable:

- `DRS_E2E_CONFIG_FILE`

E2E and integration tests read runtime parameters from that shared config file.
For HTTP route tests, the base URL is derived from `DrsListenPort` as
`http://localhost:<DrsListenPort>`.
Bearer-authenticated route tests are currently skipped by default in this
harness.

For console and CLI-oriented workflows, assume `gocmd` is available on `PATH`.

### Test substrate

Prefer `irods-grid-stack` for local live testing. Use the backend-only stack for
direct iRODS integration tests and the full frontend profile when testing DRS
through its HTTP surface alongside REST/Starbase.

The legacy compose framework under `deployments/docker-test-framework/` is
deprecated. Do not add new tests, fixtures, or sample configurations that depend
on that DRS-local stack. Keep it only for historical reproduction while active
development and sample config updates move to `irods-grid-stack`.

## S3 API

iRODS s3 api docker images - https://hub.docker.com/r/irods/irods_s3_api/tags
iRODS s3 config - https://github.com/irods/irods_client_s3_api#configuration

## Local multi-repo sync (`go.work`)

Use a workspace `go.work` file for local cross-repo development instead of
`replace ../...` directives in `go.mod`.

Current workspace scaffold includes:

- `./go-irodsclient-extensions`
- `./irods-go-rest`
- `./irods-go-drs`

Workflow:

1. develop across repos with `go.work` active
2. keep each repo `go.mod` pinned to real module versions (no local replace)
3. when shared changes are ready, push/tag in `go-irodsclient-extensions`
4. bump dependent repos with `go get <module>@<tag-or-commit>` and `go mod tidy`

Note that GOWORK should be turned off when updating go.mod 

```shell
GOWORK=off go get github.com/michael-conway/go-irodsclient-extensions@<tag-or-sha>
```
