# Development Notes

## API Docs and Swagger

When the DRS REST service is running, the embedded Swagger UI is available at `/swagger`.

The raw OpenAPI document served by the service is available at `/openapi.yaml`.

For a default local startup on port `8080`, that means:

* Swagger UI: `http://localhost:8080/swagger`
* OpenAPI spec: `http://localhost:8080/openapi.yaml`

## iRODS Conventions

### DRS Objects

A DRS object always maps to an iRODS data object, never to an iRODS collection. Collections may still be useful for
organization, but they are not addressable as DRS objects in this implementation.

The creation of a DRS object is accomplished by decorating an iRODS data object with AVU metadata. The current AVU
scheme in `drs_support` is:

* `iRODS:DRS:ID` - unique identifier for the DRS object.
* `iRODS:DRS:VERSION` - version of the DRS object. If absent, the implementation may fall back to the checksum value.
* `iRODS:DRS:MIME_TYPE` - mime type of the DRS object.
* `iRODS:DRS:DESCRIPTION` - description of the DRS object.
* `iRODS:DRS:ALIAS` - alternate identifiers for the DRS object.
* `iRODS:DRS:COMPOUND_MANIFEST` - marker indicating that the data object content is a DRS manifest.

Metadata AVU unit: `iRODS:DRS`

The DRS metadata layer is intentionally shallow. It records identity and descriptive metadata in AVUs, but does not
serialize compound membership or object trees into AVU values.

### DRS Compound Objects

A compound object is also represented by an iRODS data object. Specifically, it is a generated JSON manifest file that
has its own DRS ID and is marked with `iRODS:DRS:COMPOUND_MANIFEST`.

Compound Objects design:

* each compound object manifest is stored as a normal iRODS data object
* the manifest file itself is the DRS object for that compound object
* the manifest content is JSON and contains child DRS IDs plus optional relationship metadata such as `name` or `role`
* a child may be either a standard iRODS-backed DRS object or another manifest-backed DRS object
* nesting is therefore expressed by manifest files pointing to other manifest files by DRS ID
* manifest files do not point back to parents, because multiple compound objects may reuse the same child manifest or data object

This means compound membership is determined by parsing the manifest file bytes, not by reading AVUs. The AVU marker
only answers the question “is this object a manifest?”

Validator and traversal behavior should follow that model:

* for a non-compound object, validate checksum, size, and created/modified timestamps against observed iRODS state and update metadata if needed
* for a compound object, read and parse the manifest JSON, validate the manifest structure, then recursively descend through child DRS IDs
* broken manifest integrity should be recorded in a report, not treated as a fatal exception path

### DRS Ruleset?

Consider iRODS policies for DRS, including versioning on data object change, immuntability? DRS object validation, 
such as scans for missing DRS objects in bundles, etc?


## Testing

### Development and Test Environment

For live functional testing of the DRS console and related workflows, the local development environment should include:

* a reachable iRODS test environment, such as the docker compose stack under `deployments/docker-test-framework/5-0`
* valid iRODS test credentials
* `gocmd` installed and available on `PATH`

The `gocmd` requirement is intentional. Current development and manual functional test flows assume that `gocmd` can be
used to initialize and manage the iCommands-compatible environment and session state that `drscmd` will later consume.
If a test or harness depends on relative iRODS path resolution or the saved iRODS cwd, it should assume that `gocmd`
has already been used to establish that state.

### Unit versus Integration Testing

Use two layers: default unit tests, opt-in integration tests.
Keep unit tests as normal *_test.go files.

* No tags.
*Fast, isolated, run with go test ./....

Mark integration tests with a build tag. Put them in files like *_integration_test.go.

Add at top of each file:

```
//go:build integration
// +build integration
```

Run only when requested:

```
go test -tags=integration ./...

```

Use `DRS_TEST_BEARER_TOKEN` for integration tests that need an Authorization header. The shared test helper will attach
`Authorization: Bearer <token>` automatically when that environment variable is set. Tests that require a bearer token
should call the helper that skips when the token is missing.

CI split (recommended).

* Job 1: unit tests on every push/PR.
* Job 2: start docker compose, then run integration tests with -tags=integration.
