# drs-certification

`drscert` prepares an iRODS-backed DRS test corpus and generates a
`drs-compliance-suite` configuration for self-testing `irods-go-drs`.

The tool expects the same shared DRS YAML config used by E2E tests. Pass it
with `--drs-config`, or set `DRS_E2E_CONFIG_FILE` or `DRS_CONFIG_FILE`.
The config must include iRODS connection settings, the admin credentials used
to create the corpus, and the primary test user credentials used for Basic-auth
compliance checks.

The suggested certification report path is `CERTIFICATION.md` at the repository
root. The GitHub certification workflow checks that file and fails if the report
is missing or its `**Overall status:**` summary is not passing.

## Prepare

#### The following commands are set up from the top level directory of irods-go-drs.
f
```bash
go run ./tools/drs-certification/drscert.go prepare \
  --drs-config ./e2e/drs-config.e2e.sample.yaml \
  --output-dir .certification/drs
```

`prepare` creates a corpus under:

```text
/<zone>/home/<primary-test-user>/drs-certification/<run-id>
```

It writes:

- `corpus.json`
- `drs-compliance-config.json`

`corpus.json` records the generated compliance config path, corpus root,
effective iRODS user, and generated DRS object IDs.

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

Example with Bearer-auth coverage:

```bash
go run ./tools/drs-certification/drscert.go prepare \
  --drs-config ./e2e/drs-config.e2e.sample.yaml \
  --output-dir .certification/drs \
  --bearer-token-file .certification/bearer-token.txt
```

## Compliance Suite Environment - Running with Python

Create a Python virtual environment for the sibling `drs-compliance-suite`
checkout and install its requirements before running certification.

Check out the drs-compliance suite from this location (this is currently a fork of the GA4GH compliance tool updated to 1.5):

https://github.com/michael-conway/drs-compliance-suite

The paths in the example below assume that irods-go-drs and the drs-compliance-suite are under the same parent directory,
and that the user is running the commands from the parent directory.

```bash
python3 -m venv .venv
source .venv/bin/activate
cd ./drs-compliance-suite
python -m pip install --upgrade pip
python -m pip install -r requirements.txt
python -m pip install -e .
cd ..
```

`drscert` calls the compliance suite with the `--version` argument, so use the
`drs-compliance-suite` fork/version that supports:

```text
--version {1.2.0,1.3.0,1.5.0}
```

The installed executable is:

```text
./.venv/bin/drs-compliance-suite
```

## Run

#### The following commands are set up from the parent directory of irods-go-drs and drs-compliance-suite.

```bash
cd irods-go-drs 
PYTHONPATH=../drs-compliance-suite  \
go run ./tools/drs-certification/drscert.go run \
  --output-dir .certification/drs \
  --server-base-url http://localhost:8888/ga4gh/drs/v1 \
  --suite-bin ../.venv/bin/drs-compliance-suite \
  --version 1.5.0 \
  --report-path CERTIFICATION.md
```

`run` writes:

- `CERTIFICATION.md`
- `run.json`

The DRS server base URL and report path are run-phase settings. They are taken
from `--server-base-url` and `--report-path`; neither is stored in `corpus.json`
or `run.json`.

When commands are run from the repository root, the default report path is
`CERTIFICATION.md`, which places the compliance summary at the top level for CI.
When running from `tools/`, pass `--report-path ../CERTIFICATION.md`. When
running from `tools/drs-certification/`, pass
`--report-path ../../CERTIFICATION.md`.

## All

```bash
go run ./tools/drs-certification/drscert.go all \
  --drs-config ./e2e/drs-config.e2e.sample.yaml \
  --server-base-url http://localhost:8888/ga4gh/drs/v1 \
  --suite-bin ../.venv/bin/drs-compliance-suite \
  --version 1.5.0 \
  --report-path CERTIFICATION.md
```

`all` runs `prepare` and `run`. It does not clean up the corpus.

Add `--bearer-token-file ./path/to/token` to `prepare` or `all` to include the
Bearer-auth coverage described above.

## Cleanup

```bash
go run ./tools/drs-certification/drscert.go cleanup \
  --corpus .certification/drs/corpus.json
```

Cleanup removes the iRODS corpus root recorded in `corpus.json`.
