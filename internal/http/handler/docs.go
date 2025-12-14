package handler

import (
	"encoding/json"
	"net/http"
	"os"

	"gopkg.in/yaml.v3"
)

// GET /openapi.yaml
func OpenAPI(w http.ResponseWriter, r *http.Request) {
	// disable caching in dev
	if os.Getenv("APP_ENV") == "production" {
		w.Header().Set("Cache-Control", "public, max-age=3600")
	} else {
		w.Header().Set("Cache-Control", "no-cache")
	}

	http.ServeFile(w, r, "openapi.yaml")
}

// GET /docs
func SwaggerUI(w http.ResponseWriter, r *http.Request) {
	html := `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>Money Tracker API Docs</title>
  <link rel="stylesheet" 
        href="https://unpkg.com/swagger-ui-dist/swagger-ui.css">
</head>
<body>
<div id="swagger"></div>

<script src="https://unpkg.com/swagger-ui-dist/swagger-ui-bundle.js"></script>
<script>
SwaggerUIBundle({
  url: '/openapi.yaml',
  dom_id: '#swagger',
  presets: [
    SwaggerUIBundle.presets.apis,
    SwaggerUIBundle.SwaggerUIStandalonePreset
  ],
  layout: "BaseLayout"
});
</script>

</body>
</html>`
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// GET /docs/json
func OpenAPIJSON(w http.ResponseWriter, r *http.Request) {
	yamlBytes, err := os.ReadFile("openapi.yaml")
	if err != nil {
		http.Error(w, `{"error":"openapi.yaml not found"}`, http.StatusInternalServerError)
		return
	}

	var jsonObj any
	if err := yaml.Unmarshal(yamlBytes, &jsonObj); err != nil {
		http.Error(w, `{"error":"failed to parse yaml"}`, http.StatusInternalServerError)
		return
	}

	jsonBytes, err := json.MarshalIndent(jsonObj, "", "  ")
	if err != nil {
		http.Error(w, `{"error":"failed to convert to json"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// caching rule
	if os.Getenv("APP_ENV") != "development" {
		w.Header().Set("Cache-Control", "public, max-age=3600")
	}

	w.Write(jsonBytes)
}
