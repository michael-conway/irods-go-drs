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
DRS_RESOURCE_AFFINITY=demoResc,edgeResc

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

## iRODS authentication and SSL

`IrodsAdminLoginType` controls the admin/proxy account used by bearer-token and
ticket-backed requests. `IrodsAuthScheme` controls direct user credentials, such
as Basic auth requests.

For PAM Basic auth, set `IrodsAuthScheme: pam`. go-irodsclient requires SSL for
PAM accounts, and DRS applies the same connection settings to Basic, bearer, and
ticket accounts.

```yaml
IrodsAdminLoginType: native
IrodsAuthScheme: pam
IrodsNegotiationPolicy: CS_NEG_REQUIRE
IrodsSSLConfig:
  CACertificateFile: /etc/irods/ca.pem
  CACertificatePath:
  EncryptionKeySize: 32
  EncryptionAlgorithm: AES-256-CBC
  EncryptionSaltSize: 8
  EncryptionNumHashRounds: 16
  VerifyServer: hostname
  DHParamsFile:
  ServerName: irods.example.org
```

Environment equivalents:

```bash
DRS_IRODS_ADMIN_LOGIN_TYPE=native
DRS_IRODS_AUTH_SCHEME=pam
DRS_IRODS_NEGOTIATION_POLICY=CS_NEG_REQUIRE
DRS_IRODS_SSL_CA_CERTIFICATE_FILE=/etc/irods/ca.pem
DRS_IRODS_SSL_CA_CERTIFICATE_PATH=
DRS_IRODS_ENCRYPTION_KEY_SIZE=32
DRS_IRODS_ENCRYPTION_ALGORITHM=AES-256-CBC
DRS_IRODS_ENCRYPTION_SALT_SIZE=8
DRS_IRODS_ENCRYPTION_NUM_HASH_ROUNDS=16
DRS_IRODS_SSL_VERIFY_SERVER=hostname
DRS_IRODS_SSL_DH_PARAMS_FILE=
DRS_IRODS_SSL_SERVER_NAME=irods.example.org
```

`VerifyServer` accepts `hostname`, `cert`, or `none`. Use `hostname` for normal
production verification.

## Resource affinity

`ResourceAffinity` is optional and maps iRODS storage resources to HTTPS DRS
hosts that are proximate to those resources.

Supported forms:

```yaml
ResourceAffinity:
  - Host: https://drs-resc-a.example.org
    Resources:
      - demoResc
      - cacheResc
  - Host: https://drs-default.example.org
    Resources: []
```

or environment override:

```bash
DRS_RESOURCE_AFFINITY=demoResc,edgeResc
```

Notes:

- `resources` entries with exact names are preferred for matching replicas.
- The first entry with an empty `Resources` array is the default for unmatched resources.
- `*` is still accepted for backward compatibility.
- Environment override remains a legacy compatibility path and maps to one
  default affinity entry using `HttpsAccessMethodBaseURL` as the host base URL.

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

Configured access methods are now driven by structured booleans and provider
settings in `drs-config.yaml`.

Example:

```yaml
IrodsAccessMethodSupported: false
FileAccessMethodSupported: false
HttpsAccessMethodSupported: true
HttpsAccessImplementation: irods-go-rest
HttpsAccessMethodBaseURL: https://drs.example.org/api/v1/path/contents?irods_path=
HttpsAccessUseTicket: true
LocalAccessRootPath: /mnt/irods
S3AccessMethodSupported: true
S3AccessEndpoint: http://127.0.0.1:9001
S3AccessBucket: tempzone
S3AccessIrodsCollection: /tempZone/home
S3AccessRegion: us-east-1
```

Current behavior:

- `https` returns an `access_id` for later resolution through `/access`
- `irods` returns an `access_id`
- `local` returns a `local:///...` path
- `s3` returns a direct `s3://bucket/key` URL using the configured temporary
  bucket-to-iRODS collection mapping

Current `https` implementations:

- `irods-go-rest` is supported
- `irods-https-api` is supported

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
