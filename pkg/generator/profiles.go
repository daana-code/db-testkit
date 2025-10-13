// Package generator provides functionality for generating configuration files
// from docker-compose.yml credentials.
package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/eikolytics/db-testkit/pkg/docker"
)

// GenerateConnectionProfiles generates a connection-profiles-test.yaml file from database credentials.
// The output file is created at the specified path, with parent directories created as needed.
func GenerateConnectionProfiles(creds *docker.TestDBCredentials, outputPath string) error {
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	tmpl := `# Connection Profiles for Automated Testing
# 🤖 THIS FILE IS AUTO-GENERATED from docker-compose.yml
# DO NOT EDIT MANUALLY - Run 'go generate ./...' or your dev tool to regenerate
# Generated on: {{.Timestamp}}
# ---------------------------------------------------------------------------
connection_profiles:
  # Automated testing environment (references docker-compose.yml)
  test:
    type: "postgresql"
    host: "{{.CustomerHost}}"
    port: {{.CustomerPort}}  # From docker-compose.yml db-test-customer port
    user: "{{.CustomerUser}}"  # From docker-compose.yml POSTGRES_USER
    password: "{{.CustomerPassword}}"  # From docker-compose.yml POSTGRES_PASSWORD
    database: "{{.CustomerDB}}"  # From docker-compose.yml POSTGRES_DB
    sslmode: "disable"
    target_schema: "daana_dw"
`

	t, err := template.New("connection-profiles").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	data := struct {
		*docker.TestDBCredentials
		Timestamp string
	}{
		TestDBCredentials: creds,
		Timestamp:         time.Now().Format("2006-01-02 15:04:05 MST"),
	}

	return t.Execute(file, data)
}
