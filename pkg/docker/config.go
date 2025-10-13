// Package docker provides functionality for parsing docker-compose.yml files
// and extracting database configuration.
package docker

// DockerCompose represents the structure we need from docker-compose.yml
type DockerCompose struct {
	Services map[string]Service `yaml:"services"`
}

// Service represents a single service definition in docker-compose.yml
type Service struct {
	Environment map[string]string `yaml:"environment"`
	Ports       []string          `yaml:"ports"`
}

// TestDBCredentials holds extracted credentials from docker-compose.yml
// for both customer and internal test databases.
type TestDBCredentials struct {
	CustomerHost     string
	CustomerPort     string
	CustomerUser     string
	CustomerPassword string
	CustomerDB       string
	InternalHost     string
	InternalPort     string
	InternalUser     string
	InternalPassword string
	InternalDB       string
}
