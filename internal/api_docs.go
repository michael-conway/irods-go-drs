package internal

import (
	"fmt"
	"net/http"
	"net/url"
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
	specURL := "/openapi.yaml"
	if serverURL := configuredServerURL(r); serverURL != "" {
		specURL = fmt.Sprintf("%s://%s/openapi.yaml", requestScheme(r), serverURL)
	}

	html := strings.Replace(swaggerUIHTML, `url: "/openapi.yaml"`, fmt.Sprintf(`url: %q`, specURL), 1)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(html))
}

func configuredServerURL(r *http.Request) string {
	drsConfig, err := readRouteDrsConfig()
	if err != nil || drsConfig == nil {
		return strings.TrimSpace(r.Host)
	}

	host := requestHostName(r)
	if host == "" {
		host = "localhost"
	}

	port := drsConfig.DrsListenPort
	if port <= 0 {
		return host
	}

	return fmt.Sprintf("%s:%d", host, port)
}

func requestScheme(r *http.Request) string {
	if r == nil {
		return "http"
	}

	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwarded != "" {
		return forwarded
	}

	if r.TLS != nil {
		return "https"
	}

	return "http"
}

func requestHostName(r *http.Request) string {
	if r == nil {
		return ""
	}

	host := strings.TrimSpace(r.Host)
	if host == "" {
		return ""
	}

	if parsed := strings.TrimSpace(host); parsed != "" {
		if withScheme, err := url.Parse("http://" + parsed); err == nil {
			return strings.TrimSpace(withScheme.Hostname())
		}
	}

	return host
}
