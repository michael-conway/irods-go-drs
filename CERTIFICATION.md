# DRS Compliance Report

**Overall status:** `PASS`
Pass was with warnings shown below.

## Target

- Testbed: `DRS Compliance Suite`
- Testbed version: `v1.0.5`
- Platform: `irods-go-drs`
- Platform description: `iRODS-backed GA4GH DRS implementation`

## Inputs

- server_base_url: `http://localhost:8888/ga4gh/drs/v1`
- version: `1.5.0`

## Summary

| Passed | Failed | Warned | Skipped | Unknown |
| --- | --- | --- | --- | --- |
| 200 | 0 | 18 | 0 | 0 |

## API Coverage Highlights

> Coverage gaps in this section are informational unless an individual test case failed.

### Optional Capability Matrix

| Capability | Status | Notes |
| --- | --- | --- |
| Access ID URL resolution | `Supported` | /objects/fb7ff2eb-53c4-4d0e-af6d-456fb6358565/access/irods-go-rest-https-providerResc returned 200<br>/objects/fb7ff2eb-53c4-4d0e-af6d-456fb6358565/access/irods returned 200<br>/objects/fb7ff2eb-53c4-4d0e-af6d-456fb6358565/access/irods-go-rest-https-providerResc returned 200 |
| Bulk access POST | `Not supported` | POST /objects/access returned 501<br>Deprecated or optional capability |
| Bulk authorization OPTIONS | `Supported` | OPTIONS /objects returned 200 for 3 object IDs |
| Bulk object POST | `Not supported` | POST /objects returned 501<br>Deprecated or optional capability |
| Bundle expand | `No sample` | No configured DRS object was marked is_bundle = true<br>Deprecated or optional capability |
| Compound manifest retrieval | `Supported` | Retrieved json compound manifest from http://localhost:8888/ga4gh/drs/v1/ext/compound/d5dd7d58-68ed-407b-83b6-ffd8ee6400da with status code 200 |
| Object authorization OPTIONS | `Supported` | OPTIONS /objects/fb7ff2eb-53c4-4d0e-af6d-456fb6358565 returned 200<br>OPTIONS /objects/fb7ff2eb-53c4-4d0e-af6d-456fb6358565 returned 200<br>OPTIONS /objects/1138f628-78bd-40f2-aa36-7ab53cdc345e returned 200 |
| Passport access POST | `No sample` | No configured DRS access sample used auth_type passport with a discovered access_id |
| Passport object POST | `No sample` | No configured DRS object used auth_type passport |

### Auth Coverage Matrix

| Auth Type | Object Requests | Access Requests | Authorization Metadata | Access Method Metadata | Note |
| --- | --- | --- | --- | --- | --- |
| basic | `yes` | `yes` | `yes` | `yes` | basic auth was encountered |
| bearer | `yes` | `yes` | `yes` | `yes` | bearer auth was encountered |
| passport | `no` | `no` | `no` | `no` | passport auth was not encountered in configured requests or DRS metadata |

### Compound Manifest Support

- Supported: `yes`
- Manifest types observed: `json`

Sample manifest:

```json
// Sample compound manifest returned by the DRS server
{
  "host": "irods-provider",
  "manifest": {
    "alias": ".",
    "children": [
      {
        "alias": "ignored",
        "children": [
          {
            "drsId": "",
            "nodeType": "data_object",
            "path": "/tempZone/home/test1/drs-certification/1780320672997639000/compound-root/ignored/ignored.txt",
            "relativePath": "ignored/ignored.txt",
            "willAssignDrsId": true
          }
        ],
        "description": "ignored",
        "drsId": "",
        "nodeType": "collection",
        "path": "/tempZone/home/test1/drs-certification/1780320672997639000/compound-root/ignored",
        "relativePath": "ignored"
      },
      {
        "alias": "included",
        "children": [
          {
            "drsId": "dbc8edc1-708d-41b5-90b1-0daec36a6b04",
            "nodeType": "data_object",
            "path": "/tempZone/home/test1/drs-certification/1780320672997639000/compound-root/included/child.txt",
            "relativePath": "included/child.txt"
          }
        ],
        "description": "included",
        "drsId": "",
        "nodeType": "collection",
        "path": "/tempZone/home/test1/drs-certification/1780320672997639000/compound-root/included",
        "relativePath": "included"
      }
    ],
    "description": ".",
    "drsId": "d5dd7d58-68ed-407b-83b6-ffd8ee6400da",
    "nodeType": "collection",
    "path": "/tempZone/home/test1/drs-certification/1780320672997639000/compound-root",
    "relativePath": ""
  },
  "port": 1247,
  "rootDrsId": "d5dd7d58-68ed-407b-83b6-ffd8ee6400da",
  "rootPath": "/tempZone/home/test1/drs-certification/1780320672997639000/compound-root",
  "zone": "tempZone"
}
```

