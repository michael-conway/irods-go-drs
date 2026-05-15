# USERGUIDE

## Purpose

`drscmd` is an administration tool for DRS operations in iRODS.

## How `gocmd` and `drscmd` work together

`gocmd` is the general-purpose iRODS command line interface from CyVerse.
It is the right tool for day-to-day iRODS operations such as:

- listing collections and data objects
- uploading and downloading data
- setting general metadata
- managing tickets, ACLs, and environment files

`drscmd` is narrower. It is intended for DRS-specific administration tasks such as:

- creating a single-object DRS registration for an iRODS data object
- creating a collection-backed compound DRS object
- looking up DRS metadata by iRODS path or DRS id
- listing DRS objects under an iRODS collection
- updating supported DRS metadata fields on an existing DRS object
- removing single-object DRS metadata from an iRODS data object
- stripping DRS metadata from a single object or an entire compound subtree
- adding a starter `.drsignore` file to a collection

The expected workflow is:

1. Use `gocmd` to establish and maintain your normal iRODS environment.
2. Use `drscmd` when you need to administer DRS-specific metadata and behavior.

`drscmd` loads the saved iCommands environment and session information before connecting to iRODS. In practice this means:

- a connection created with `gocmd init` is reused by `drscmd`
- relative iRODS paths are resolved against the saved iCommands current working directory when one exists
- if no current working directory has been saved yet, `drscmd` falls back to the iRODS home directory

For commands that accept either `<path-or-drs-id>` (`drsinfo`, `drsupdate`, and `drsrm`), use `--path` when passing a
bare relative path such as `USERGUIDE.md`. Without an explicit selector, only absolute iRODS paths and values beginning
with `.` are treated as paths; other values are treated as DRS ids.

Successful DRS commands write JSON to stdout. Command errors are written to stderr and return a non-zero exit status.

## Current commands

Implemented command set:

- `iinit`
- `drsinfo`
- `drsmake`
- `drsmakecompound`
- `drsls`
- `drsupdate`
- `drsrm`
- `add_drsignore`

### Initialize environment

```bash
drscmd iinit -h <host> -o <port> -u <user> -z <zone> -p <password> -t native
```

This writes an iRODS environment file that `drscmd` will use for later commands. Note that the 
gocmd init method works in the same manner, so a valid environment established by
gocmd will be recognized by drscmd.

Show command help:

```bash
drscmd iinit --help
```

### Show DRS information

By iRODS path:

```bash
drscmd drsinfo --path /tempZone/home/rods/file.txt
```

By DRS id:

```bash
drscmd drsinfo --id <drs-id>
```

If no explicit selector flag is given, the tool treats values that look like
iRODS paths as paths and everything else as DRS ids. Use `--path` for bare
relative paths:

```bash
drscmd drsinfo --path USERGUIDE.md
```

Show command help:

```bash
drscmd drsinfo --help
```

For compound DRS objects, `drsinfo` includes the generated runtime manifest.
At the API layer, the same compound object resolves from `GET /objects/{id}`
with a direct HTTPS `access_url` pointing to:

```text
/ga4gh/drs/v1/ext/compound/{object_id}
```

Compound object access methods do not require a compound `access_id` lookup hop.

### Create a DRS object

drsmake creates a DRS object from an existing iRODS data object. This is only for a single object. 
For compound objects, utilize the drsmakecompound command.

Note that a compound object can include DRS objects created prior to the compound object being created. 

```bash
drscmd drsmake /tempZone/home/rods/file.txt \
  --description "example object" \
  --alias sample-1 \
  --alias alternate-id
```

Optional flags:

- `--mime-type <type>` to explicitly set the MIME type
- `--description <text>` to store a human-readable description
- repeated `--alias <value>` flags to add alternate identifiers

If `--mime-type` is omitted, the MIME type is derived from the file extension.

Relative iRODS paths are resolved against the saved iCommands current working directory. For example, if `gocmd cd`
or another iCommands-compatible tool has set the cwd to `/tempZone/home/rods/projects/demo`, then:

```bash
drscmd drsmake USERGUIDE.md --description "example object"
```

is resolved as:

```text
/tempZone/home/rods/projects/demo/USERGUIDE.md
```

Show command help:

```bash
drscmd drsmake --help
```

### Create a compound DRS object from a collection

Create compound metadata on an existing collection:

```bash
drscmd drsmakecompound /tempZone/home/rods/projects/compound-a
```

If `.drsignore` is not present at the collection root, `drsmakecompound` stops and requires an explicit override:

```bash
drscmd drsmakecompound /tempZone/home/rods/projects/compound-a --allow-no-ignore
```

