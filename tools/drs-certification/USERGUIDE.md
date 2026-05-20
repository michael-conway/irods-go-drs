# drs-certification

`drscert` prepares an iRODS-backed DRS test corpus and generates a
`drs-compliance-suite` configuration for self-testing `irods-go-drs`.

The tool expects the same shared DRS YAML config used by E2E tests. Pass it
with `--drs-config`, or set `DRS_E2E_CONFIG_FILE` or `DRS_CONFIG_FILE`.

## Prepare

```bash
go run ./tools/drs-certification prepare \
  --drs-config ./e2e/drs-config.e2e.sample.yaml \
  --server-base-url http://localhost:8888/ga4gh/drs/v1 \
  --output-dir .certification/drs \
  --report-path CERTFICATION.md
```

`prepare` creates a corpus under:

```text
/<zone>/home/<primary-test-user>/drs-certification/<run-id>
```

It writes:

- `corpus.json`
- `drs-compliance-config.json`

The generated compliance-suite config includes:

- valid Basic-auth object and access checks
- a compound object for manifest retrieval
- invalid DRS id and invalid access id checks
- invalid Basic auth checks

If `--bearer-token-file <path>` is provided, the tool reads a bearer token
from the file, strips a leading `Bearer ` prefix if present, and also adds:

- valid Bearer-auth object checks for each generated DRS object
- a Bearer-auth access check for the primary object
- an invalid Bearer-auth check

Without `--bearer-token-file`, the generated config only exercises Basic auth.
The generated compliance-suite config contains the bearer token, so write it to
an ignored artifact directory.

## Run

```bash
go run ./tools/drs-certification run \
  --output-dir .certification/drs \
  --suite-bin ../drs-compliance-suite/.venv/bin/drs-compliance-suite \
  --report-path CERTFICATION.md
```

`run` writes:

- `CERTFICATION.md`
- `run.json`

When commands are run from the repository root, the default report path is
`CERTFICATION.md`, which places the compliance summary at the top level for CI.
When running from `tools/`, pass `--report-path ../CERTFICATION.md`.

## All

```bash
go run ./tools/drs-certification all \
  --drs-config ./e2e/drs-config.e2e.sample.yaml \
  --server-base-url http://localhost:8888/ga4gh/drs/v1 \
  --suite-bin ../drs-compliance-suite/.venv/bin/drs-compliance-suite \
  --report-path CERTFICATION.md
```

`all` runs `prepare` and `run`. It does not clean up the corpus.

Add `--bearer-token-file ./path/to/token` to `prepare` or `all` to include the
Bearer-auth coverage described above.

## Cleanup

```bash
go run ./tools/drs-certification cleanup \
  --corpus .certification/drs/corpus.json
```

Cleanup removes the iRODS corpus root recorded in `corpus.json`.
