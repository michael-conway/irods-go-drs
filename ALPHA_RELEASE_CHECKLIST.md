# Alpha Release Checklist

This checklist defines the minimum hardening, API alignment, and operational readiness for an alpha release of `irods-go-drs`.

## P0 - Must Complete Before Alpha

- [x] API contract and implementation alignment for core object/access endpoints.
  - `POST /objects/{object_id}`
  - `POST /objects`
  - `POST /objects/{object_id}/access/{access_id}`
  - `POST /objects/access`
  - Acceptance: every operation in `api/swagger.yaml` is either implemented or explicitly documented as unsupported with consistent status behavior.
  - Status: these Passport/bulk POST endpoints are explicitly unsupported in alpha and return `501 Not Implemented` consistently.

- [x] Document and publish extension endpoints clearly.
  - Include `/ga4gh/drs/v1/ext/compound/{object_id}` in extension docs.
  - Acceptance: clients can distinguish GA4GH base endpoints from iRODS-specific extensions.
  - Status: published in `README.md` and OpenAPI (`api/swagger.yaml`) as an iRODS-specific extension endpoint with explicit purpose.

- [x] Compound object behavior/docs consistency pass.
  - Compound `GET /objects/{id}` response uses direct `access_url` for compound manifest access.
  - Acceptance: README, user guide, and API docs/examples match runtime behavior.
  - Status: updated in `README.md`, `tools/drs-console/USERGUIDE.md`, and `api/swagger.yaml`; e2e coverage added for compound `GET /objects/{id}` direct `access_url` and `/ext/compound/{id}` retrieval.

- [x] Documentation correctness pass.
  - Remove placeholder metadata (`Current Version`, `License`).
  - Fix stale/broken links and naming mismatches.
  - Acceptance: docs are internally consistent and accurate for current commands and routes.
  - Status: replaced README placeholders, set license metadata to BSD-2-Clause, and fixed stale `DEV_NOTES.md` references to `DEVELOPER_NOTES.md`.

- [x] Developer notes refresh.
  - Update compound model statements to match current collection-backed behavior.
  - Update access method semantics to match current code behavior.
  - Acceptance: `DEVELOPER_NOTES.md` is authoritative for current architecture.
  - Status: `DEVELOPER_NOTES.md` updated for collection-backed compound model, AVU-derived runtime manifest behavior, direct compound `access_url`, and current alpha endpoint support.

- [ ] S3 access method hardening.
  - Resolve current TODOs around S3 access method behavior (including affinity expectations).
  - Acceptance: S3 method generation behavior is deterministic and documented.

- [ ] Dependency and reproducible build hygiene.
  - Ensure pinned module versions exist remotely.
  - Ensure `go.mod`/`go.sum` are CI-clean with workspace disabled.
  - Acceptance: CI passes with `GOWORK=off` and no local `replace` assumptions.

## P1 - Strongly Recommended Before Alpha

- [x] Expand e2e coverage for compound workflows.
  - Create compound object
  - Read DRS object
  - Read compound manifest via extension endpoint
  - Strip/remove DRS semantics
  - Verify `.drsignore` exclusion behavior
  - Acceptance: green e2e run for compound happy paths and key edge cases.
  - Status: compound workflow coverage added in `e2e/object_e2e_test.go` and `e2e/object_fixture_test.go` (including `.drsignore` exclusion and strip semantics). Full `-tags=e2e` execution is currently blocked in this environment without `DRS_E2E_CONFIG_FILE`.

- [x] Expand access method e2e/API tests.
  - `https`, `irods`, and `s3` access method generation and resolution
  - Basic/Bearer auth behavior
  - Ticketed URL/URI behavior where configured
  - Acceptance: each enabled access method has passing positive/negative tests.

- [ ] Response semantics normalization.
  - Use consistent status policy for unsupported endpoints/features.
  - Avoid placeholder success responses with empty payloads.
  - Acceptance: unsupported behavior is explicit and predictable to clients.

- [ ] Observability baseline.
  - Structured request logs include route, object id/access id, auth mode, and error class.
  - Acceptance: production debugging can trace failures without code changes.

- [ ] iRODS operation safety/timeouts.
  - Add and verify timeout/cancellation handling in long-running iRODS calls on request path.
  - Acceptance: service does not hang indefinitely on backend latency/failures.

## P2 - Post-Alpha (Planned Hardening)

- [ ] Publish compatibility matrix.
  - iRODS versions
  - auth modes
  - S3 integration support level
  - known limits

- [ ] Security policy maturity update.
  - Define alpha support window, disclosure expectations, and update cadence.

- [ ] Release process checklist.
  - Module bump sequence for multi-repo work
  - `go.sum` refresh policy
  - smoke test matrix before tag/cut

## Exit Criteria for Alpha

- [ ] All P0 items complete.
- [ ] No known critical-severity defects in object resolution or access method generation.
- [ ] CI build/test green from a clean checkout with workspace mode off.
- [ ] Primary docs and CLI guide accurately describe shipped behavior.
