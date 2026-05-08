# .drsignore Guide

`.drsignore` controls which paths are included during **compound DRS object
creation/bootstrap**.

Important scope rule:

- `.drsignore` is applied during creation/bootstrap only.
- Runtime compound manifest generation is AVU-driven and does not evaluate
  `.drsignore`.

This behavior keeps bootstrap selection deterministic while allowing metadata to
evolve over time through AVU edits.
objec
## Planned command usage

 `drscmd` workflow:

- `drscmd add_drsignore --path <collection>`

Expected behavior for that command:

1. verify target path is a collection
2. check `.drsignore` does not already exist in that collection
3. write the built-in sample `.drsignore` template

The sample template content is embedded in `drs-support` so it is
programmatically available.

## Syntax summary (gitignore-style)

Rules follow gitignore-like matching semantics over iRODS logical paths:

- blank lines are ignored
- lines beginning with `#` are comments
- `\#` matches a literal leading `#`
- trailing spaces are ignored unless escaped
- `!pattern` negates a previous ignore
- `\!` matches a literal leading `!`
- `/` is path separator
- trailing `/` means directory-only
- `*` matches non-`/` characters
- `?` matches one non-`/` character
- `[a-z]` character classes are supported
- `**` supports recursive matching forms:
  - `**/foo`
  - `abc/**`
  - `a/**/b`
- last matching rule wins

## Behavior notes

- During compound creation, preflight checks for existing descendant compound
  objects run before ignore evaluation.
- This prevents ignore rules from masking invalid nested compound structures.
- After bootstrap, AVU state is authoritative for manifest output.

## Example patterns

See the embedded sample template:

- `drs-support/templates/sample.drsignore`

