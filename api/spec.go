package api

import _ "embed"

// SwaggerSpec contains the bundled OpenAPI specification served by the app.
//
//go:embed swagger.yaml
var SwaggerSpec []byte
