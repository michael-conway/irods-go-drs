# Configuration Notes

## Configuration of the Docker compose framework

The docker compose framework is in the [deployments](./deployments/docker-test-framework/5-0) directory.

Configuration files in that directory should be set up in a private area.

### keycloak.env

This env file is used to configure the keycloak server. Copy this file to a private area where you can provision
authentication. The current version uses Google Auth and OAuth2.0 for authentication, so the appropriate keys should be
added to the file.

Once configured the file location can be passed to compose via the an environment variable:

```
KEYCLOAK_ENV_FILE=/path/to/keycloak.env
```

This should be set prior to running the compose command.

## Configuration for running the DRS API (for runtime and for running integration tests)

The [docker-compose](deployments/docker-test-framework/5-0) runs a compact iRODS, Postgres, and Keycloak framework that is
used for development and testing. This is not meant to be a production deployment but can be used to document the various
configuration points.

* The iRODS server runs a catalog provider with a set of test users.
* Keycloak creates a realm and client based on the keycloak.env configuration passed in (see above about the keycloak.env)
* Postgres is configured to back both Keycloak and iRODS.
* TBD will probably add a generic REST api that can support login and general iRODS operations used in DRS.


### drs-config.yaml

The drs-config.yaml file is used to provide configuration to the DRS implementation itself. This includes configuration
of iRODS connections and behaviors, configuration of Authn/Authz (i.e. Keycloak bits for auth), as well as tuning of the 
behavior of the DRS api.