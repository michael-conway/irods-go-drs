# Development Notes



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

CI split (recommended).

* Job 1: unit tests on every push/PR.
* Job 2: start docker compose, then run integration tests with -tags=integration.
