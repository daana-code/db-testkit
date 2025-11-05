# db-testkit

A Go library for managing standardized database test infrastructure across projects. Eliminates code duplication by providing a single source of truth for Docker-based PostgreSQL test environments.

[![Go Reference](https://pkg.go.dev/badge/github.com/daana-code/db-testkit.svg)](https://pkg.go.dev/github.com/daana-code/db-testkit)
[![License](https://img.shields.io/github/license/daana-code/db-testkit)](LICENSE)

## What is db-testkit?

db-testkit provides:

- **Standard Database Template**: A ready-to-use `docker-compose.databases.yml` with 4 databases (dev + test pairs)
- **Configuration Parser**: Extract credentials from any docker-compose.yml
- **Code Generators**: Auto-generate Taskfile.yml, Go constants, and connection profiles
- **Health Check Scripts**: Robust Docker health monitoring and verification
- **Zero Duplication**: Single source of truth for database configuration

## Features

- 🎯 **Single Source of Truth**: `docker-compose.yml` defines all credentials
- 🔄 **Auto-Generation**: Generate Taskfile, Go constants, and connection profiles automatically
- 🐳 **Docker-Based**: Standardized PostgreSQL 15 containers with health checks
- 📦 **Library-Only**: Import as a Go package, no extra binaries needed
- 🎨 **Flexible**: Use the provided template or parse your own docker-compose.yml
- 🧪 **Test-Ready**: Separate dev and test database pairs for clean testing

## Quick Start

### Option 1: Use the Standard Template

1. **Add db-testkit as a dependency**:
```bash
go get github.com/daana-code/db-testkit
```

2. **Use the provided docker-compose template**:
```yaml
# docker-compose.yml
include:
  - path: ../db-testkit/templates/docker-compose.databases.yml

services:
  # Add your project-specific services here
```

Or copy the template to your project:
```bash
cp ../db-testkit/templates/docker-compose.databases.yml ./docker-compose.yml
```

3. **Generate configuration files** (in your CLI tool):
```go
import (
    "github.com/daana-code/db-testkit/pkg/docker"
    "github.com/daana-code/db-testkit/pkg/generator"
)

func generateConfig() error {
    // Parse docker-compose.yml
    compose, err := docker.ParseDockerCompose("docker-compose.yml")
    if err != nil {
        return err
    }

    // Extract credentials
    creds, err := docker.ExtractCredentials(compose)
    if err != nil {
        return err
    }

    // Generate files
    generator.GenerateTaskfile(creds, "Taskfile.generated.yml")
    generator.GenerateGoConstants(creds, "internal/project/generated_config.go")
    generator.GenerateConnectionProfiles(creds, "testdata/connection-profiles/connection-profiles-test.yaml")

    return nil
}
```

### Option 2: Parse Your Own docker-compose.yml

If you have a custom docker-compose.yml, db-testkit can parse it as long as it contains `db-test-customer` and `db-test-internal` services:

```yaml
# Your custom docker-compose.yml
services:
  db-test-customer:
    image: postgres:15
    environment:
      POSTGRES_USER: myuser
      POSTGRES_PASSWORD: mypass
      POSTGRES_DB: mydb
    ports:
      - "5555:5432"
    # ... other config

  db-test-internal:
    # ... similar structure
```

Then use the same generation code as above.

## Standard Database Setup

The provided template includes 4 databases:

| Database | Port | User | Password | Database Name | Purpose |
|----------|------|------|----------|---------------|---------|
| **db** | 5432 | dev | devpass | customerdb | Dev customer database |
| **db-internal** | 5434 | dev | devpass | internaldb | Dev metadata database |
| **db-test-customer** | 5555 | autotester | autotestpass | testcustomerdb | Test customer database |
| **db-test-internal** | 6666 | autotester | autotestpass | testinternaldb | Test metadata database |

All databases:
- Use PostgreSQL 15
- Have health checks configured
- Use persistent volumes
- Support SQL seed files via `/docker-entrypoint-initdb.d/`

## Seed Data

db-testkit provides centralized seed data that can be shared across multiple projects. The seed data is stored in `testdata/seeds/` and includes Taskfile tasks for flexible loading and management.

### Available Seed Files

- **olist.sql**: Comprehensive Brazilian e-commerce dataset (Olist) with customers, orders, payments, reviews, products, sellers, and geolocation data in the `stage` schema.

### Using Seed Data (Recommended: Taskfile Tasks)

db-testkit includes a Taskfile with convenient tasks for loading, verifying, and reloading seed data. This is the **recommended approach** because it:
- Can load seed data anytime (not just on first initialization)
- Allows selective reloading without destroying all database data
- Provides verification tasks to check data loaded correctly
- Works across all environments (dev and test databases)

```bash
# Load olist seed into both dev and test customer databases
task seed:load:olist

# Load into specific environment
task seed:load:olist:dev        # Dev database only
task seed:load:olist:test       # Test database only

# Verify seed data loaded correctly
task seed:verify:olist           # Check both databases
task seed:verify:olist:dev       # Check dev only

# Reload seed data (drops stage schema and reloads)
task seed:reload:olist           # Reload both databases
task seed:reload:olist:dev       # Reload dev only

# Clean up seed data
task seed:clean                  # Drop stage schema from all databases
task seed:clean:dev              # Drop from dev only

# List available seed files
task seed:list

# Check database status
task db:status                   # All databases
task db:status:dev               # Dev databases only
```

**Note**: These tasks work from the db-testkit directory. Projects can call them using:
```bash
# From your project directory
cd ../db-testkit && task seed:load:olist

# Or use task's -t flag
task -t ../db-testkit seed:load:olist
```

### Alternative: Volume Mounting (First Initialization Only)

You can also mount seed files into the PostgreSQL initialization directory for automatic loading on first container startup:

```yaml
# docker-compose.yml
services:
  db:
    image: postgres:15
    volumes:
      - pg-customer-data:/var/lib/postgresql/data
      - ../db-testkit/testdata/seeds/olist.sql:/docker-entrypoint-initdb.d/olist.sql
```

**Note**: Volume-mounted seed files only run on first initialization. To reload, you must remove volumes:
```bash
docker compose down -v  # Remove volumes (destroys ALL data!)
docker compose up -d    # Recreate with seed data
```

This approach is less flexible than using Taskfile tasks, but useful for automated CI/CD pipelines.

## Generated Files

### 1. Taskfile.generated.yml

Generated Taskfile with database credentials and common tasks:

```yaml
vars:
  TEST_CUSTOMER_HOST: "localhost"
  TEST_CUSTOMER_PORT: "5555"
  TEST_CUSTOMER_USER: "autotester"
  # ... all credentials

tasks:
  test:db:start:generated:
    desc: Start test databases with health checks
    cmds:
      - docker compose up -d db-test-customer db-test-internal
      - ./scripts/wait-for-healthy.sh pg-test-customer 90
      - ./scripts/wait-for-healthy.sh pg-test-internal 90
```

### 2. Generated Go Constants

Type-safe Go constants in `internal/project/generated_config.go`:

```go
const (
    DefaultCustomerHost     = "localhost"
    DefaultCustomerPort     = 5555
    DefaultCustomerUser     = "autotester"
    DefaultCustomerPassword = "autotestpass"
    DefaultCustomerDB       = "testcustomerdb"
    // ... internal database constants
)
```

### 3. Connection Profiles YAML

Database connection profiles for testing:

```yaml
connection_profiles:
  test:
    type: "postgresql"
    host: "localhost"
    port: 5555
    user: "autotester"
    password: "autotestpass"
    database: "testcustomerdb"
```

## Health Check Scripts

db-testkit includes two robust health check scripts in the `scripts/` directory:

### wait-for-healthy.sh

Event-based Docker health monitoring with no arbitrary sleeps:

```bash
./scripts/wait-for-healthy.sh pg-test-customer 90
```

Features:
- Uses Docker events for instant notification
- Cross-platform timeout handling (macOS/Linux)
- Clear error messages
- No polling or arbitrary sleeps

### verify-database-health.sh

Comprehensive database verification:

```bash
./scripts/verify-database-health.sh
```

Features:
- Extracts credentials from docker-compose.yml
- Tests database connectivity
- Verifies schema creation
- Reports detailed status

**Note**: Copy these scripts to your project's `scripts/` directory to use them.

## Integration Example

Here's a complete example of integrating db-testkit into your CLI tool:

```go
// cmd/dev/generate.go
package main

import (
    "fmt"
    "path/filepath"

    "github.com/daana-code/db-testkit/pkg/docker"
    "github.com/daana-code/db-testkit/pkg/generator"
    "github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
    Use:   "generate",
    Short: "Generate test configuration from docker-compose.yml",
    RunE:  runGenerate,
}

func runGenerate(cmd *cobra.Command, args []string) error {
    fmt.Println("Generating configuration from docker-compose.yml...")

    // Parse docker-compose.yml
    compose, err := docker.ParseDockerCompose("docker-compose.yml")
    if err != nil {
        return fmt.Errorf("failed to parse docker-compose.yml: %w", err)
    }

    // Extract credentials
    creds, err := docker.ExtractCredentials(compose)
    if err != nil {
        return fmt.Errorf("failed to extract credentials: %w", err)
    }

    // Generate all configuration files
    if err := generator.GenerateConnectionProfiles(creds,
        "testdata/connection-profiles/connection-profiles-test.yaml"); err != nil {
        return err
    }

    if err := generator.GenerateGoConstants(creds,
        "internal/project/generated_config.go"); err != nil {
        return err
    }

    if err := generator.GenerateTaskfile(creds,
        "Taskfile.generated.yml"); err != nil {
        return err
    }

    fmt.Println("✓ Generated testdata/connection-profiles/connection-profiles-test.yaml")
    fmt.Println("✓ Generated internal/project/generated_config.go")
    fmt.Println("✓ Generated Taskfile.generated.yml")

    return nil
}
```

## Workflow

```
1. Define databases in docker-compose.yml (or use provided template)
           ↓
2. Run your CLI's generate command (which uses db-testkit)
           ↓
3. db-testkit parses docker-compose.yml
           ↓
4. Generates: Taskfile.yml + Go constants + connection profiles
           ↓
5. Start databases: docker compose up -d
           ↓
6. Run tests with auto-generated credentials
```

## Code Reduction Example

**Before db-testkit** (focal-cli):
- `cmd/dev/generate.go`: ~425 lines
- Duplicated across multiple projects

**After db-testkit**:
- `cmd/dev/generate.go`: ~140 lines (67% reduction)
- Shared logic: ~400 lines in db-testkit
- Zero duplication across projects

## Benefits

- **Zero Duplication**: One library, many projects
- **Type Safety**: Go constants prevent typos
- **Single Source of Truth**: All credentials in docker-compose.yml
- **Maintainable**: Update once, propagate everywhere
- **Testable**: Separate dev and test environments
- **Version Controlled**: Lock to specific db-testkit versions

## Requirements

- Go 1.21+
- Docker & Docker Compose
- PostgreSQL 15 (automatically provided via Docker)

## Development Setup

For local development, use a `replace` directive in your `go.mod`:

```go
replace github.com/daana-code/db-testkit => /path/to/local/db-testkit
```

This allows you to make changes to db-testkit and test them immediately in your project.

## Package Structure

```
db-testkit/
├── pkg/
│   ├── docker/
│   │   ├── config.go    # Types: DockerCompose, Service, TestDBCredentials
│   │   └── compose.go   # ParseDockerCompose(), ExtractCredentials()
│   └── generator/
│       ├── taskfile.go  # GenerateTaskfile()
│       ├── goconfig.go  # GenerateGoConstants()
│       └── profiles.go  # GenerateConnectionProfiles()
├── templates/
│   └── docker-compose.databases.yml  # Standard 4-database template
└── scripts/
    ├── wait-for-healthy.sh           # Event-based health waiting
    └── verify-database-health.sh     # Comprehensive verification
```

## License

[MIT License](LICENSE)

## Contributing

Contributions welcome! This library is designed to be simple and focused. If you have ideas for improvements, please open an issue first to discuss.

## Support

For issues, questions, or feature requests, please open an issue on GitHub.
