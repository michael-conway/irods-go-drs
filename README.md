# iRODS DRS Client - GoLang

[![Go](https://github.com/michael-conway/irods-go-drs/actions/workflows/go.yml/badge.svg)](https://github.com/michael-conway/irods-go-drs/actions/workflows/go.yml)
[![CodeQL Advanced](https://github.com/michael-conway/irods-go-drs/actions/workflows/codeql.yml/badge.svg)](https://github.com/michael-conway/irods-go-drs/actions/workflows/codeql.yml)

A Go implementation of the GA4GH Data Repository Service (DRS) for iRODS.

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
| Current Version | `TBD`                                            |
| Status | `Active Development`                             |
| Primary Developer | `Mike Conway`                                    |
| Co-Developer | `Deep Patel`                                     |
| Organization | `NIEHS`                                          |
| Repository | `https://github.com/michael-conway/irods-go-drs` |
| Contact | `mike.conway@nih.gov`                            |
| Issue Tracker | `https://github.com/michael-conway/irods-go-drs/issues` |
| License | `TBD`                                            |

## Master Index

* [DRS Console User Guide](./tools/drs-console/USERGUIDE.md)
* [Configuration Notes](./CONFIGURATION_NOTES.md)
* [Development Notes](./DEV_NOTES.md)

## Project Structure

The repository follows a conventional Go layout centered around a generated-and-customized REST service, iRODS/DRS support packages, and developer tooling.

| Path | Purpose |
| --- | --- |
| `main.go` | Service entrypoint for the DRS REST API |
| `internal/` | HTTP handlers, routing, generated models, OpenAPI-serving endpoints, and service implementation details |
| `drs-support/` | DRS-to-iRODS mapping logic, manifest support, validation helpers, and configuration support |
| `tools/drs-console/` | `drscmd` command-line tool for DRS administration |
| `api/` | OpenAPI source documents embedded and served by the service |
| `config/` | Sample runtime configuration including `drs-config.yaml` and `service-info.json` |
| `test/` | Unit tests plus integration-tagged functional tests |
| `deployments/` | Development and integration test deployment assets, including docker-based test environments |

## Stack and Testing Strategy

The implementation is written in Go and uses a generated Swagger/OpenAPI server foundation with project-specific routing, handlers, and iRODS integration layered on top. The CLI is built in Go as well and is designed to work alongside `gocmd` for iRODS environment and session management.

Testing is organized in layers:

* unit tests run by default with `go test ./...`
* integration tests are opt-in with the `integration` build tag
* live CLI functional tests use the built `drscmd` binary and a reachable iRODS test environment

For CLI-centered development, `gocmd` should be installed and on `PATH` so that `drscmd` can consume the saved iCommands-compatible environment and session state.

## DRS Console

This repository includes a DRS administration command line tool at [`tools/drs-console`](./tools/drs-console).

`drscmd` is intended to work alongside CyVerse `gocmd`:

* use `gocmd` for general iRODS operations and environment/session management
* use `drscmd` for DRS-specific administration such as `drsinfo`, `drsmake`, and `drsrm`

GoCommands resources:

* CyVerse GoCommands repository: https://github.com/cyverse/gocommands
* CyVerse GoCommands user guide: https://learning.cyverse.org/ds/gocommands/

See the [DRS Console User Guide](./tools/drs-console/USERGUIDE.md) for usage and workflow details.

## API Documentation

When the REST service is running locally on the default port, the API documentation is available at:

* Swagger UI: `http://localhost:8080/swagger`
* OpenAPI spec: `http://localhost:8080/openapi.yaml`

## References

* GA4GH Data Repository Service (DRS): https://ga4gh.github.io/data-repository-service-schemas/
* OpenAPI Specification: https://github.com/OAI/OpenAPI-Specification
* Swagger Codegen: https://github.com/swagger-api/swagger-codegen
* Standard Go project layout: https://github.com/golang-standards/project-layout
* go-irodsclient: https://github.com/cyverse/go-irodsclient
* urfave/cli: https://github.com/urfave/cli
* urfave/cli docs: https://cli.urfave.org/v3/getting-started/
* Viper: https://github.com/spf13/viper
* Zerolog: https://github.com/rs/zerolog
* slog logging guide: https://betterstack.com/community/guides/logging/logging-in-go/
* Go OIDC library: https://github.com/coreos/go-oidc
* Keycloak: https://www.keycloak.org/documentation
* Keycloak quickstarts: https://github.com/keycloak/keycloak-quickstarts/
* Gorilla Mux: https://github.com/gorilla/mux
* Gorilla Mux docs: https://gorilla.github.io/mux/
* GoCloak: https://github.com/Nerzal/gocloak
* Go contexts: https://golang.org/pkg/context/