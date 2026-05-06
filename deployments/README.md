# Docker iRODS grid test/build tools

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
