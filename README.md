# iRODS DRS Client - GoLang

A GoLang based implementation of the GA4GH Data Repository Service (DRS)

## drscmd

This repository now includes a DRS administration command line tool at [tools/drs-console](/Users/conwaymc/Documents/workspace-gabble/irods-go-drs/tools/drs-console).

`drscmd` is intended to work alongside CyVerse `gocmd`:

- use `gocmd` for general iRODS operations and environment management
- use `drscmd` for DRS-specific administration such as `drsinfo`, `drsmake`, and `drsrm`

Usage and workflow notes are documented in [USERGUIDE.md](/Users/conwaymc/Documents/workspace-gabble/irods-go-drs/tools/drs-console/USERGUIDE.md).


# docs and links

* standard Go project layout - https://github.com/golang-standards/project-layout
* client/console apps in go - https://github.com/urfave/cli
* cli docs - https://cli.urfave.org/v3/getting-started/
* logging with slog - https://betterstack.com/community/guides/logging/logging-in-go/
* Go for OIDC - https://github.com/coreos/go-oidc
* Keycloak quickstart - https://github.com/keycloak/keycloak-quickstarts/
* Gorilla Mux - https://github.com/gorilla/mux - router framework
* Gorilla Mux docs - https://gorilla.github.io/mux/
* Using Keycloak with Gorilla Mux - https://mikebolshakov.medium.com/keycloak-with-go-web-services-why-not-f806c0bc820a
* GoCloak - https://github.com/Nerzal/gocloak
* go contexts - https://golang.org/pkg/context/


