# Docker iRODS grid test/build tools

## Deprecation Notice

The built-in `deployments/docker-test-framework` compose stack is deprecated
for new DRS development. It is retained as a compatibility fixture while local
testing moves to the shared `irods-grid-stack` workspace.

Prefer `irods-grid-stack` for current development:

```bash
cd ../irods-grid-stack
cp .env.example .env
docker compose up -d --build
```

That backend-only command starts iRODS provider/resource services, Keycloak, and
the S3 API endpoints without starting the REST, DRS, or Starbase frontend
services. This is the preferred mode when running `irods-go-drs` or
`irods-go-rest` locally from source.

For a complete demo stack that also starts REST, DRS, and Starbase containers:

```bash
docker compose --profile frontend up -d --build
```

For host-run `irods-go-drs` tests, review `../e2e/drs-config.e2e.sample.yaml`
and set:

```bash
export DRS_E2E_CONFIG_FILE=./e2e/drs-config.e2e.sample.yaml
```

The legacy notes below describe the old in-repository stack and should not be
used as the source of truth for new environment work.

The docker-test-framework subdirectory includes entries for various iRODS versions. Upon selection of a version, the docker-compose up command
can be issued from that subdirectory

e.g.

```

cd docker-test-framework
cd 5-0
docker-compose build
docker-compose up

```

This should start an iRODS server a Docker private network.

The 5-0 stack also starts the iRODS S3 API. Its bucket and user mapping files
are read from a shared directory mounted at `/shared-s3-config` in the S3 API
container. By default, this is `5-0/shared-s3-config`. To use a directory
outside the repository, set:

```
ENV_SHARED_S3_CONFIG=/absolute/path/to/shared-s3-config
```

The directory must contain `irods-s3-bucket-mapping.json` and
`irods-s3-user-mapping.json`.

Note the settings.xml file is mounted that has the correct coordinates for the iRODS grid pre-configured with test accounts, resources, groups, etc as expected by the Jargon unit test framework.