## Phases

### service info

**Status:** `PASS`

run all the tests for service_info endpoint

| Passed | Failed | Warned | Skipped | Unknown |
| --- | --- | --- | --- | --- |
| 3 | 0 | 0 | 0 | 0 |

#### Run test cases on the service-info endpoint; auth_type = basic

**Status:** `PASS`

validate service-info status code, content-type and response schemas

| Case | Status | Message |
| --- | --- | --- |
| Service Info response status code validation | `PASS` | Response status code is 200 |
| Service Info response content-type validation | `PASS` | Content-Type matches expected type |
| Service Info response schema validation | `PASS` | Schema Validation Successful |

### service info semantics

**Status:** `PASS`

run DRS 1.5.0-specific checks on service-info

| Passed | Failed | Warned | Skipped | Unknown |
| --- | --- | --- | --- | --- |
| 4 | 0 | 0 | 0 | 0 |

#### Validate DRS 1.5.0 service-info metadata

**Status:** `PASS`

validate DRS service type and bulk capability metadata

| Case | Status | Message |
| --- | --- | --- |
| service-info type.group | `PASS` | service-info type.group is org.ga4gh |
| service-info type.artifact | `PASS` | service-info type.artifact is drs |
| service-info type.version | `PASS` | type.version is 1.5.0 |
| DRS maxBulkRequestLength metadata | `PASS` | maxBulkRequestLength is 1000 |

### authorization discovery

**Status:** `PASS`

run optional DRS 1.5.0 OPTIONS authorization discovery checks

| Passed | Failed | Warned | Skipped | Unknown |
| --- | --- | --- | --- | --- |
| 33 | 0 | 0 | 0 | 0 |

#### Run DRS 1.5.0 authorization discovery for drs id = fb7ff2eb-53c4-4d0e-af6d-456fb6358565

**Status:** `PASS`

validate optional OPTIONS /objects/{object_id} authorization metadata

| Case | Status | Message |
| --- | --- | --- |
| DRS Object Authorizations response status code validation | `PASS` | Authorization discovery metadata was returned |
| DRS Object Authorizations response content-type validation | `PASS` | Content-Type matches expected type |
| DRS Object Authorizations response schema validation | `PASS` | Schema Validation Successful |

#### Run DRS 1.5.0 authorization discovery for drs id = fb7ff2eb-53c4-4d0e-af6d-456fb6358565

**Status:** `PASS`

validate optional OPTIONS /objects/{object_id} authorization metadata

| Case | Status | Message |
| --- | --- | --- |
| DRS Object Authorizations response status code validation | `PASS` | Authorization discovery metadata was returned |
| DRS Object Authorizations response content-type validation | `PASS` | Content-Type matches expected type |
| DRS Object Authorizations response schema validation | `PASS` | Schema Validation Successful |

#### Run DRS 1.5.0 authorization discovery for drs id = 1138f628-78bd-40f2-aa36-7ab53cdc345e

**Status:** `PASS`

validate optional OPTIONS /objects/{object_id} authorization metadata

| Case | Status | Message |
| --- | --- | --- |
| DRS Object Authorizations response status code validation | `PASS` | Authorization discovery metadata was returned |
| DRS Object Authorizations response content-type validation | `PASS` | Content-Type matches expected type |
| DRS Object Authorizations response schema validation | `PASS` | Schema Validation Successful |

#### Run DRS 1.5.0 authorization discovery for drs id = 1138f628-78bd-40f2-aa36-7ab53cdc345e

**Status:** `PASS`

validate optional OPTIONS /objects/{object_id} authorization metadata

| Case | Status | Message |
| --- | --- | --- |
| DRS Object Authorizations response status code validation | `PASS` | Authorization discovery metadata was returned |
| DRS Object Authorizations response content-type validation | `PASS` | Content-Type matches expected type |
| DRS Object Authorizations response schema validation | `PASS` | Schema Validation Successful |

#### Run DRS 1.5.0 authorization discovery for drs id = a5d6feee-03e3-48f9-8e78-798efd9f0aaf

**Status:** `PASS`

validate optional OPTIONS /objects/{object_id} authorization metadata

