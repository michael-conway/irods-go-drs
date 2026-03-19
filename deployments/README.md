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

Note the settings.xml file is mounted that has the correct coordinates for the iRODS grid pre-configured with test accounts, resources, groups, etc as expected by the Jargon unit test framework.

