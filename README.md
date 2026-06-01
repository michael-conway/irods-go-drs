# iRODS Go DRS

[![Go](https://github.com/michael-conway/irods-go-drs/actions/workflows/go.yml/badge.svg)](https://github.com/michael-conway/irods-go-drs/actions/workflows/go.yml)
[![CodeQL Advanced](https://github.com/michael-conway/irods-go-drs/actions/workflows/codeql.yml/badge.svg)](https://github.com/michael-conway/irods-go-drs/actions/workflows/codeql.yml)
[![Certification Report](https://github.com/michael-conway/irods-go-drs/actions/workflows/certification-report.yml/badge.svg)](https://github.com/michael-conway/irods-go-drs/actions/workflows/certification-report.yml)

`irods-go-drs` is an alpha GA4GH Data Repository Service (DRS) implementation for iRODS. It maps iRODS data objects and collections into DRS objects, exposes DRS HTTP endpoints, and includes command-line tooling for DRS metadata administration and certification testing.

## Status

| Field | Value |
| --- | --- |
| Release | `1.0.0-alpha` |
| Stability | Alpha |
| DRS API version | `1.5.0` |
| Certification report | `PASS` self-certification report in [CERTIFICATION.md](./CERTIFICATION.md) |
| License | `BSD-2-Clause` |
| Repository | `https://github.com/michael-conway/irods-go-drs` |
| Issues | `https://github.com/michael-conway/irods-go-drs/issues` |

The certification report is based on the GA4GH DRS test suite and project-local self-certification workflow. It is not an official GA4GH certification.

## Quick Start

Run from source:

```bash
go run .
```

Then open:

- `http://localhost:8080/swagger`
- `http://localhost:8080/openapi.yaml`
- `http://localhost:8080/ga4gh/drs/v1/service-info`

Use a specific config file with:

```bash
DRS_CONFIG_FILE=/path/to/drs-config.yaml go run .
```

Runtime configuration details live in [docs/CONFIGURATION_NOTES.md](./docs/CONFIGURATION_NOTES.md).

## DRS Model

- Atomic DRS objects map to iRODS data objects.
- Compound DRS objects map to iRODS collections marked with DRS AVUs.
- DRS metadata is stored as AVUs on iRODS collections and data objects.
- Compound manifest payloads are generated at request time.
- Checksum and version data come from iRODS state when available.

Current DRS AVUs:

- `iRODS:DRS:ID`
- `iRODS:DRS:VERSION`
- `iRODS:DRS:MIME_TYPE`
- `iRODS:DRS:DESCRIPTION`
- `iRODS:DRS:ALIAS`
- `iRODS:DRS:COMPOUND_MANIFEST`

## API Notes

Core DRS routes follow the GA4GH DRS API shape:

- `GET /ga4gh/drs/v1/service-info`
- `GET /ga4gh/drs/v1/objects/{object_id}`
- `GET /ga4gh/drs/v1/objects/{object_id}/access/{access_id}`

Project-specific extension route:

- `GET /ga4gh/drs/v1/ext/compound/{object_id}`

Compound objects return a direct HTTPS `access_url` to the compound extension route. Atomic objects return `access_id` entries for later access resolution.

Passport and bulk POST flows are intentionally unsupported in alpha and return `501 Not Implemented`:

- `POST /ga4gh/drs/v1/objects/{object_id}`
- `POST /ga4gh/drs/v1/objects`
- `POST /ga4gh/drs/v1/objects/{object_id}/access/{access_id}`
- `POST /ga4gh/drs/v1/objects/access`

See [api/swagger.yaml](./api/swagger.yaml) for the OpenAPI contract.

## DRS Console

The `drscmd` tool in [tools/drs-console](./tools/drs-console) supports DRS administration workflows such as:

- `drsinfo`
- `drsls`
- `drsmake`
- `drsupdate`
- `drsrm`

It is intended to work alongside CyVerse `gocmd` for iRODS environment and session management. See [tools/drs-console/USERGUIDE.md](./tools/drs-console/USERGUIDE.md).

## Local Test Stack

Use [`irods-grid-stack`](https://github.com/michael-conway/irods-grid-stack) for live iRODS, Keycloak, REST, S3 API, and Starbase workflows.

Backend-only grid:

```bash
cd ../irods-grid-stack
cp .env.example .env
docker compose up -d --build
```

Full frontend stack:

```bash
docker compose --profile frontend up -d --build
```

For host-run integration or E2E tests, point `DRS_E2E_CONFIG_FILE` at a host-facing config:

```bash
export DRS_E2E_CONFIG_FILE=./e2e/drs-config.e2e.sample.yaml
```

## Tests

Unit tests:

```bash
GOWORK=off go test ./...
```

Direct integration tests:

```bash
GOWORK=off DRS_E2E_CONFIG_FILE=./e2e/drs-config.e2e.sample.yaml \
  go test -tags=integration ./test/...
```

HTTP E2E tests:

```bash
GOWORK=off DRS_E2E_CONFIG_FILE=./e2e/drs-config.e2e.sample.yaml \
  go test -tags=e2e ./e2e/...
```

## Container Images

Build locally:

```bash
docker build -t irods-go-drs:local .
```

Published images use GitHub Container Registry:

```text
ghcr.io/michael-conway/irods-go-drs:<tag>
```

The container workflow publishes branch tags, SHA tags, `latest` for the default branch, and release tags. Publishing the GitHub release `1.0.0-alpha` publishes:

```text
ghcr.io/michael-conway/irods-go-drs:1.0.0-alpha
```

## Developer References

- [docs/DEVELOPER_NOTES.md](./docs/DEVELOPER_NOTES.md) - implementation rules, testing layers, and release checklist
- [docs/CONFIGURATION_NOTES.md](./docs/CONFIGURATION_NOTES.md) - runtime configuration reference
- [CERTIFICATION.md](./CERTIFICATION.md) - current self-certification report
- [docs/DRS_IGNORE.md](./docs/DRS_IGNORE.md) - `.drsignore` behavior for compound objects
- [tools/drs-certification/USERGUIDE.md](./tools/drs-certification/USERGUIDE.md) - certification tool guide
- [go-irodsclient](https://github.com/cyverse/go-irodsclient)
- [go-irodsclient-extensions](https://github.com/michael-conway/go-irodsclient-extensions)
