# Configuration Notes

Use this file as the quick reference for `irods-go-drs` configuration.

## Config sources

The service reads configuration in this order:

1. `drs-config.yaml`
2. `DRS_*` environment variable overrides
3. Secret files for sensitive values

To use one exact config file, set:

```bash
DRS_CONFIG_FILE=/path/to/drs-config.yaml
```

## Main runtime settings

These are the settings you will usually care about:

```bash
DRS_LISTEN_PORT=8080
DRS_DRS_LOG_LEVEL=info

DRS_IRODS_HOST=irods-provider
DRS_IRODS_PORT=1247
DRS_IRODS_ZONE=tempZone
DRS_IRODS_ADMIN_USER=rods
DRS_IRODS_PRIMARY_TEST_USER=test1
DRS_IRODS_PRIMARY_TEST_PASSWORD=test1
DRS_IRODS_SECONDARY_TEST_USER=test2
DRS_IRODS_SECONDARY_TEST_PASSWORD=test2

DRS_OIDC_URL=https://localhost:8443
DRS_OIDC_REALM=drs
DRS_OIDC_CLIENT_ID=irods-go-drs
DRS_OIDC_INSECURE_SKIP_VERIFY=false
```

If your local Keycloak uses a self-signed certificate, you can temporarily use:

```bash
DRS_OIDC_INSECURE_SKIP_VERIFY=true
```

Use that only for local development.

In YAML config files, use:

```yaml
OidcInsecureSkipVerify: true
```

`OidcSkipTLSVerify` is still accepted for compatibility, but
`OidcInsecureSkipVerify` is the preferred config key.

## Secrets

Prefer secret files over inline secrets.

Supported file-backed secret settings:

```yaml
IrodsAdminPasswordFile: /run/secrets/irods_admin_password
OidcClientSecretFile: /run/secrets/oidc_client_secret
```

Environment variable equivalents:

```bash
DRS_IRODS_ADMIN_PASSWORD_FILE=/run/secrets/irods_admin_password
DRS_OIDC_CLIENT_SECRET_FILE=/run/secrets/oidc_client_secret
```

Secret precedence is:

1. explicit value
2. secret file
3. empty

## Test user settings

For integration and E2E work, keep the test users in the same config file:

```yaml
IrodsAdminUser: rods
IrodsAdminPasswordFile: /run/secrets/irods_admin_password
IrodsPrimaryTestUser: test1
IrodsPrimaryTestPassword: test1
IrodsSecondaryTestUser: test2
IrodsSecondaryTestPassword: test2
```

The test helpers use proxy authentication through `IrodsAdminUser` and
`IrodsAdminPassword`, and they default the effective test user to
`IrodsPrimaryTestUser`.

If you add Basic-authenticated E2E tests, use `IrodsPrimaryTestPassword` and
`IrodsSecondaryTestPassword` as the source of truth for those user credentials.

Do not use the old YAML keys:

```yaml
IrodsDrsAdminUser:
IrodsDrsAdminPassword:
IrodsDrsAdminPasswordFile:
```

Use:

```yaml
IrodsAdminUser:
IrodsAdminPassword:
IrodsAdminPasswordFile:
```

## Access methods

Configured access methods are still partly stubbed. Supported names are:

- `http`
- `irods`
- `local`
- `s3`

Example:

```yaml
AccessMethods:
  - http
  - irods
  - local
HTTPAccessBaseURL: https://drs.example.org
IRODSAccessHost: irods.example.org
IRODSAccessPort: 1247
LocalAccessRootPath: /mnt/irods
```

Current behavior:

- `http` returns an `access_id`
- `irods` returns an `access_id`
- `local` returns a `local:///...` path
- `s3` is a placeholder

## Service-info JSON

You can keep service-info metadata in a separate JSON file:

```yaml
ServiceInfoFilePath: service-info.json
ServiceInfoSampleIntervalMinutes: 5
```

Environment variable equivalent:

```bash
DRS_SERVICE_INFO_FILE_PATH=/path/to/service-info.json
```

If the path is relative, it is resolved relative to `drs-config.yaml`.

## Docker test framework

The local Docker test stack is under:

```text
deployments/docker-test-framework/5-0
```

This is for development and testing, not production.

If you keep a private `keycloak.env` outside the repo, point Compose at it with:

```bash
KEYCLOAK_ENV_FILE=/path/to/keycloak.env
```
