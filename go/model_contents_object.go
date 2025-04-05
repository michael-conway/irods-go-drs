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

type ContentsObject struct {
	// A name declared by the bundle author that must be used when materialising this object, overriding any name directly associated with the object itself. The name must be unique within the containing bundle. This string is made up of uppercase and lowercase letters, decimal digits, hyphen, period, and underscore [A-Za-z0-9.-_]. See http://pubs.opengroup.org/onlinepubs/9699919799/basedefs/V1_chap03.html#tag_03_282[portable filenames].
	Name string `json:"name"`
	// A DRS identifier of a `DrsObject` (either a single blob or a nested bundle). If this ContentsObject is an object within a nested bundle, then the id is optional. Otherwise, the id is required.
	Id string `json:"id,omitempty"`
	// A list of full DRS identifier URI paths that may be used to obtain the object. These URIs may be external to this DRS instance.
	DrsUri []string `json:"drs_uri,omitempty"`
	// If this ContentsObject describes a nested bundle and the caller specified \"?expand=true\" on the request, then this contents array must be present and describe the objects within the nested bundle.
	Contents []ContentsObject `json:"contents,omitempty"`
}