Run a no-write preflight to preview the generated manifest from current data and AVUs:

```bash
drscmd drsmakecompound /tempZone/home/rods/projects/compound-a --preflight
```

Use both when you want preflight without a `.drsignore`:

```bash
drscmd drsmakecompound /tempZone/home/rods/projects/compound-a --preflight --allow-no-ignore
```

Show command help:

```bash
drscmd drsmakecompound --help
```

Compound creation writes the root collection DRS id and compound marker, and it
ensures included data objects have DRS ids, MIME type, and version metadata.
It does not create default `iRODS:DRS:DESCRIPTION` or `iRODS:DRS:ALIAS` AVUs
for the root collection, child collections, or data objects. Add or update
descriptions and aliases explicitly when they are needed.

### List DRS objects in a collection

List DRS objects directly under the saved iRODS current working directory:

```bash
drscmd drsls
```

List DRS objects under a specific collection:

```bash
drscmd drsls /tempZone/home/rods/projects/demo
```

List recursively through child collections:

```bash
drscmd drsls --recursive /tempZone/home/rods/projects
```

Paging options:

- `--offset <n>` for a zero-based page offset
- `--limit <n>` for page size
- `--exact_total` to scan all matches and include an exact total

By default, `drsls` uses a fast listing path and may omit `total`. Use
`--exact_total` when the exact total count is required.

Scope options:

- `--scope_all` includes DRS data objects and compound collection objects. This is the default.
- `--scope_objects` includes only DRS data objects.
- `--scope_compound` includes only compound collection objects.

Examples:

```bash
drscmd drsls --scope_objects /tempZone/home/rods/projects
drscmd drsls --scope_compound --recursive /tempZone/home/rods/projects
drscmd drsls --exact_total --limit 50 /tempZone/home/rods/projects
```

The response includes top-level paging fields:

- `path`
- `offset`
- `limit`
- `hasMore`

When `--exact_total` is used, the response also includes `total`.

Each object row includes:

- `drsId`
- `path`
- `isBundle`
- `description`

Show command help:

```bash
drscmd drsls --help
```

### Update DRS metadata on an existing DRS object

Update description by DRS id:

```bash
drscmd drsupdate --id <drs-id> description "updated description"
```

Update MIME type by iRODS path:

```bash
drscmd drsupdate --path /tempZone/home/rods/file.txt mimeType application/json
```

Update version:

```bash
drscmd drsupdate --path /tempZone/home/rods/file.txt version v2
```

Replace the alias set using repeatable `-a` flags:

```bash
drscmd drsupdate --path /tempZone/home/rods/file.txt alias \
  -a sample-2 \
  -a alternate-id-2
```

For alias updates, the provided aliases become the complete alias set. Any existing alias that is not included in the
new `-a/--alias` list is removed.

Supported update items are:

- `mimeType`
- `version`
- `description`
- `alias`

The target must already be a DRS object. If the target path or id does not resolve to an existing DRS object, the
command fails.

Show command help:

```bash
drscmd drsupdate --help
```

### Remove DRS semantics from a DRS object

By iRODS path:

```bash
drscmd drsrm --path /tempZone/home/rods/file.txt
```

Relative path from the saved iRODS current working directory:

```bash
drscmd drsrm --path USERGUIDE.md
```

By DRS id:

```bash
drscmd drsrm --id <drs-id>
```

This removes DRS AVUs from the resolved DRS object path. For a single-object DRS
registration, it strips one data object. For a compound DRS object, it strips
the full collection subtree. It does not delete objects/collections.

Show command help:

```bash
drscmd drsrm --help
```

### Add a sample `.drsignore`

```bash
drscmd add_drsignore /tempZone/home/rods/projects/compound-a
```

This writes the built-in sample `.drsignore` file to the target collection.
If the file already exists, the command fails and leaves the existing file unchanged.

Show command help:

```bash
drscmd add_drsignore --help
```

## Help and usage errors

Each command has its own help screen. The current command-specific help entry points are:

- `drscmd iinit --help`
- `drscmd drsinfo --help`
- `drscmd drsls --help`
- `drscmd drsmake --help`
- `drscmd drsmakecompound --help`
- `drscmd drsupdate --help`
- `drscmd drsrm --help`
- `drscmd add_drsignore --help`

If a command is invoked with a usage error, such as a missing required path or conflicting selector flags, `drscmd`
prints that command's help content before returning the error.

## Notes

- `drscmd` supports both single-object and collection-backed compound DRS workflows.
- `drsmakecompound --preflight` returns a generated manifest preview and performs no AVU writes.
- JSON output is used so the tool is friendly to scripting and automation.