| Case | Status | Message |
| --- | --- | --- |
| DRS Object Authorizations response status code validation | `PASS` | Authorization discovery metadata was returned |
| DRS Object Authorizations response content-type validation | `PASS` | Content-Type matches expected type |
| DRS Object Authorizations response schema validation | `PASS` | Schema Validation Successful |

#### Run DRS 1.5.0 authorization discovery for drs id = a5d6feee-03e3-48f9-8e78-798efd9f0aaf

**Status:** `PASS`

validate optional OPTIONS /objects/{object_id} authorization metadata

| Case | Status | Message |
| --- | --- | --- |
| DRS Object Authorizations response status code validation | `PASS` | Authorization discovery metadata was returned |
| DRS Object Authorizations response content-type validation | `PASS` | Content-Type matches expected type |
| DRS Object Authorizations response schema validation | `PASS` | Schema Validation Successful |

#### Run DRS 1.5.0 authorization discovery for drs id = dd9e1020-87b9-4128-a485-123bba08fe8b

**Status:** `PASS`

validate optional OPTIONS /objects/{object_id} authorization metadata

| Case | Status | Message |
| --- | --- | --- |
| DRS Object Authorizations response status code validation | `PASS` | Authorization discovery metadata was returned |
| DRS Object Authorizations response content-type validation | `PASS` | Content-Type matches expected type |
| DRS Object Authorizations response schema validation | `PASS` | Schema Validation Successful |

#### Run DRS 1.5.0 authorization discovery for drs id = dd9e1020-87b9-4128-a485-123bba08fe8b

**Status:** `PASS`

validate optional OPTIONS /objects/{object_id} authorization metadata

| Case | Status | Message |
| --- | --- | --- |
| DRS Object Authorizations response status code validation | `PASS` | Authorization discovery metadata was returned |
| DRS Object Authorizations response content-type validation | `PASS` | Content-Type matches expected type |
| DRS Object Authorizations response schema validation | `PASS` | Schema Validation Successful |

#### Run DRS 1.5.0 authorization discovery for drs id = d5dd7d58-68ed-407b-83b6-ffd8ee6400da

**Status:** `PASS`

validate optional OPTIONS /objects/{object_id} authorization metadata

| Case | Status | Message |
| --- | --- | --- |
| DRS Object Authorizations response status code validation | `PASS` | Authorization discovery metadata was returned |
| DRS Object Authorizations response content-type validation | `PASS` | Content-Type matches expected type |
| DRS Object Authorizations response schema validation | `PASS` | Schema Validation Successful |

#### Run DRS 1.5.0 authorization discovery for drs id = d5dd7d58-68ed-407b-83b6-ffd8ee6400da

**Status:** `PASS`

validate optional OPTIONS /objects/{object_id} authorization metadata

| Case | Status | Message |
| --- | --- | --- |
| DRS Object Authorizations response status code validation | `PASS` | Authorization discovery metadata was returned |
| DRS Object Authorizations response content-type validation | `PASS` | Content-Type matches expected type |
| DRS Object Authorizations response schema validation | `PASS` | Schema Validation Successful |

#### Run DRS 1.5.0 bulk authorization discovery

**Status:** `PASS`

validate optional OPTIONS /objects authorization metadata for up to three configured DRS object IDs

| Case | Status | Message |
| --- | --- | --- |
| Bulk DRS Object Authorizations response status code validation | `PASS` | Authorization discovery metadata was returned |
| Bulk DRS Object Authorizations response content-type validation | `PASS` | Content-Type matches expected type |
| Bulk DRS Object Authorizations response schema validation | `PASS` | Schema Validation Successful |

### drs object info

**Status:** `WARN`

run DRS 1.5.0 tests for drs object info endpoint

| Passed | Failed | Warned | Skipped | Unknown |
| --- | --- | --- | --- | --- |
| 132 | 0 | 16 | 0 | 0 |

#### Run DRS 1.5.0 object tests for drs id = fb7ff2eb-53c4-4d0e-af6d-456fb6358565; auth_type = basic

**Status:** `WARN`

validate DRS object status, schema, required fields, and access methods

