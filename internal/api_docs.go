package internal

import (
	"fmt"
	"net/http"
	"strings"

	api "github.com/michael-conway/irods-go-drs/api"
)

const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>iRODS DRS API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.ui = SwaggerUIBundle({
      url: "/openapi.yaml",
      dom_id: "#swagger-ui"
    });
  </script>
</body>
</html>
`

func GetOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	specBytes := api.SwaggerSpec
	if serverURL := configuredServerURL(r); serverURL != "" {
		specBytes = openAPISpecWithServerURL(api.SwaggerSpec, serverURL)
	}

	w.Header().Set("Content-Type", "application/yaml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(specBytes)
}

func GetSwaggerUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(swaggerUIHTML))
}

func configuredServerURL(r *http.Request) string {
	if r != nil {
		if host := strings.TrimSpace(r.Host); host != "" {
			return host
		}
	}

	drsConfig, err := readRouteDrsConfig()
	if err != nil || drsConfig == nil {
		return ""
	}

	port := drsConfig.DrsListenPort
	if port <= 0 {
		return "localhost"
	}

	return fmt.Sprintf("localhost:%d", port)
}

func openAPISpecWithServerURL(spec []byte, serverURL string) []byte {
	serverURL = strings.TrimSpace(serverURL)
	if serverURL == "" {
		return spec
	}

	specText := string(spec)
	serverURLIndex := strings.Index(specText, "serverURL:")
	if serverURLIndex < 0 {
		return spec
	}

	defaultIndex := strings.Index(specText[serverURLIndex:], "default:")
	if defaultIndex < 0 {
		return spec
	}
	defaultIndex += serverURLIndex

	valueStart := defaultIndex + len("default:")
	for valueStart < len(specText) && (specText[valueStart] == ' ' || specText[valueStart] == '\t') {
		valueStart++
	}

	lineEnd := strings.IndexByte(specText[valueStart:], '\n')
	if lineEnd < 0 {
		return []byte(specText[:valueStart] + serverURL)
	}
	lineEnd += valueStart

	return []byte(specText[:valueStart] + serverURL + specText[lineEnd:])
}
