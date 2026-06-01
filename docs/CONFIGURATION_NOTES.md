# Configuration Notes

This is the runtime configuration reference for `irods-go-drs`.

## Sources And Precedence

The service reads `drs-config.yaml` by default. To use one exact file, set:

```bash
DRS_CONFIG_FILE=/path/to/drs-config.yaml
```

Search paths without `DRS_CONFIG_FILE`:

1. paths passed by the caller
2. `/etc/irods-ext/`
3. `$HOME/.irods-drs`
4. current working directory

Environment variables override config-file values. Secret-file settings are used only when the corresponding explicit secret value is empty. Relative secret paths, service-info paths, and local access root paths resolve relative to the config file.

## Common Runtime Settings

```bash
DRS_LISTEN_PORT=8080
DRS_DRS_LOG_LEVEL=info
DRS_PUBLIC_URL=http://localhost:8080
DRS_HTTP_READ_TIMEOUT_SECONDS=30
DRS_HTTP_READ_HEADER_TIMEOUT_SECONDS=30
DRS_HTTP_WRITE_TIMEOUT_SECONDS=60
DRS_HTTP_IDLE_TIMEOUT_SECONDS=120
```

`DRS_PUBLIC_URL` is the externally visible origin used for `self_uri` and compound extension `access_url` generation. It must be an `http` or `https` origin without path, query, or fragment.

Transport timeout defaults are applied when unset or non-positive.

## iRODS Connection

```bash
DRS_IRODS_HOST=irods-provider
DRS_IRODS_PORT=1247
DRS_IRODS_ZONE=tempZone
DRS_IRODS_ADMIN_USER=rods
DRS_IRODS_ADMIN_PASSWORD=rods
DRS_IRODS_ADMIN_LOGIN_TYPE=native
DRS_IRODS_AUTH_SCHEME=native
DRS_IRODS_DEFAULT_RESOURCE=providerResc
DRS_IRODS_NEGOTIATION_POLICY=CS_NEG_REQUIRE
```

`IrodsAdminLoginType` controls the admin/proxy account used by bearer-token and ticket-backed requests. `IrodsAuthScheme` controls direct user credentials, including Basic auth requests.

PAM auth requires SSL in go-irodsclient. If the iRODS server returns `CS_NEG_REFUSE`, use native auth for that connection path or enable SSL negotiation on the iRODS server.

## iRODS SSL

For SSL-configured zones:

```yaml
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

`VerifyServer` accepts `hostname`, `cert`, or `none`. Use `hostname` for production verification.

## OIDC

```bash
DRS_OIDC_URL=https://keycloak:8443
DRS_OIDC_REALM=drs
DRS_OIDC_CLIENT_ID=irods-go-drs
DRS_OIDC_CLIENT_SECRET=secret
DRS_OIDC_SCOPE="openid profile email"
DRS_OIDC_INSECURE_SKIP_VERIFY=false
```

`DRS_OIDC_URL` is the issuer URL used by DRS for token validation. In `irods-grid-stack`, use `https://keycloak:8443` for containerized DRS and `https://localhost:8443` for host-run DRS.

For self-signed local Keycloak certificates, use `DRS_OIDC_INSECURE_SKIP_VERIFY=true` only in development. `DRS_OIDC_SKIP_TLS_VERIFY` is still accepted for compatibility; `DRS_OIDC_INSECURE_SKIP_VERIFY` is preferred.

## Access Methods

```yaml
IrodsAccessMethodSupported: true
FileAccessMethodSupported: false
HttpsAccessMethodSupported: true
HttpsAccessImplementation: irods-go-rest
HttpsAccessMethodBaseURL: /api/v1/path/contents?irods_path=
HttpsAccessUseTicket: true
DefaultTicketLifetimeMinutes: 720
DefaultTicketUseLimit: 50
LocalAccessRootPath: /mnt/irods
S3AccessMethodSupported: true
S3AccessMethodBaseURL: s3://
```

Current behavior:

- `https` returns an `access_id` for atomic objects and a direct `access_url` for compound objects.
- `irods` returns an `access_id`.
- `local` returns a `local:///...` URL when enabled.
- `s3` returns an inline `s3://bucket/key` URL for objects under an ancestor collection with an `iRODS:S3:Bucket` AVU.

Supported HTTPS implementations:

- `irods-go-rest`
- `irods-https-api`

## Resource Affinity

`HttpsResourceAffinity` maps iRODS storage resources to HTTPS DRS hosts that are proximate to those resources:

```yaml
HttpsResourceAffinity:
  - Host: https://drs-provider.example.org
    Resources:
      - providerResc
  - Host: https://drs-default.example.org
    Resources: []
```

Exact resource names are preferred for matching replicas. The first entry with an empty `Resources` list is the default for unmatched resources. `*` is accepted for backward compatibility.

`S3ResourceAffinity` follows the same shape for S3 URL host selection when S3 access methods are enabled.

## Service Info

Keep service-info metadata in JSON and point DRS at it:

```yaml
ServiceInfoFilePath: service-info.json
ServiceInfoSampleIntervalMinutes: 5
```

Environment equivalent:

```bash
DRS_SERVICE_INFO_FILE_PATH=/path/to/service-info.json
DRS_SERVICE_INFO_SAMPLE_INTERVAL_MINUTES=5
```

If `ServiceInfoFilePath` is relative, it is resolved relative to `drs-config.yaml`.

## Secrets

Prefer secret files over inline secrets in container deployments:

```yaml
IrodsAdminPasswordFile: /run/secrets/irods_admin_password
OidcClientSecretFile: /run/secrets/oidc_client_secret
```

Environment equivalents:

```bash
DRS_IRODS_ADMIN_PASSWORD_FILE=/run/secrets/irods_admin_password
DRS_OIDC_CLIENT_SECRET_FILE=/run/secrets/oidc_client_secret
```

Secret resolution order:

1. explicit value, such as `IrodsAdminPassword` or `DRS_IRODS_ADMIN_PASSWORD`
2. secret file
3. empty value

## Test Settings

Integration and E2E tests share `DRS_E2E_CONFIG_FILE`.

Useful test-only keys:

```yaml
IrodsPrimaryTestUser: test1
IrodsPrimaryTestPassword: test
IrodsSecondaryTestUser: test2
IrodsSecondaryTestPassword: test
```

Use [../e2e/drs-config.e2e.sample.yaml](../e2e/drs-config.e2e.sample.yaml) with `irods-grid-stack` as the starting point for host-run live tests.

Do not use the legacy YAML keys:

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

## Container Pattern

Mount config and secrets separately:

```bash
DRS_CONFIG_FILE=/config/drs-config.yaml
DRS_IRODS_ADMIN_PASSWORD_FILE=/run/secrets/irods_admin_password
DRS_OIDC_CLIENT_SECRET_FILE=/run/secrets/oidc_client_secret
```

For `irods-grid-stack`, prefer setting DRS environment through that stack's `.env` and mounted config files rather than editing this repository's sample config for deployment-specific values.