| Case | Status | Message |
| --- | --- | --- |
| DRS Object Info response status code validation | `PASS` | Response status code is 200 |
| DRS Object Info response content-type validation | `PASS` | Content-Type matches expected type |
| DRS Object Info response schema validation | `PASS` | Schema Validation Successful |
| DRS Object required fields | `PASS` | DRS object includes required fields |
| DRS Object id | `PASS` | Returned id matches requested object_id fb7ff2eb-53c4-4d0e-af6d-456fb6358565 |
| DRS Object self_uri | `PASS` | self_uri is drs://localhost:8888/fb7ff2eb-53c4-4d0e-af6d-456fb6358565 |
| DRS Object size | `PASS` | size is 33 |
| DRS Object created_time | `PASS` | created_time is 2026-06-01T13:31:13Z |
| DRS Object updated_time | `PASS` | updated_time is 2026-06-01T13:31:13Z |
| DRS Object checksum fields | `PASS` | Each checksum includes type and checksum values |
| DRS Object checksum encoding | `WARN` | Checksum values were not hex-like at indexes: 0 |
| DRS Object alternative access methods | `WARN` | Alternative access method types present: irods |
| DRS Object access_id uniqueness | `PASS` | All access_id values were unique |
| DRS Object Info has access information | `PASS` | 'access_methods' is provided and it is non-empty. |
| DRS Object Info has access information | `PASS` | At least 'access_url' or 'access_id' is provided in all access_methods |

#### Run DRS 1.5.0 object tests for drs id = fb7ff2eb-53c4-4d0e-af6d-456fb6358565; auth_type = bearer

**Status:** `WARN`

validate DRS object status, schema, required fields, and access methods

| Case | Status | Message |
| --- | --- | --- |
| DRS Object Info response status code validation | `PASS` | Response status code is 200 |
| DRS Object Info response content-type validation | `PASS` | Content-Type matches expected type |
| DRS Object Info response schema validation | `PASS` | Schema Validation Successful |
| DRS Object required fields | `PASS` | DRS object includes required fields |
| DRS Object id | `PASS` | Returned id matches requested object_id fb7ff2eb-53c4-4d0e-af6d-456fb6358565 |
| DRS Object self_uri | `PASS` | self_uri is drs://localhost:8888/fb7ff2eb-53c4-4d0e-af6d-456fb6358565 |
| DRS Object size | `PASS` | size is 33 |
| DRS Object created_time | `PASS` | created_time is 2026-06-01T13:31:13Z |
| DRS Object updated_time | `PASS` | updated_time is 2026-06-01T13:31:13Z |
| DRS Object checksum fields | `PASS` | Each checksum includes type and checksum values |
| DRS Object checksum encoding | `WARN` | Checksum values were not hex-like at indexes: 0 |
| DRS Object alternative access methods | `WARN` | Alternative access method types present: irods |
| DRS Object access_id uniqueness | `PASS` | All access_id values were unique |
| DRS Object Info has access information | `PASS` | 'access_methods' is provided and it is non-empty. |
| DRS Object Info has access information | `PASS` | At least 'access_url' or 'access_id' is provided in all access_methods |

#### Run DRS 1.5.0 object tests for drs id = 1138f628-78bd-40f2-aa36-7ab53cdc345e; auth_type = basic

**Status:** `WARN`

validate DRS object status, schema, required fields, and access methods

| Case | Status | Message |
| --- | --- | --- |
| DRS Object Info response status code validation | `PASS` | Response status code is 200 |
| DRS Object Info response content-type validation | `PASS` | Content-Type matches expected type |
| DRS Object Info response schema validation | `PASS` | Schema Validation Successful |
| DRS Object required fields | `PASS` | DRS object includes required fields |
| DRS Object id | `PASS` | Returned id matches requested object_id 1138f628-78bd-40f2-aa36-7ab53cdc345e |
| DRS Object self_uri | `PASS` | self_uri is drs://localhost:8888/1138f628-78bd-40f2-aa36-7ab53cdc345e |
| DRS Object size | `PASS` | size is 32 |
| DRS Object created_time | `PASS` | created_time is 2026-06-01T13:31:13Z |
| DRS Object updated_time | `PASS` | updated_time is 2026-06-01T13:31:13Z |
| DRS Object checksum fields | `PASS` | Each checksum includes type and checksum values |
| DRS Object checksum encoding | `WARN` | Checksum values were not hex-like at indexes: 0 |
| DRS Object alternative access methods | `WARN` | Alternative access method types present: irods |
| DRS Object access_id uniqueness | `PASS` | All access_id values were unique |
| DRS Object Info has access information | `PASS` | 'access_methods' is provided and it is non-empty. |
| DRS Object Info has access information | `PASS` | At least 'access_url' or 'access_id' is provided in all access_methods |

#### Run DRS 1.5.0 object tests for drs id = 1138f628-78bd-40f2-aa36-7ab53cdc345e; auth_type = bearer

**Status:** `WARN`

validate DRS object status, schema, required fields, and access methods

