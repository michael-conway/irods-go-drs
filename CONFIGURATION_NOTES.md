# Configuration Notes

## Configuration of the Docker compose framework

The docker compose framework is in the [deployments](./deployments/docker-test-framework/5-0) directory.

Configuration files in that directory should be set up in a private area. This docker compose framework is not meant to
be a production deployment, it is a harness for development and integration testing.

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

The loader supports three configuration layers:

1. A YAML configuration file such as `drs-config.yaml`
2. Environment variable overrides with the `DRS_` prefix
3. File-backed secrets for sensitive values

If you want to point the service at one specific config file and skip all search paths, set:

```bash
DRS_CONFIG_FILE=/path/to/drs-config.yaml
```

When `DRS_CONFIG_FILE` is set, it overrides the default config search locations and that exact file is used.

If `DRS_CONFIG_FILE` is not set, the loader looks for `drs-config.yaml` using its normal search paths and then applies
environment variable overrides on top of the file values.

Examples of supported environment variable overrides:

```bash
DRS_LISTEN_PORT=8080
DRS_IRODS_HOST=irods-provider
DRS_IRODS_PORT=1247
DRS_IRODS_ZONE=tempZone
DRS_IRODS_DRS_ADMIN_USER=rods
DRS_OIDC_URL=https://keycloak.example.org
DRS_OIDC_REALM=drs
DRS_OIDC_CLIENT_ID=irods-go-drs
DRS_DRS_LOG_LEVEL=debug
```

The DRS server listen port is configured through `DrsListenPort` in `drs-config.yaml` or `DRS_LISTEN_PORT` in the
environment. If it is omitted, the service defaults to port `8080`.

### Service info JSON configuration

The DRS service-info endpoint can be backed by a site-specific JSON document. This allows deployers to keep local
metadata such as service name, description, organization, contact URL, documentation URL, environment, and other
static GA4GH DRS service-info fields in a separate JSON file instead of hardcoding them in the service.

The JSON file is configured from `drs-config.yaml` with:

```yaml
ServiceInfoSampleIntervalMinutes: 5
ServiceInfoFilePath: service-info.json
```

`ServiceInfoFilePath` points to the JSON file that should be loaded by the `ServiceInfoSampler` when the service starts.
On each sampling interval, the sampler reads that JSON, updates fields that are maintained by the service, and stores
the resulting JSON payload for the `/ga4gh/drs/v1/service-info` response.

At the moment, the sampler fills placeholder values for dynamic fields under the `drs` section such as:

```json
{
  "drs": {
    "objectCount": 0,
    "totalObjectSize": 0
  }
}
```

This keeps the site-specific metadata in JSON while leaving room for the service to inject live values later.

`ServiceInfoFilePath` may be absolute or relative:

* If it is absolute, that path is used directly.
* If it is relative, it is resolved relative to the directory containing `drs-config.yaml`.

For example, with the checked-in sample config:

```yaml
ServiceInfoFilePath: service-info.json
```

and:

```text
config/
  drs-config.yaml
  service-info.json
```

the loader resolves the JSON file to the sibling file next to `drs-config.yaml`.

The equivalent environment variable override is:

```bash
DRS_SERVICE_INFO_FILE_PATH=/path/to/service-info.json
```

See the sample file at [config/service-info.json](./config/service-info.json) for the expected shape.

For secrets, prefer file-backed values over putting secrets directly in YAML or plain environment variables.

The loader supports:

```yaml
IrodsDrsAdminPasswordFile: /path/to/irods-admin-password.txt
OidcClientSecretFile: /path/to/oidc-client-secret.txt
```

and the matching environment variables:

```bash
DRS_IRODS_DRS_ADMIN_PASSWORD_FILE=/path/to/irods-admin-password.txt
DRS_OIDC_CLIENT_SECRET_FILE=/path/to/oidc-client-secret.txt
```

Direct secret values are still supported, but the effective precedence is:

1. Explicit secret value from environment or YAML
2. Secret file path from environment or YAML
3. Empty value if neither is provided

The following test fixtures show the expected file layout for file-backed secrets:

```text
drs-config-secret-files.yaml
irods-admin-password.txt
oidc-client-secret.txt
```

The `drs-config-secret-files.yaml` fixture points at `irods-admin-password.txt` and `oidc-client-secret.txt`, and the
loader reads and trims those files at startup. This is the preferred pattern for Docker or Kubernetes-style mounted
secrets.

## Sample Bearer Token

```json
{
  "exp": 1776788002,
  "iat": 1776787702,
  "auth_time": 1776787702,
  "jti": "onrtac:8427a08e-4129-8040-b483-7e2d24d42a34",
  "iss": "https://localhost:8443/realms/drs",
  "sub": "4b603570-0b59-4adc-ade7-493ea8d56493",
  "typ": "Bearer",
  "azp": "irods-go-rest",
  "sid": "65yhGlh1ynSCUBF9rZ5KQ-Ch",
  "acr": "1",
  "allowed-origins": [
    "http://localhost:8080"
  ],
  "scope": "openid profile email",
  "email_verified": false,
  "name": "test1 test",
  "preferred_username": "test1",
  "given_name": "test1",
  "family_name": "test",
  "email": "test1@irods.org"
}
```
