/*
 * Data Repository Service
 *
 * No description provided (generated by Swagger Codegen https://github.com/swagger-api/swagger-codegen)
 *
 * API version: 1.5.0
 * Contact: ga4gh-cloud@ga4gh.org
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */
package swagger

type BulkObjectAccessIdBulkObjectAccessIds struct {
	// DRS object ID
	BulkObjectId string `json:"bulk_object_id,omitempty"`
	// DRS object access ID
	BulkAccessIds []string `json:"bulk_access_ids,omitempty"`
}