| Case | Status | Message |
| --- | --- | --- |
| DRS Object Info response status code validation | `PASS` | Response status code is 200 |
| DRS Object Info response content-type validation | `PASS` | Content-Type matches expected type |
| DRS Object Info response schema validation | `PASS` | Schema Validation Successful |
| DRS Object required fields | `PASS` | DRS object includes required fields |
| DRS Object id | `PASS` | Returned id matches requested object_id 1138f628-78bd-40f2-aa36-7ab53cdc345e |
| DRS Object self_uri | `PASS` | self_uri is drs://localhost:8888/1138f628-78bd-40f2-aa36-7ab53cdc345e |
| DRS Object size | `PASS` | size is 32 |
| DRS Object created_time | `PASS` | created_time is 2026-06-01T13:31:13Z |
| DRS Object updated_time | `PASS` | updated_time is 2026-06-01T13:31:13Z |
| DRS Object checksum fields | `PASS` | Each checksum includes type and checksum values |
| DRS Object checksum encoding | `WARN` | Checksum values were not hex-like at indexes: 0 |
| DRS Object alternative access methods | `WARN` | Alternative access method types present: irods |
| DRS Object access_id uniqueness | `PASS` | All access_id values were unique |
| DRS Object Info has access information | `PASS` | 'access_methods' is provided and it is non-empty. |
| DRS Object Info has access information | `PASS` | At least 'access_url' or 'access_id' is provided in all access_methods |

#### Run DRS 1.5.0 object tests for drs id = a5d6feee-03e3-48f9-8e78-798efd9f0aaf; auth_type = basic

**Status:** `WARN`

validate DRS object status, schema, required fields, and access methods

| Case | Status | Message |
| --- | --- | --- |
| DRS Object Info response status code validation | `PASS` | Response status code is 200 |
| DRS Object Info response content-type validation | `PASS` | Content-Type matches expected type |
| DRS Object Info response schema validation | `PASS` | Schema Validation Successful |
| DRS Object required fields | `PASS` | DRS object includes required fields |
| DRS Object id | `PASS` | Returned id matches requested object_id a5d6feee-03e3-48f9-8e78-798efd9f0aaf |
| DRS Object self_uri | `PASS` | self_uri is drs://localhost:8888/a5d6feee-03e3-48f9-8e78-798efd9f0aaf |
| DRS Object size | `PASS` | size is 32 |
| DRS Object created_time | `PASS` | created_time is 2026-06-01T13:31:13Z |
| DRS Object updated_time | `PASS` | updated_time is 2026-06-01T13:31:13Z |
| DRS Object checksum fields | `PASS` | Each checksum includes type and checksum values |
| DRS Object checksum encoding | `WARN` | Checksum values were not hex-like at indexes: 0 |
| DRS Object alternative access methods | `WARN` | Alternative access method types present: irods |
| DRS Object access_id uniqueness | `PASS` | All access_id values were unique |
| DRS Object Info has access information | `PASS` | 'access_methods' is provided and it is non-empty. |
| DRS Object Info has access information | `PASS` | At least 'access_url' or 'access_id' is provided in all access_methods |

#### Run DRS 1.5.0 object tests for drs id = a5d6feee-03e3-48f9-8e78-798efd9f0aaf; auth_type = bearer

**Status:** `WARN`

validate DRS object status, schema, required fields, and access methods

| Case | Status | Message |
| --- | --- | --- |
| DRS Object Info response status code validation | `PASS` | Response status code is 200 |
| DRS Object Info response content-type validation | `PASS` | Content-Type matches expected type |
| DRS Object Info response schema validation | `PASS` | Schema Validation Successful |
| DRS Object required fields | `PASS` | DRS object includes required fields |
| DRS Object id | `PASS` | Returned id matches requested object_id a5d6feee-03e3-48f9-8e78-798efd9f0aaf |
| DRS Object self_uri | `PASS` | self_uri is drs://localhost:8888/a5d6feee-03e3-48f9-8e78-798efd9f0aaf |
| DRS Object size | `PASS` | size is 32 |
| DRS Object created_time | `PASS` | created_time is 2026-06-01T13:31:13Z |
| DRS Object updated_time | `PASS` | updated_time is 2026-06-01T13:31:13Z |
| DRS Object checksum fields | `PASS` | Each checksum includes type and checksum values |
| DRS Object checksum encoding | `WARN` | Checksum values were not hex-like at indexes: 0 |
| DRS Object alternative access methods | `WARN` | Alternative access method types present: irods |
| DRS Object access_id uniqueness | `PASS` | All access_id values were unique |
| DRS Object Info has access information | `PASS` | 'access_methods' is provided and it is non-empty. |
| DRS Object Info has access information | `PASS` | At least 'access_url' or 'access_id' is provided in all access_methods |

