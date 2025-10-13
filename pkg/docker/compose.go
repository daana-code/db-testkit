package docker

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ParseDockerCompose reads and parses a docker-compose.yml file.
func ParseDockerCompose(path string) (*DockerCompose, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read docker-compose.yml: %w", err)
	}

	var dockerCompose DockerCompose
	if err := yaml.Unmarshal(data, &dockerCompose); err != nil {
		return nil, fmt.Errorf("failed to parse docker-compose.yml: %w", err)
	}

	return &dockerCompose, nil
}

// ExtractCredentials extracts database credentials from a parsed docker-compose configuration.
// It looks for db-test-customer and db-test-internal services and extracts their PostgreSQL
// environment variables and port mappings.
func ExtractCredentials(dockerCompose *DockerCompose) (*TestDBCredentials, error) {
	customerService, ok := dockerCompose.Services["db-test-customer"]
	if !ok {
		return nil, fmt.Errorf("db-test-customer service not found in docker-compose.yml")
	}

	internalService, ok := dockerCompose.Services["db-test-internal"]
	if !ok {
		return nil, fmt.Errorf("db-test-internal service not found in docker-compose.yml")
	}

	// Extract customer credentials
	customerUser := customerService.Environment["POSTGRES_USER"]
	customerPassword := customerService.Environment["POSTGRES_PASSWORD"]
	customerDB := customerService.Environment["POSTGRES_DB"]

	// Extract port from port mapping (e.g., "5555:5432" -> "5555")
	customerPort := extractHostPort(customerService.Ports, "5555")

	// Extract internal credentials
	internalUser := internalService.Environment["POSTGRES_USER"]
	internalPassword := internalService.Environment["POSTGRES_PASSWORD"]
	internalDB := internalService.Environment["POSTGRES_DB"]

	// Extract port from port mapping (e.g., "6666:5432" -> "6666")
	internalPort := extractHostPort(internalService.Ports, "6666")

	return &TestDBCredentials{
		CustomerHost:     "localhost",
		CustomerPort:     customerPort,
		CustomerUser:     customerUser,
		CustomerPassword: customerPassword,
		CustomerDB:       customerDB,
		InternalHost:     "localhost",
		InternalPort:     internalPort,
		InternalUser:     internalUser,
		InternalPassword: internalPassword,
		InternalDB:       internalDB,
	}, nil
}

// extractHostPort extracts the host port from a port mapping string.
// For example, "5555:5432" returns "5555".
// If no port mapping is found, returns the default port.
func extractHostPort(ports []string, defaultPort string) string {
	if len(ports) == 0 {
		return defaultPort
	}

	port := ports[0]
	for i, c := range port {
		if c == ':' {
			return port[:i]
		}
	}

	return defaultPort
}
