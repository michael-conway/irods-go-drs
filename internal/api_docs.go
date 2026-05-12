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
		specBytes = []byte(strings.Replace(string(api.SwaggerSpec), "default: localhost:8080", "default: "+serverURL, 1))
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
		if forwardedHost := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); forwardedHost != "" {
			return forwardedHost
		}

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
		return ""
	}

	return fmt.Sprintf("localhost:%d", port)
}