#### Run DRS 1.5.0 object tests for drs id = dd9e1020-87b9-4128-a485-123bba08fe8b; auth_type = basic

**Status:** `WARN`

validate DRS object status, schema, required fields, and access methods

| Case | Status | Message |
| --- | --- | --- |
| DRS Object Info response status code validation | `PASS` | Response status code is 200 |
| DRS Object Info response content-type validation | `PASS` | Content-Type matches expected type |
| DRS Object Info response schema validation | `PASS` | Schema Validation Successful |
| DRS Object required fields | `PASS` | DRS object includes required fields |
| DRS Object id | `PASS` | Returned id matches requested object_id dd9e1020-87b9-4128-a485-123bba08fe8b |
| DRS Object self_uri | `PASS` | self_uri is drs://localhost:8888/dd9e1020-87b9-4128-a485-123bba08fe8b |
| DRS Object size | `PASS` | size is 32 |
| DRS Object created_time | `PASS` | created_time is 2026-06-01T13:31:13Z |
| DRS Object updated_time | `PASS` | updated_time is 2026-06-01T13:31:13Z |
| DRS Object checksum fields | `PASS` | Each checksum includes type and checksum values |
| DRS Object checksum encoding | `WARN` | Checksum values were not hex-like at indexes: 0 |
| DRS Object alternative access methods | `WARN` | Alternative access method types present: irods |
| DRS Object access_id uniqueness | `PASS` | All access_id values were unique |
| DRS Object Info has access information | `PASS` | 'access_methods' is provided and it is non-empty. |
| DRS Object Info has access information | `PASS` | At least 'access_url' or 'access_id' is provided in all access_methods |

#### Run DRS 1.5.0 object tests for drs id = dd9e1020-87b9-4128-a485-123bba08fe8b; auth_type = bearer

**Status:** `WARN`

validate DRS object status, schema, required fields, and access methods

| Case | Status | Message |
| --- | --- | --- |
| DRS Object Info response status code validation | `PASS` | Response status code is 200 |
| DRS Object Info response content-type validation | `PASS` | Content-Type matches expected type |
| DRS Object Info response schema validation | `PASS` | Schema Validation Successful |
| DRS Object required fields | `PASS` | DRS object includes required fields |
| DRS Object id | `PASS` | Returned id matches requested object_id dd9e1020-87b9-4128-a485-123bba08fe8b |
| DRS Object self_uri | `PASS` | self_uri is drs://localhost:8888/dd9e1020-87b9-4128-a485-123bba08fe8b |
| DRS Object size | `PASS` | size is 32 |
| DRS Object created_time | `PASS` | created_time is 2026-06-01T13:31:13Z |
| DRS Object updated_time | `PASS` | updated_time is 2026-06-01T13:31:13Z |
| DRS Object checksum fields | `PASS` | Each checksum includes type and checksum values |
| DRS Object checksum encoding | `WARN` | Checksum values were not hex-like at indexes: 0 |
| DRS Object alternative access methods | `WARN` | Alternative access method types present: irods |
| DRS Object access_id uniqueness | `PASS` | All access_id values were unique |
| DRS Object Info has access information | `PASS` | 'access_methods' is provided and it is non-empty. |
| DRS Object Info has access information | `PASS` | At least 'access_url' or 'access_id' is provided in all access_methods |

#### Run DRS 1.5.0 object tests for drs id = d5dd7d58-68ed-407b-83b6-ffd8ee6400da; auth_type = basic

**Status:** `PASS`

validate DRS object status, schema, required fields, and access methods

| Case | Status | Message |
| --- | --- | --- |
| DRS Object Info response status code validation | `PASS` | Response status code is 200 |
| DRS Object Info response content-type validation | `PASS` | Content-Type matches expected type |
| DRS Object Info response schema validation | `PASS` | Schema Validation Successful |
| DRS Object required fields | `PASS` | DRS object includes required fields |
| DRS Object id | `PASS` | Returned id matches requested object_id d5dd7d58-68ed-407b-83b6-ffd8ee6400da |
| DRS Object self_uri | `PASS` | self_uri is drs://localhost:8888/d5dd7d58-68ed-407b-83b6-ffd8ee6400da |
| DRS Object size | `PASS` | size is 0 |
| DRS Object created_time | `PASS` | created_time is 2026-06-01T13:31:14Z |
| DRS Object updated_time | `PASS` | updated_time is 2026-06-01T13:31:14Z |
| DRS Object checksum fields | `PASS` | Each checksum includes type and checksum values |
| DRS Object checksum encoding | `PASS` | Checksum values were hex-like |
| DRS Object direct access_url | `PASS` | Found 1 direct access_url values |
| DRS Object Info has access information | `PASS` | 'access_methods' is provided and it is non-empty. |
| DRS Object Info has access information | `PASS` | At least 'access_url' or 'access_id' is provided in all access_methods |

