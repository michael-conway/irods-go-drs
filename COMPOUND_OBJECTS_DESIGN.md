# DRS Compound Objects Design (Plan v4)


## 1) Goal

Support DRS compound objects in `irods-go-drs` using iRODS collections as the
compound root and AVUs as the source of structure and metadata.

For this phase:

- no persisted manifest payload document
- runtime manifest is generated from current AVUs
- `.drsignore` is used only during compound creation/bootstrap

## 2) Core model

- A compound DRS object is an iRODS collection (may contain nested
  subcollections).
- The root compound collection is marked using
  `iRODS:DRS:COMPOUND_MANIFEST` (`iRODS:DRS` unit).
- Root collection has a DRS ID (existing `iRODS:DRS:ID` AVU model).
- Subcollections are semantic manifest nodes using DRS alias/description AVUs.
- Intermediate subcollections do not receive DRS IDs (`drs_id: null` in
  manifest).
- Included data objects are expected to be DRS objects with IDs.
- Non-DRS AVUs are preserved as metadata at each manifest node.

## 3) Design details

1. Compound marker AVU: reuse existing `iRODS:DRS:COMPOUND_MANIFEST`.
2. Root already compound on create: fail strict.
3. Descendant compound conflict: fail if any descendant is compound.
4. Scaffold scope on create: root + included subcollections + included data objects.
5. `.drsignore` runtime behavior: creation/bootstrap only.
6. Runtime inclusion policy: include DRS-tagged nodes plus required structural parents.
7. Intermediate nodes: include with `drs_id: null`; include non-DRS AVUs too.
8. Manifest order: preserve iRODS `fs.ListEntries` ordering.
9. Non-DRS AVUs: include all on included nodes.
10. Existing data object DRS IDs: always preserve.
11. Snapshot staleness handling in create flow: refresh snapshot once before write phase.
12. Data objects without DRS ID during create: auto-assign IDs only if not `.drsignore` excluded.
13. Subcollection DRS IDs: never assign to intermediate subcollections.
14. Root DRS ID at create: auto-generate if missing.
15. Creation write strategy: best-effort write, return per-node errors.
16. Runtime missing/corrupt AVUs: return partial manifest + warnings.
17. Manifest identity fields: include iRODS context and root DRS ID.
18. Manifest node type model: explicit `collection` / `data_object`.
19. Runtime child data object selection: include only objects with DRS IDs.
20. Warning surface: top-level `warnings[]` only.

## 4) Creation-time optimization and ordering

Creation workflow uses a tree snapshot in memory:

- Build one unfiltered tree snapshot first (collections, data objects, AVUs).
- Check for existing descendant compound markers before ignore pass.
- If any descendant compound exists, fail immediately.
- Only after conflict check, evaluate `.drsignore` for bootstrap/tagging scope.

This guarantees ignore rules cannot mask an invalid nested compound object.

## 5) Creation workflow (planned)

1. Validate root path exists and is a collection.
2. Build full tree snapshot from root.
3. Conflict check: fail if any descendant collection has compound marker.
4. Root setup:
   - ensure root has DRS ID (auto-generate if missing)
   - apply root compound marker AVU
   - apply skeleton alias/description AVUs
5. Load and preprocess root `.drsignore`.
6. Select included collections/data objects using ignore engine.
7. Apply scaffolding to included nodes:
   - subcollections: alias/description skeleton (initialize to relative path)
   - data objects: create DRS IDs where missing (only if included)
8. Refresh snapshot once before writes (staleness guard).
9. Execute best-effort AVU writes.
10. Return success + per-node error report payload.

## 6) Runtime manifest generation (planned)

At request time:

1. Resolve compound root by DRS ID.
2. Traverse hierarchy and query AVUs on collections/data objects.
3. Build nested manifest from current AVUs (no `.drsignore` application).
4. Include:
   - root iRODS context: host, port, zone, absolute path
   - root DRS ID
   - explicit node type for each node
   - mapped DRS fields (`drs_id`, `aliases`, `description`)
   - all additional non-DRS AVUs (`attribute`, `value`, `unit`)
5. Include top-level `warnings[]` for incomplete/corrupt nodes.
6. Return partial manifest when recoverable issues exist.

## 7) Inclusion rules summary

Creation/bootstrap:

- `.drsignore` controls which nodes are scaffolded/tagged.
- Included data objects should carry DRS IDs.

Runtime manifest:

- Based on AVUs as they currently exist.
- Only DRS-ID-bearing data objects are emitted.
- Subcollections can be emitted without DRS IDs (`drs_id: null`) when they
  carry semantic DRS metadata (alias/description) or are required structural
  parents.

## 8) Runtime warning shape

Top-level warnings list only:

```json
{
  "warnings": [
    {
      "path": "/tempZone/home/test1/compound/subA",
      "code": "missing_drs_id",
      "message": "collection node has DRS alias/description but no DRS ID"
    }
  ]
}
```

## 9) Implementation sequencing 

1. Define AVU contract constants and validation helpers.
2. Add tree snapshot builder (path, kind, avus, children).
3. Add create preflight conflict scanner (pre-ignore).
4. Add create bootstrap flow with ignore integration and best-effort reporting.
5. Add runtime manifest assembler and warning collector.
6. Add tests:
   - nested compound conflict before ignore
   - ignore-scaffold behavior
   - ID preservation and assignment rules
   - runtime AVU-driven manifest with warnings

