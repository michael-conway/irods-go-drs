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
- looking up DRS metadata by iRODS path or DRS id
- removing single-object DRS metadata from an iRODS data object

The expected workflow is:

1. Use `gocmd` to establish and maintain your normal iRODS environment.
2. Use `drscmd` when you need to administer DRS-specific metadata and behavior.

`drscmd` loads the saved iCommands environment and session information before connecting to iRODS. In practice this means:

- a connection created with `gocmd init` is reused by `drscmd`
- relative iRODS paths are resolved against the saved iCommands current working directory when one exists
- if no current working directory has been saved yet, `drscmd` falls back to the iRODS home directory

## Current commands

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
iRODS paths as paths and everything else as DRS ids.

Show command help:

```bash
drscmd drsinfo --help
```

### Create a DRS object

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

### Remove a single-object DRS registration

```bash
drscmd drsrm /tempZone/home/rods/file.txt
```

This removes the DRS AVUs from the data object but does not delete the data object itself.

Show command help:

```bash
drscmd drsrm --help
```

## Help and usage errors

Each command has its own help screen. The current command-specific help entry points are:

- `drscmd iinit --help`
- `drscmd drsinfo --help`
- `drscmd drsmake --help`
- `drscmd drsrm --help`

If a command is invoked with a usage error, such as a missing required path or conflicting selector flags, `drscmd`
prints that command's help content before returning the error.

## Notes

- `drscmd` currently manages single-object DRS registrations.
- Compound manifest administration can be added later as dedicated commands.
- JSON output is used so the tool is friendly to scripting and automation.
