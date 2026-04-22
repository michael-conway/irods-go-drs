# Development Notes

## iRODS Conventions

### DRS Objects

A DRS Object is a blob, and maps to an iRODS data object. The creation of a DRS object is accomplished by
decorating the iRODS data object with AVU metadata. 

Metadata AVU Attributes:

* `DRS_OBJECT_ID` - unique identifier for the DRS object. This will be a standard GUID by default.
* `DRS_OBJECT_TYPE` - mime type of the DRS object (optional).
* `DRS_OBJECT_VERSION` - version of the DRS object (optional). Will be the Hash by default.
* `DRS_OBJECT_DESCRIPTION` - Description of the DRS object (optional).


Metadata AVU Unit: `DRS_OBJECT_AVU`


## Testing

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
