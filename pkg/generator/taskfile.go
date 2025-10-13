package generator

import (
	"fmt"
	"os"
	"text/template"
	"time"

	"github.com/eikolytics/db-testkit/pkg/docker"
)

// GenerateTaskfile generates a Taskfile.generated.yml with database tasks and credentials.
// The output file is created at the specified path.
func GenerateTaskfile(creds *docker.TestDBCredentials, outputPath string) error {
	tmpl := `# 🤖 THIS FILE IS AUTO-GENERATED from docker-compose.yml
# DO NOT EDIT MANUALLY - Run 'go generate ./...' or your dev tool to regenerate
# Generated on: {{.Timestamp}}
# Source: docker-compose.yml
# ---------------------------------------------------------------------------

version: '3'

vars:
  # Automated testing database credentials (from docker-compose.yml)
  TEST_CUSTOMER_HOST: "{{.CustomerHost}}"
  TEST_CUSTOMER_PORT: "{{.CustomerPort}}"
  TEST_CUSTOMER_USER: "{{.CustomerUser}}"
  TEST_CUSTOMER_PASSWORD: "{{.CustomerPassword}}"
  TEST_CUSTOMER_DB: "{{.CustomerDB}}"

  TEST_INTERNAL_HOST: "{{.InternalHost}}"
  TEST_INTERNAL_PORT: "{{.InternalPort}}"
  TEST_INTERNAL_USER: "{{.InternalUser}}"
  TEST_INTERNAL_PASSWORD: "{{.InternalPassword}}"
  TEST_INTERNAL_DB: "{{.InternalDB}}"

tasks:
  # Override database commands with generated credentials
  test:db:start:generated:
    desc: Start the automated testing PostgreSQL databases (using generated credentials)
    cmds:
      - echo "Starting automated testing PostgreSQL databases..."
      - docker compose up -d db-test-customer db-test-internal
      - echo "Waiting for test databases to become healthy..."
      - ./scripts/wait-for-healthy.sh pg-test-customer 90
      - ./scripts/wait-for-healthy.sh pg-test-internal 90
      - echo "✅ Automated testing databases are ready!"

  test:db:psql:generated:
    desc: Connect to automated testing customer PostgreSQL with psql (using generated credentials)
    cmds:
      - docker exec -it pg-test-customer psql -U {{.CustomerUser}} -d {{.CustomerDB}}

  test:db:psql:internal:generated:
    desc: Connect to automated testing internal PostgreSQL with psql (using generated credentials)
    cmds:
      - docker exec -it pg-test-internal psql -U {{.InternalUser}} -d {{.InternalDB}}
`

	t, err := template.New("taskfile").Parse(tmpl)
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
