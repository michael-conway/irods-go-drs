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
| 110 | 0 | 10 | 0 | 0 |

## API Coverage Highlights

> Coverage gaps in this section are informational unless an individual test case failed.

### Optional Capability Matrix

| Capability | Status | Notes |
| --- | --- | --- |
| Access ID URL resolution | `Supported` | /objects/cf293ba2-f8be-4dc0-9434-c4a9edf35f52/access/irods-go-rest-https-demoResc returned 200<br>/objects/cf293ba2-f8be-4dc0-9434-c4a9edf35f52/access/irods returned 200 |
| Bulk access POST | `Not supported` | POST /objects/access returned 501<br>Deprecated or optional capability |
| Bulk authorization OPTIONS | `Supported` | OPTIONS /objects returned 200 for 3 object IDs |
| Bulk object POST | `Not supported` | POST /objects returned 501<br>Deprecated or optional capability |
| Bundle expand | `No sample` | No configured DRS object was marked is_bundle = true<br>Deprecated or optional capability |
| Compound manifest retrieval | `Supported` | Retrieved json compound manifest from http://localhost:8888/ga4gh/drs/v1/ext/compound/b5e243fb-ee87-4903-9bdf-6d6b6514a323 with status code 200 |
| Object authorization OPTIONS | `Supported` | OPTIONS /objects/cf293ba2-f8be-4dc0-9434-c4a9edf35f52 returned 200<br>OPTIONS /objects/1538e5b3-3091-4ace-a2e7-27feb163d3fb returned 200<br>OPTIONS /objects/765a9eed-200b-4bf2-8eb5-fd9523616290 returned 200 |
| Passport access POST | `No sample` | No configured DRS access sample used auth_type passport with a discovered access_id |
| Passport object POST | `No sample` | No configured DRS object used auth_type passport |

### Auth Coverage Matrix

| Auth Type | Object Requests | Access Requests | Authorization Metadata | Access Method Metadata | Note |
| --- | --- | --- | --- | --- | --- |
| basic | `yes` | `yes` | `yes` | `yes` | basic auth was encountered |
| bearer | `no` | `no` | `yes` | `yes` | bearer auth was advertised but no configured sample used it |
| passport | `no` | `no` | `no` | `no` | passport auth was not encountered in configured requests or DRS metadata |

### Compound Manifest Support

- Supported: `yes`
- Manifest types observed: `json`

Sample manifest:

```json
// Sample compound manifest returned by the DRS server
{
  "host": "localhost",
  "manifest": {
    "alias": ".",
    "children": [
      {
        "alias": "ignored",
        "children": [
          {
            "drsId": "",
            "nodeType": "data_object",
            "path": "/tempZone/home/test1/drs-certification/1779286500139847000/compound-root/ignored/ignored.txt",
            "relativePath": "ignored/ignored.txt",
            "willAssignDrsId": true
          }
        ],
        "description": "ignored",
        "drsId": "",
        "nodeType": "collection",
        "path": "/tempZone/home/test1/drs-certification/1779286500139847000/compound-root/ignored",
        "relativePath": "ignored"
      },
      {
        "alias": "included",
        "children": [
          {
            "drsId": "24c860c4-5d91-4b3d-93bd-490150f09218",
            "nodeType": "data_object",
            "path": "/tempZone/home/test1/drs-certification/1779286500139847000/compound-root/included/child.txt",
            "relativePath": "included/child.txt"
          }
        ],
        "description": "included",
        "drsId": "",
        "nodeType": "collection",
        "path": "/tempZone/home/test1/drs-certification/1779286500139847000/compound-root/included",
        "relativePath": "included"
      }
    ],
    "description": ".",
    "drsId": "b5e243fb-ee87-4903-9bdf-6d6b6514a323",
    "nodeType": "collection",
    "path": "/tempZone/home/test1/drs-certification/1779286500139847000/compound-root",
    "relativePath": ""
  },
  "port": 1247,
  "rootDrsId": "b5e243fb-ee87-4903-9bdf-6d6b6514a323",
  "rootPath": "/tempZone/home/test1/drs-certification/1779286500139847000/compound-root",
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
| 18 | 0 | 0 | 0 | 0 |

#### Run DRS 1.5.0 authorization discovery for drs id = cf293ba2-f8be-4dc0-9434-c4a9edf35f52

**Status:** `PASS`

validate optional OPTIONS /objects/{object_id} authorization metadata

| Case | Status | Message |
| --- | --- | --- |
| DRS Object Authorizations response status code validation | `PASS` | Authorization discovery metadata was returned |
| DRS Object Authorizations response content-type validation | `PASS` | Content-Type matches expected type |
| DRS Object Authorizations response schema validation | `PASS` | Schema Validation Successful |

#### Run DRS 1.5.0 authorization discovery for drs id = 1538e5b3-3091-4ace-a2e7-27feb163d3fb

**Status:** `PASS`

validate optional OPTIONS /objects/{object_id} authorization metadata

| Case | Status | Message |
| --- | --- | --- |
| DRS Object Authorizations response status code validation | `PASS` | Authorization discovery metadata was returned |
| DRS Object Authorizations response content-type validation | `PASS` | Content-Type matches expected type |
| DRS Object Authorizations response schema validation | `PASS` | Schema Validation Successful |

#### Run DRS 1.5.0 authorization discovery for drs id = 765a9eed-200b-4bf2-8eb5-fd9523616290

**Status:** `PASS`

validate optional OPTIONS /objects/{object_id} authorization metadata

| Case | Status | Message |
| --- | --- | --- |
| DRS Object Authorizations response status code validation | `PASS` | Authorization discovery metadata was returned |
| DRS Object Authorizations response content-type validation | `PASS` | Content-Type matches expected type |
| DRS Object Authorizations response schema validation | `PASS` | Schema Validation Successful |

#### Run DRS 1.5.0 authorization discovery for drs id = fdcc8c1b-4efd-440d-94a0-70c60651ae0f

**Status:** `PASS`

validate optional OPTIONS /objects/{object_id} authorization metadata

| Case | Status | Message |
| --- | --- | --- |
| DRS Object Authorizations response status code validation | `PASS` | Authorization discovery metadata was returned |
| DRS Object Authorizations response content-type validation | `PASS` | Content-Type matches expected type |
| DRS Object Authorizations response schema validation | `PASS` | Schema Validation Successful |

#### Run DRS 1.5.0 authorization discovery for drs id = b5e243fb-ee87-4903-9bdf-6d6b6514a323

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
| 66 | 0 | 8 | 0 | 0 |

#### Run DRS 1.5.0 object tests for drs id = cf293ba2-f8be-4dc0-9434-c4a9edf35f52; auth_type = basic

**Status:** `WARN`

validate DRS object status, schema, required fields, and access methods

| Case | Status | Message |
| --- | --- | --- |
| DRS Object Info response status code validation | `PASS` | Response status code is 200 |
| DRS Object Info response content-type validation | `PASS` | Content-Type matches expected type |
| DRS Object Info response schema validation | `PASS` | Schema Validation Successful |
| DRS Object required fields | `PASS` | DRS object includes required fields |
| DRS Object id | `PASS` | Returned id matches requested object_id cf293ba2-f8be-4dc0-9434-c4a9edf35f52 |
| DRS Object self_uri | `PASS` | self_uri is drs://localhost:8888/cf293ba2-f8be-4dc0-9434-c4a9edf35f52 |
| DRS Object size | `PASS` | size is 33 |
| DRS Object created_time | `PASS` | created_time is 2026-05-20T10:15:00-04:00 |
| DRS Object updated_time | `PASS` | updated_time is 2026-05-20T10:15:00-04:00 |
| DRS Object checksum fields | `PASS` | Each checksum includes type and checksum values |
| DRS Object checksum encoding | `WARN` | Checksum values were not hex-like at indexes: 0 |
| DRS Object alternative access methods | `WARN` | Alternative access method types present: irods |
| DRS Object access_id uniqueness | `PASS` | All access_id values were unique |
| DRS Object Info has access information | `PASS` | 'access_methods' is provided and it is non-empty. |
| DRS Object Info has access information | `PASS` | At least 'access_url' or 'access_id' is provided in all access_methods |

#### Run DRS 1.5.0 object tests for drs id = 1538e5b3-3091-4ace-a2e7-27feb163d3fb; auth_type = basic

**Status:** `WARN`

validate DRS object status, schema, required fields, and access methods

| Case | Status | Message |
| --- | --- | --- |
| DRS Object Info response status code validation | `PASS` | Response status code is 200 |
| DRS Object Info response content-type validation | `PASS` | Content-Type matches expected type |
| DRS Object Info response schema validation | `PASS` | Schema Validation Successful |
| DRS Object required fields | `PASS` | DRS object includes required fields |
| DRS Object id | `PASS` | Returned id matches requested object_id 1538e5b3-3091-4ace-a2e7-27feb163d3fb |
| DRS Object self_uri | `PASS` | self_uri is drs://localhost:8888/1538e5b3-3091-4ace-a2e7-27feb163d3fb |
| DRS Object size | `PASS` | size is 32 |
| DRS Object created_time | `PASS` | created_time is 2026-05-20T10:15:00-04:00 |
| DRS Object updated_time | `PASS` | updated_time is 2026-05-20T10:15:00-04:00 |
| DRS Object checksum fields | `PASS` | Each checksum includes type and checksum values |
| DRS Object checksum encoding | `WARN` | Checksum values were not hex-like at indexes: 0 |
| DRS Object alternative access methods | `WARN` | Alternative access method types present: irods |
| DRS Object access_id uniqueness | `PASS` | All access_id values were unique |
| DRS Object Info has access information | `PASS` | 'access_methods' is provided and it is non-empty. |
| DRS Object Info has access information | `PASS` | At least 'access_url' or 'access_id' is provided in all access_methods |

#### Run DRS 1.5.0 object tests for drs id = 765a9eed-200b-4bf2-8eb5-fd9523616290; auth_type = basic

**Status:** `WARN`

validate DRS object status, schema, required fields, and access methods

| Case | Status | Message |
| --- | --- | --- |
| DRS Object Info response status code validation | `PASS` | Response status code is 200 |
| DRS Object Info response content-type validation | `PASS` | Content-Type matches expected type |
| DRS Object Info response schema validation | `PASS` | Schema Validation Successful |
| DRS Object required fields | `PASS` | DRS object includes required fields |
| DRS Object id | `PASS` | Returned id matches requested object_id 765a9eed-200b-4bf2-8eb5-fd9523616290 |
| DRS Object self_uri | `PASS` | self_uri is drs://localhost:8888/765a9eed-200b-4bf2-8eb5-fd9523616290 |
| DRS Object size | `PASS` | size is 32 |
| DRS Object created_time | `PASS` | created_time is 2026-05-20T10:15:00-04:00 |
| DRS Object updated_time | `PASS` | updated_time is 2026-05-20T10:15:00-04:00 |
| DRS Object checksum fields | `PASS` | Each checksum includes type and checksum values |
| DRS Object checksum encoding | `WARN` | Checksum values were not hex-like at indexes: 0 |
| DRS Object alternative access methods | `WARN` | Alternative access method types present: irods |
| DRS Object access_id uniqueness | `PASS` | All access_id values were unique |
| DRS Object Info has access information | `PASS` | 'access_methods' is provided and it is non-empty. |
| DRS Object Info has access information | `PASS` | At least 'access_url' or 'access_id' is provided in all access_methods |

#### Run DRS 1.5.0 object tests for drs id = fdcc8c1b-4efd-440d-94a0-70c60651ae0f; auth_type = basic

**Status:** `WARN`

validate DRS object status, schema, required fields, and access methods

| Case | Status | Message |
| --- | --- | --- |
| DRS Object Info response status code validation | `PASS` | Response status code is 200 |
| DRS Object Info response content-type validation | `PASS` | Content-Type matches expected type |
| DRS Object Info response schema validation | `PASS` | Schema Validation Successful |
| DRS Object required fields | `PASS` | DRS object includes required fields |
| DRS Object id | `PASS` | Returned id matches requested object_id fdcc8c1b-4efd-440d-94a0-70c60651ae0f |
| DRS Object self_uri | `PASS` | self_uri is drs://localhost:8888/fdcc8c1b-4efd-440d-94a0-70c60651ae0f |
| DRS Object size | `PASS` | size is 32 |
| DRS Object created_time | `PASS` | created_time is 2026-05-20T10:15:00-04:00 |
| DRS Object updated_time | `PASS` | updated_time is 2026-05-20T10:15:00-04:00 |
| DRS Object checksum fields | `PASS` | Each checksum includes type and checksum values |
| DRS Object checksum encoding | `WARN` | Checksum values were not hex-like at indexes: 0 |
| DRS Object alternative access methods | `WARN` | Alternative access method types present: irods |
| DRS Object access_id uniqueness | `PASS` | All access_id values were unique |
| DRS Object Info has access information | `PASS` | 'access_methods' is provided and it is non-empty. |
| DRS Object Info has access information | `PASS` | At least 'access_url' or 'access_id' is provided in all access_methods |

#### Run DRS 1.5.0 object tests for drs id = b5e243fb-ee87-4903-9bdf-6d6b6514a323; auth_type = basic

**Status:** `PASS`

validate DRS object status, schema, required fields, and access methods

| Case | Status | Message |
| --- | --- | --- |
| DRS Object Info response status code validation | `PASS` | Response status code is 200 |
| DRS Object Info response content-type validation | `PASS` | Content-Type matches expected type |
| DRS Object Info response schema validation | `PASS` | Schema Validation Successful |
| DRS Object required fields | `PASS` | DRS object includes required fields |
| DRS Object id | `PASS` | Returned id matches requested object_id b5e243fb-ee87-4903-9bdf-6d6b6514a323 |
| DRS Object self_uri | `PASS` | self_uri is drs://localhost:8888/b5e243fb-ee87-4903-9bdf-6d6b6514a323 |
| DRS Object size | `PASS` | size is 0 |
| DRS Object created_time | `PASS` | created_time is 2026-05-20T10:15:00-04:00 |
| DRS Object updated_time | `PASS` | updated_time is 2026-05-20T10:15:00-04:00 |
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
| 6 | 0 | 0 | 0 | 0 |

#### Run DRS 1.5.0 access URL tests for drs id = cf293ba2-f8be-4dc0-9434-c4a9edf35f52 and access id = irods-go-rest-https-demoResc; auth_type = basic

**Status:** `PASS`

validate DRS access URL status, Retry-After behavior, and response schema

| Case | Status | Message |
| --- | --- | --- |
| DRS Access response status code validation | `PASS` | Response status code is 200 |
| DRS Access response content-type validation | `PASS` | Content-Type matches expected type |
| DRS Access response schema validation | `PASS` | Schema Validation Successful |

#### Run DRS 1.5.0 access URL tests for drs id = cf293ba2-f8be-4dc0-9434-c4a9edf35f52 and access id = irods; auth_type = basic

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

#### Run DRS 1.5.0 compound manifest tests for drs id = b5e243fb-ee87-4903-9bdf-6d6b6514a323

**Status:** `PASS`

resolve an HTTP(S) access URL and validate the returned compound manifest

| Case | Status | Message |
| --- | --- | --- |
| Compound DRS Object response status code validation | `PASS` | Response status code is 200 |
| Compound HTTP(S) access_url advertised | `PASS` | Found direct HTTP(S) access_url: http://localhost:8888/ga4gh/drs/v1/ext/compound/b5e243fb-ee87-4903-9bdf-6d6b6514a323 |
| Compound HTTP(S) manifest retrieval | `PASS` | Retrieved compound manifest with status code 200 |
| Compound JSON manifest payload | `PASS` | Compound manifest payload was valid JSON |

### error behavior

**Status:** `PASS`

run configured DRS 1.5.0 negative tests for documented error responses

| Passed | Failed | Warned | Skipped | Unknown |
| --- | --- | --- | --- | --- |
| 9 | 0 | 0 | 0 | 0 |

#### Run DRS 1.5.0 invalid object ID test for drs id = __drs_certification_missing_object_1779286500139847000

**Status:** `PASS`

validate GET /objects/{object_id} returns a documented error for a known invalid ID

| Case | Status | Message |
| --- | --- | --- |
| Invalid DRS Object error status code validation | `PASS` | Response status code is 404 |
| Invalid DRS Object error response content-type validation | `PASS` | Content-Type matches expected type |
| Invalid DRS Object error response schema validation | `PASS` | Schema Validation Successful |

#### Run DRS 1.5.0 invalid auth test for drs id = cf293ba2-f8be-4dc0-9434-c4a9edf35f52

**Status:** `PASS`

validate GET /objects/{object_id} rejects known invalid credentials

| Case | Status | Message |
| --- | --- | --- |
| Invalid DRS Auth error status code validation | `PASS` | Response status code is 401 |
| Invalid DRS Auth error response content-type validation | `PASS` | Content-Type matches expected type |
| Invalid DRS Auth error response schema validation | `PASS` | Schema Validation Successful |

#### Run DRS 1.5.0 invalid access ID test for drs id = cf293ba2-f8be-4dc0-9434-c4a9edf35f52 and access id = __drs_certification_missing_access__

**Status:** `PASS`

validate GET /objects/{object_id}/access/{access_id} returns a documented error for a known invalid access ID

| Case | Status | Message |
| --- | --- | --- |
| Invalid DRS Access error status code validation | `PASS` | Response status code is 404 |
| Invalid DRS Access error response content-type validation | `PASS` | Content-Type matches expected type |
| Invalid DRS Access error response schema validation | `PASS` | Schema Validation Successful |
