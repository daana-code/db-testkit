package generator

import (
	"fmt"
	"os"
	"text/template"
	"time"

	"github.com/daana-code/db-testkit/pkg/docker"
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

  # Seed data management tasks (from db-testkit)
  # Dev database seed tasks (manual testing)
  seed:load:dev:generated:
    desc: Load seed data into dev customer database (configurable via SEED_DATA_PATH)
    cmds:
      - |
        SEED_FILE="${SEED_DATA_PATH:-../db-testkit/testdata/seeds/olist.sql}"
        echo "Checking seed data file: $SEED_FILE"
        if [ ! -f "$SEED_FILE" ]; then
          echo "⚠️  WARNING: Seed data file not found at: $SEED_FILE"
          echo "⚠️  Skipping seed data loading for dev database"
          echo "⚠️  To fix: Set SEED_DATA_PATH environment variable or ensure db-testkit is cloned"
          echo "⚠️  Example: export SEED_DATA_PATH=/path/to/your/seed.sql"
          exit 0
        fi
        echo "Loading seed data from $SEED_FILE into dev customer database..."
        if cat "$SEED_FILE" | docker exec -i pg-customer psql -U dev -d customerdb 2>&1 | grep -v "does not exist, skipping"; then
          echo "✓ Successfully loaded seed data into dev customer database"
        else
          echo "⚠️  WARNING: Failed to load seed data into dev customer database"
          echo "⚠️  This may be expected if the database is not running or the seed data has issues"
          exit 0
        fi

  seed:verify:dev:generated:
    desc: Verify seed data in dev customer database (from db-testkit)
    cmds:
      - echo "Verifying seed data in dev customer database..."
      - docker exec pg-customer psql -U dev -d customerdb -c "SELECT schemaname, relname as tablename, n_live_tup as row_count FROM pg_stat_user_tables WHERE schemaname = 'stage' AND relname LIKE 'olist%' ORDER BY relname;"

  seed:reload:dev:generated:
    desc: Reload seed data in dev customer database (from db-testkit)
    cmds:
      - echo "Dropping stage schema in dev customer database..."
      - docker exec pg-customer psql -U dev -d customerdb -c "DROP SCHEMA IF EXISTS stage CASCADE;"
      - task: seed:load:dev:generated

  seed:clean:dev:generated:
    desc: Clean seed data from dev customer database (from db-testkit)
    cmds:
      - echo "Dropping stage schema from dev customer database..."
      - docker exec pg-customer psql -U dev -d customerdb -c "DROP SCHEMA IF EXISTS stage CASCADE;"
      - echo "✓ Successfully dropped stage schema from dev customer database"

  # Test database seed tasks (automated testing)
  seed:load:test:generated:
    desc: Load seed data into test customer database (configurable via SEED_DATA_PATH)
    cmds:
      - |
        SEED_FILE="${SEED_DATA_PATH:-../db-testkit/testdata/seeds/olist.sql}"
        echo "Checking seed data file: $SEED_FILE"
        if [ ! -f "$SEED_FILE" ]; then
          echo "⚠️  WARNING: Seed data file not found at: $SEED_FILE"
          echo "⚠️  Skipping seed data loading for test database"
          echo "⚠️  To fix: Set SEED_DATA_PATH environment variable or ensure db-testkit is cloned"
          echo "⚠️  Example: export SEED_DATA_PATH=/path/to/your/seed.sql"
          exit 0
        fi
        echo "Loading seed data from $SEED_FILE into test customer database..."
        if cat "$SEED_FILE" | docker exec -i pg-test-customer psql -U {{.CustomerUser}} -d {{.CustomerDB}} 2>&1 | grep -v "does not exist, skipping"; then
          echo "✓ Successfully loaded seed data into test customer database"
        else
          echo "⚠️  WARNING: Failed to load seed data into test customer database"
          echo "⚠️  This may be expected if the database is not running or the seed data has issues"
          exit 0
        fi

  seed:verify:test:generated:
    desc: Verify seed data in test customer database (from db-testkit)
    cmds:
      - echo "Verifying seed data in test customer database..."
      - docker exec pg-test-customer psql -U {{.CustomerUser}} -d {{.CustomerDB}} -c "SELECT schemaname, relname as tablename, n_live_tup as row_count FROM pg_stat_user_tables WHERE schemaname = 'stage' AND relname LIKE 'olist%' ORDER BY relname;"

  seed:reload:test:generated:
    desc: Reload seed data in test customer database (from db-testkit)
    cmds:
      - echo "Dropping stage schema in test customer database..."
      - docker exec pg-test-customer psql -U {{.CustomerUser}} -d {{.CustomerDB}} -c "DROP SCHEMA IF EXISTS stage CASCADE;"
      - task: seed:load:test:generated

  seed:clean:test:generated:
    desc: Clean seed data from test customer database (from db-testkit)
    cmds:
      - echo "Dropping stage schema from test customer database..."
      - docker exec pg-test-customer psql -U {{.CustomerUser}} -d {{.CustomerDB}} -c "DROP SCHEMA IF EXISTS stage CASCADE;"
      - echo "✓ Successfully dropped stage schema from test customer database"

  # Both databases seed tasks
  seed:load:all:generated:
    desc: Load seed data into both dev and test customer databases (from db-testkit)
    cmds:
      - echo "Loading seed data into all customer databases..."
      - task: seed:load:dev:generated
      - task: seed:load:test:generated

  seed:verify:all:generated:
    desc: Verify seed data in both dev and test customer databases (from db-testkit)
    cmds:
      - task: seed:verify:dev:generated
      - echo ""
      - task: seed:verify:test:generated

  seed:reload:all:generated:
    desc: Reload seed data in both dev and test customer databases (from db-testkit)
    cmds:
      - task: seed:reload:dev:generated
      - task: seed:reload:test:generated

  seed:clean:all:generated:
    desc: Clean seed data from both dev and test customer databases (from db-testkit)
    cmds:
      - task: seed:clean:dev:generated
      - task: seed:clean:test:generated
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