#### Run DRS 1.5.0 object tests for drs id = d5dd7d58-68ed-407b-83b6-ffd8ee6400da; auth_type = bearer

**Status:** `PASS`

validate DRS object status, schema, required fields, and access methods

| Case | Status | Message |
| --- | --- | --- |
| DRS Object Info response status code validation | `PASS` | Response status code is 200 |
| DRS Object Info response content-type validation | `PASS` | Content-Type matches expected type |
| DRS Object Info response schema validation | `PASS` | Schema Validation Successful |
| DRS Object required fields | `PASS` | DRS object includes required fields |
| DRS Object id | `PASS` | Returned id matches requested object_id d5dd7d58-68ed-407b-83b6-ffd8ee6400da |
| DRS Object self_uri | `PASS` | self_uri is drs://localhost:8888/d5dd7d58-68ed-407b-83b6-ffd8ee6400da |
| DRS Object size | `PASS` | size is 0 |
| DRS Object created_time | `PASS` | created_time is 2026-06-01T13:31:14Z |
| DRS Object updated_time | `PASS` | updated_time is 2026-06-01T13:31:14Z |
| DRS Object checksum fields | `PASS` | Each checksum includes type and checksum values |
| DRS Object checksum encoding | `PASS` | Checksum values were hex-like |
| DRS Object direct access_url | `PASS` | Found 1 direct access_url values |
| DRS Object Info has access information | `PASS` | 'access_methods' is provided and it is non-empty. |
| DRS Object Info has access information | `PASS` | At least 'access_url' or 'access_id' is provided in all access_methods |

### drs object access

**Status:** `PASS`

run DRS 1.5.0 tests for drs access endpoint

| Passed | Failed | Warned | Skipped | Unknown |
| --- | --- | --- | --- | --- |
| 12 | 0 | 0 | 0 | 0 |

#### Run DRS 1.5.0 access URL tests for drs id = fb7ff2eb-53c4-4d0e-af6d-456fb6358565 and access id = irods-go-rest-https-providerResc; auth_type = basic

**Status:** `PASS`

validate DRS access URL status, Retry-After behavior, and response schema

| Case | Status | Message |
| --- | --- | --- |
| DRS Access response status code validation | `PASS` | Response status code is 200 |
| DRS Access response content-type validation | `PASS` | Content-Type matches expected type |
| DRS Access response schema validation | `PASS` | Schema Validation Successful |

#### Run DRS 1.5.0 access URL tests for drs id = fb7ff2eb-53c4-4d0e-af6d-456fb6358565 and access id = irods; auth_type = basic

**Status:** `PASS`

validate DRS access URL status, Retry-After behavior, and response schema

| Case | Status | Message |
| --- | --- | --- |
| DRS Access response status code validation | `PASS` | Response status code is 200 |
| DRS Access response content-type validation | `PASS` | Content-Type matches expected type |
| DRS Access response schema validation | `PASS` | Schema Validation Successful |

#### Run DRS 1.5.0 access URL tests for drs id = fb7ff2eb-53c4-4d0e-af6d-456fb6358565 and access id = irods-go-rest-https-providerResc; auth_type = bearer

**Status:** `PASS`

validate DRS access URL status, Retry-After behavior, and response schema

| Case | Status | Message |
| --- | --- | --- |
| DRS Access response status code validation | `PASS` | Response status code is 200 |
| DRS Access response content-type validation | `PASS` | Content-Type matches expected type |
| DRS Access response schema validation | `PASS` | Schema Validation Successful |

#### Run DRS 1.5.0 access URL tests for drs id = fb7ff2eb-53c4-4d0e-af6d-456fb6358565 and access id = irods; auth_type = bearer

**Status:** `PASS`

validate DRS access URL status, Retry-After behavior, and response schema

| Case | Status | Message |
| --- | --- | --- |
| DRS Access response status code validation | `PASS` | Response status code is 200 |
| DRS Access response content-type validation | `PASS` | Content-Type matches expected type |
| DRS Access response schema validation | `PASS` | Schema Validation Successful |

### bulk drs objects

**Status:** `WARN`

run optional DRS 1.5.0 bulk object endpoint checks

