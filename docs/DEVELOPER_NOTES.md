# Developer Notes

These notes capture maintainer rules for `irods-go-drs`. User-facing setup belongs in [../README.md](../README.md); runtime configuration belongs in [CONFIGURATION_NOTES.md](./CONFIGURATION_NOTES.md).

## Service Boundaries

`irods-go-drs` is the DRS-facing service for iRODS.

Keep responsibilities separated:

- `api/swagger.yaml` is the DRS OpenAPI contract source.
- `internal/` owns HTTP routing, request parsing, auth middleware, and response mapping.
- `drs-support/` owns DRS-to-iRODS behavior, AVU conventions, manifests, validation, and access-method construction.
- `tools/drs-console/` owns DRS administration CLI behavior.
- `tools/drs-certification/` owns certification corpus/report tooling.

If logic is not HTTP-specific, prefer adding it to `drs-support/` before `internal/`.

## Shared Client Logic

Monitor reusable iRODS workflows against `go-irodsclient-extensions`.

Move logic into `go-irodsclient-extensions` when it is:

- useful to both `irods-go-drs` and `irods-go-rest`
- not DRS HTTP response shaping
- stable enough to carry as shared client behavior

Ticket parsing, ticket creation helpers, checksum helpers, and metadata workflows should be considered for extraction when they become cross-service concerns.

## DRS Model Rules

- Atomic DRS objects map to iRODS data objects.
- Compound DRS objects map to iRODS collections marked with DRS AVUs.
- DRS metadata is stored as AVUs on iRODS collections and data objects.
- Compound manifest payloads are generated at request time.
- `.drsignore` is a creation/preflight control file and is excluded from compound bundles.
- Checksum and version data should come from real iRODS state when possible.

Current DRS AVUs:

- `iRODS:DRS:ID`
- `iRODS:DRS:VERSION`
- `iRODS:DRS:MIME_TYPE`
- `iRODS:DRS:DESCRIPTION`
- `iRODS:DRS:ALIAS`
- `iRODS:DRS:COMPOUND_MANIFEST`

Metadata unit:

- `iRODS:DRS`

## Access Methods

Access method generation belongs in `drs-support`.

Current behavior:

- Atomic object `https` returns an `access_id`; clients resolve it through `/objects/{object_id}/access/{access_id}`.
- Atomic object `irods` returns `access_id=irods`.
- Compound object `https` returns a direct `access_url` to `/ga4gh/drs/v1/ext/compound/{object_id}`.
- `local` returns a direct mapped `local://` URL when enabled.
- `s3` returns inline `s3://bucket/key` data for objects under collections marked with `iRODS:S3:Bucket` AVUs.

Passport and bulk POST endpoints are intentionally unsupported in alpha and return `501 Not Implemented`.

## Auth And Security

OIDC bearer validation is configured through Keycloak-compatible OIDC settings. Basic auth maps to iRODS user credentials. Ticket-backed access uses the configured iRODS account path and generated ticket policy.

Production posture:

- `DRS_OIDC_INSECURE_SKIP_VERIFY=false`
- `DRS_IRODS_NEGOTIATION_POLICY=CS_NEG_REQUIRE`
- explicit `DRS_PUBLIC_URL`
- secret files instead of inline secrets

Local-only test stacks may use self-signed OIDC TLS and relaxed iRODS negotiation, but docs and samples must clearly mark those values as development-only.

## Testing

Use three layers:

```bash
GOWORK=off go test ./...
```

```bash
GOWORK=off DRS_E2E_CONFIG_FILE=./e2e/drs-config.e2e.sample.yaml \
  go test -tags=integration ./test/...
```

```bash
GOWORK=off DRS_E2E_CONFIG_FILE=./e2e/drs-config.e2e.sample.yaml \
  go test -tags=e2e ./e2e/...
```

Live-test settings come from `DRS_E2E_CONFIG_FILE`. HTTP route tests derive the base URL from `DrsListenPort` as `http://localhost:<DrsListenPort>`.

For CLI workflows, assume `gocmd` is available on `PATH`.

Prefer `irods-grid-stack` for live-test infrastructure. The older in-repository Docker test framework under `deployments/docker-test-framework/` is a compatibility fixture only.

## Local Multi-Repo Development

Use a workspace `go.work` file for cross-repo development instead of committing local `replace` directives.

Typical workspace at `workspace-gabble/go.work`:

- `./go-irodsclient-extensions`
- `./irods-go-rest`
- `./irods-go-drs`

Workflow:

1. Develop with `go.work` active locally.
2. Keep each repository `go.mod` pinned to real module versions.
3. Tag shared changes in `go-irodsclient-extensions`.
4. Bump dependent repositories with `GOWORK=off go get <module>@<tag-or-commit>`.
5. Run `GOWORK=off go mod tidy` and tests in each dependent repository.

## Release Checklist

For `1.0.0-alpha`:

1. Confirm `go.mod` points to released dependency versions, especially `go-irodsclient-extensions`.
2. Run `GOWORK=off go test ./...`.
3. Run integration and E2E tests against `irods-grid-stack` when the grid is available.
4. Confirm [../CERTIFICATION.md](../CERTIFICATION.md) has `Overall status: PASS`.
5. Review [../README.md](../README.md), [CONFIGURATION_NOTES.md](./CONFIGURATION_NOTES.md), and [../api/swagger.yaml](../api/swagger.yaml) for release alignment.
6. Create a GitHub release tagged `1.0.0-alpha`.
7. Confirm the workflow publishes `ghcr.io/michael-conway/irods-go-drs:1.0.0-alpha`.
8. Update downstream stack defaults or deployment manifests that should consume the released image.

If Go tooling should install this repository by semantic module version, also publish a leading-`v` tag such as `v1.0.0-alpha`. Container release tags may use `1.0.0-alpha` to match deployment naming.

## Known Client Gaps

- S3 access method affinity needs more deployment feedback.
- Checksum and ticket helper workflows should continue moving toward shared extension code when they are not DRS-specific.
- go-irodsclient caching does not yet provide a public no-cache or explicit fresh-read API for path existence/lookups.
