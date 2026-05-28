# iRODS DRS Client - GoLang

[![Go](https://github.com/michael-conway/irods-go-drs/actions/workflows/go.yml/badge.svg)](https://github.com/michael-conway/irods-go-drs/actions/workflows/go.yml)
[![CodeQL Advanced](https://github.com/michael-conway/irods-go-drs/actions/workflows/codeql.yml/badge.svg)](https://github.com/michael-conway/irods-go-drs/actions/workflows/codeql.yml)
[![Certification Report](https://github.com/michael-conway/irods-go-drs/actions/workflows/certification-report.yml/badge.svg)](https://github.com/michael-conway/irods-go-drs/actions/workflows/certification-report.yml)


A Go implementation of the GA4GH Data Repository Service (DRS) for iRODS.

Note that the DRS certification as described in the CERTIFICATION.md is based on proposed self-certification test based
on the [GA4GH DRS Test Suite](https://github.com/ga4gh/data-repository-service-tests) and does not reflect any official GA4GH DRS certification.

## Overview

This project provides a DRS-oriented service layer for exposing iRODS-backed content through the GA4GH DRS model, along with command-line and Go package tooling for administering DRS metadata and validating DRS behavior in development and test environments.

It includes:

* a REST service for serving DRS API endpoints
* Go packages for mapping DRS concepts onto iRODS data objects
* a `drscmd` command-line tool for DRS administration workflows
* unit, integration, and live functional test support

## Project Metadata

| Field | Value                                            |
| --- |--------------------------------------------------|
| Project Name | iRODS DRS Client - GoLang                        |
| Current Version | `Unreleased Alpha`                               |
| Status | `Alpha`                                          |
| Primary Developer | `Mike Conway`                                    |
| Co-Developer | `Deep Patel`                                     |
| Organization | `NIEHS`                                          |
| Repository | `https://github.com/michael-conway/irods-go-drs` |
| Contact | `mike.conway@nih.gov`                            |
| Issue Tracker | `https://github.com/michael-conway/irods-go-drs/issues` |
| License | `BSD-2-Clause`                                   |

## Master Index

* [Documentation Directory](./docs/)
* [Configuration Notes](./docs/CONFIGURATION_NOTES.md)
* [Development Notes](./docs/DEVELOPER_NOTES.md)
* [`drscmd` Tool User Guide](./tools/drs-console/USERGUIDE.md)
* [DRS Certification Tool User Guide](./tools/drs-certification/USERGUIDE.md)

## Project Structure

The repository follows a conventional Go layout centered around a generated-and-customized REST service, iRODS/DRS support packages, and developer tooling.

| Path | Purpose |
| --- | --- |
| `main.go` | Service entrypoint for the DRS REST API |
| `internal/` | HTTP handlers, routing, generated models, OpenAPI-serving endpoints, and service implementation details |
| `drs-support/` | DRS-to-iRODS mapping logic, manifest support, validation helpers, and configuration support |
| [`docs/`](./docs/) | Project notes, design docs, release checklist, and DRS ignore documentation |
| [`tools/drs-console/`](./tools/drs-console/) | [`drscmd`](./tools/drs-console/USERGUIDE.md) command-line tool for DRS administration |
| [`tools/drs-certification/`](./tools/drs-certification/) | [DRS certification tool](./tools/drs-certification/USERGUIDE.md) for preparing a compliance-test corpus and report |
| `api/` | OpenAPI source documents embedded and served by the service |
| `config/` | Sample runtime configuration including `drs-config.yaml` and `service-info.json` |
| `test/` | Broader integration tests that span packages and run against a reachable iRODS test grid |
| `e2e/` | End-to-end HTTP and workflow tests that run against a reachable DRS service and iRODS test grid |
| `deployments/` | Legacy docker-test-framework assets retained during migration to `irods-grid-stack` |

## Stack and Testing Strategy

The implementation is written in Go and uses a generated Swagger/OpenAPI server foundation with project-specific routing, handlers, and iRODS integration layered on top. The CLI is built in Go as well and is designed to work alongside `gocmd` for iRODS environment and session management.

Testing is organized in layers:

* unit tests live next to the code they validate and run by default with `go test ./...`
* integration tests live under `test/` and are opt-in with the `integration` build tag
* end-to-end tests live under `e2e/` and are opt-in with the `e2e` build tag
* live CLI functional tests use the built `drscmd` binary and a reachable iRODS test environment

The preferred local test environment is now
[`irods-grid-stack`](https://github.com/michael-conway/irods-grid-stack). The
legacy compose files under `deployments/docker-test-framework/` are deprecated
and should be treated as compatibility fixtures while DRS and REST development
workflows move to the shared grid stack.

Use `irods-grid-stack` in one of two modes:

* backend-only: run `docker compose up -d --build` from `irods-grid-stack` to
  start iRODS provider/resource, Keycloak, and S3 API services, then run
  `irods-go-drs` or `irods-go-rest` locally from source
* full stack: run `docker compose --profile frontend up -d --build` from
  `irods-grid-stack` to also start REST, DRS, and Starbase containers

For host-run `irods-go-drs` integration or E2E tests, point
`DRS_E2E_CONFIG_FILE` at a host-facing config such as
`./e2e/drs-config.e2e.sample.yaml` after reviewing local ports and credentials.
That sample is aligned with the default `irods-grid-stack` ports and resource
names.

For OIDC, keep URL context consistent with where DRS runs:

* containerized DRS in `irods-grid-stack`: `DRS_OIDC_URL=https://keycloak:8443`
* host-run DRS against grid-stack: `DRS_OIDC_URL=https://localhost:8443`

For CLI-centered development, `gocmd` should be installed and on `PATH` so that `drscmd` can consume the saved iCommands-compatible environment and session state.

## DRS Console

This repository includes a DRS administration command line tool at [`tools/drs-console`](./tools/drs-console).

`drscmd` is intended to work alongside CyVerse `gocmd`:

* use `gocmd` for general iRODS operations and environment/session management
* use `drscmd` for DRS-specific administration such as `drsinfo`, `drsls`, `drsmake`, `drsupdate`, and `drsrm`

GoCommands resources:

* CyVerse GoCommands repository: https://github.com/cyverse/gocommands
* CyVerse GoCommands user guide: https://learning.cyverse.org/ds/gocommands/

See the [DRS Console User Guide](./tools/drs-console/USERGUIDE.md) for usage and workflow details.

## API Documentation

When the REST service is running locally on the default port, the API documentation is available at:

* Swagger UI: `http://localhost:8080/swagger`
* OpenAPI spec: `http://localhost:8080/openapi.yaml`

### iRODS extension endpoints

The following endpoints are **iRODS-specific extensions**. They are not part of the GA4GH DRS base API contract.

| Endpoint | Purpose |
| --- | --- |
| `GET /ga4gh/drs/v1/ext/compound/{object_id}` | Generate and return the runtime compound manifest JSON for a collection-backed compound DRS object. This is the internal HTTPS target used by compound-object `access_url` entries. |

### Compound object access behavior

For a compound DRS object, `GET /ga4gh/drs/v1/objects/{object_id}` returns a direct HTTPS `access_url` in the access method.
It does not require an `access_id` round-trip through `/objects/{object_id}/access/{access_id}`.

Example shape:

```json
{
  "id": "compound-drs-id",
  "name": "/tempZone/home/test1/compound-root",
  "access_methods": [
    {
      "type": "https",
      "access_url": {
        "url": "https://<drs-host>/ga4gh/drs/v1/ext/compound/compound-drs-id"
      },
      "authorizations": {
        "supported_types": [
          "BasicAuth",
          "BearerAuth"
        ]
      }
    }
  ]
}
```

### Passport / Bulk POST status

`irods-go-drs` alpha currently does **not** support Passport-based DRS POST flows.
The following endpoints are intentionally disabled and return `501 Not Implemented`:

* `POST /ga4gh/drs/v1/objects/{object_id}`
* `POST /ga4gh/drs/v1/objects`
* `POST /ga4gh/drs/v1/objects/{object_id}/access/{access_id}`
* `POST /ga4gh/drs/v1/objects/access`

## References

* GA4GH Data Repository Service (DRS): https://ga4gh.github.io/data-repository-service-schemas/
* go-irodsclient: https://github.com/cyverse/go-irodsclient
* Viper: https://github.com/spf13/viper
* Zerolog: https://github.com/rs/zerolog
* slog logging guide: https://betterstack.com/community/guides/logging/logging-in-go/
* Go OIDC library: https://github.com/coreos/go-oidc
* Keycloak: https://www.keycloak.org/documentation
* GoCloak - https://github.com/Nerzal/gocloak
* Gorilla Mux: https://github.com/gorilla/mux

### S3 API

iRODS s3 api docker images - https://hub.docker.com/r/irods/irods_s3_api/tags
iRODS s3 config - https://github.com/irods/irods_client_s3_api#configuration
iRODS s3 dockerhub - https://hub.docker.com/r/irods/irods_s3_api