| Passed | Failed | Warned | Skipped | Unknown |
| --- | --- | --- | --- | --- |
| 0 | 0 | 1 | 0 | 0 |

#### Run DRS 1.5.0 bulk object tests

**Status:** `WARN`

validate POST /objects for configured DRS object IDs

| Case | Status | Message |
| --- | --- | --- |
| Bulk DRS Object optional support | `WARN` | Optional endpoint is not supported; server returned 501 |

### bulk drs access

**Status:** `WARN`

run optional DRS 1.5.0 bulk access URL endpoint checks

| Passed | Failed | Warned | Skipped | Unknown |
| --- | --- | --- | --- | --- |
| 0 | 0 | 1 | 0 | 0 |

#### Run DRS 1.5.0 bulk access URL tests

**Status:** `WARN`

validate POST /objects/access for discovered access IDs

| Case | Status | Message |
| --- | --- | --- |
| Bulk DRS Access optional support | `WARN` | Optional endpoint is not supported; server returned 501 |

### compound manifests

**Status:** `PASS`

retrieve HTTP(S) manifests for configured DRS 1.5.0 compound objects

| Passed | Failed | Warned | Skipped | Unknown |
| --- | --- | --- | --- | --- |
| 4 | 0 | 0 | 0 | 0 |

#### Run DRS 1.5.0 compound manifest tests for drs id = d5dd7d58-68ed-407b-83b6-ffd8ee6400da

**Status:** `PASS`

resolve an HTTP(S) access URL and validate the returned compound manifest

| Case | Status | Message |
| --- | --- | --- |
| Compound DRS Object response status code validation | `PASS` | Response status code is 200 |
| Compound HTTP(S) access_url advertised | `PASS` | Found direct HTTP(S) access_url: http://localhost:8888/ga4gh/drs/v1/ext/compound/d5dd7d58-68ed-407b-83b6-ffd8ee6400da |
| Compound HTTP(S) manifest retrieval | `PASS` | Retrieved compound manifest with status code 200 |
| Compound JSON manifest payload | `PASS` | Compound manifest payload was valid JSON |

### error behavior

**Status:** `PASS`

run configured DRS 1.5.0 negative tests for documented error responses

| Passed | Failed | Warned | Skipped | Unknown |
| --- | --- | --- | --- | --- |
| 12 | 0 | 0 | 0 | 0 |

#### Run DRS 1.5.0 invalid object ID test for drs id = __drs_certification_missing_object_1780320672997639000

**Status:** `PASS`

validate GET /objects/{object_id} returns a documented error for a known invalid ID

| Case | Status | Message |
| --- | --- | --- |
| Invalid DRS Object error status code validation | `PASS` | Response status code is 404 |
| Invalid DRS Object error response content-type validation | `PASS` | Content-Type matches expected type |
| Invalid DRS Object error response schema validation | `PASS` | Schema Validation Successful |

#### Run DRS 1.5.0 invalid auth test for drs id = fb7ff2eb-53c4-4d0e-af6d-456fb6358565

**Status:** `PASS`

validate GET /objects/{object_id} rejects known invalid credentials

| Case | Status | Message |
| --- | --- | --- |
| Invalid DRS Auth error status code validation | `PASS` | Response status code is 401 |
| Invalid DRS Auth error response content-type validation | `PASS` | Content-Type matches expected type |
| Invalid DRS Auth error response schema validation | `PASS` | Schema Validation Successful |

#### Run DRS 1.5.0 invalid auth test for drs id = fb7ff2eb-53c4-4d0e-af6d-456fb6358565

**Status:** `PASS`

validate GET /objects/{object_id} rejects known invalid credentials

| Case | Status | Message |
| --- | --- | --- |
| Invalid DRS Auth error status code validation | `PASS` | Response status code is 401 |
| Invalid DRS Auth error response content-type validation | `PASS` | Content-Type matches expected type |
| Invalid DRS Auth error response schema validation | `PASS` | Schema Validation Successful |

#### Run DRS 1.5.0 invalid access ID test for drs id = fb7ff2eb-53c4-4d0e-af6d-456fb6358565 and access id = __drs_certification_missing_access__

**Status:** `PASS`

validate GET /objects/{object_id}/access/{access_id} returns a documented error for a known invalid access ID

| Case | Status | Message |
| --- | --- | --- |
| Invalid DRS Access error status code validation | `PASS` | Response status code is 404 |
| Invalid DRS Access error response content-type validation | `PASS` | Content-Type matches expected type |
| Invalid DRS Access error response schema validation | `PASS` | Schema Validation Successful |
