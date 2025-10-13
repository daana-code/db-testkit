#!/bin/bash

# Database Health Verification Script
# Extracts credentials from docker-compose.yml (single source of truth)
# This script provides robust verification of database connectivity and schema health

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
DOCKER_COMPOSE_FILE="$PROJECT_ROOT/docker-compose.yml"

# Function to extract value from docker-compose.yml
extract_compose_value() {
    local service=$1
    local key=$2
    grep -A 20 "$service:" "$DOCKER_COMPOSE_FILE" | grep "POSTGRES_$key:" | sed 's/.*POSTGRES_'"$key"': //' | tr -d ' '
}

# Function to extract port from docker-compose.yml
extract_compose_port() {
    local service=$1
    grep -A 20 "^  $service:" "$DOCKER_COMPOSE_FILE" | grep "ports:" -A 1 | tail -1 | sed 's/.*"\([0-9]*\):.*/\1/' | tr -d ' '
}

# Extract credentials from docker-compose.yml
echo -e "${BLUE}📖 Reading credentials from docker-compose.yml...${NC}"

# Customer database (test)
CUSTOMER_HOST="localhost"
CUSTOMER_PORT=$(extract_compose_port "db-test-customer")
CUSTOMER_USER=$(extract_compose_value "db-test-customer" "USER")
CUSTOMER_PASSWORD=$(extract_compose_value "db-test-customer" "PASSWORD")
CUSTOMER_DB=$(extract_compose_value "db-test-customer" "DB")

# Internal database (test)
INTERNAL_HOST="localhost"
INTERNAL_PORT=$(extract_compose_port "db-test-internal")
INTERNAL_USER=$(extract_compose_value "db-test-internal" "USER")
INTERNAL_PASSWORD=$(extract_compose_value "db-test-internal" "PASSWORD")
INTERNAL_DB=$(extract_compose_value "db-test-internal" "DB")

echo -e "${BLUE}Customer DB: ${CUSTOMER_USER}@${CUSTOMER_HOST}:${CUSTOMER_PORT}/${CUSTOMER_DB}${NC}"
echo -e "${BLUE}Internal DB: ${INTERNAL_USER}@${INTERNAL_HOST}:${INTERNAL_PORT}/${INTERNAL_DB}${NC}"

# Function to print status
print_status() {
    local status=$1
    local message=$2
    case $status in
        "SUCCESS")
            echo -e "${GREEN}✅ $message${NC}"
            ;;
        "WARNING")
            echo -e "${YELLOW}⚠️  $message${NC}"
            ;;
        "ERROR")
            echo -e "${RED}❌ $message${NC}"
            ;;
        "INFO")
            echo -e "${BLUE}ℹ️  $message${NC}"
            ;;
    esac
}

# Function to test database connectivity
test_db_connection() {
    local host=$1
    local port=$2
    local user=$3
    local password=$4
    local database=$5
    local name=$6

    print_status "INFO" "Testing $name database connection..."

    local container_name
    if [ "$name" = "Customer" ]; then
        container_name="pg-test-customer"
    else
        container_name="pg-test-internal"
    fi

    if docker exec "$container_name" pg_isready -h localhost -U "$user" -d "$database" -t 5 > /dev/null 2>&1; then
        print_status "SUCCESS" "$name database ($host:$port) is accepting connections"
        return 0
    else
        print_status "ERROR" "$name database ($host:$port) is not responding"
        return 1
    fi
}

# Function to verify schema existence
verify_schema() {
    local container=$1
    local user=$2
    local database=$3
    local schema_name=$4
    local description=$5

    local count=$(docker exec "$container" psql -U "$user" -d "$database" -t -c "SELECT COUNT(*) FROM information_schema.schemata WHERE schema_name='$schema_name';" 2>/dev/null | tr -d ' \n' | head -1)

    # Default to 0 if count is empty
    if [ -z "$count" ]; then
        count="0"
    fi

    if [ "$count" -eq "1" ]; then
        print_status "SUCCESS" "$description schema '$schema_name' exists"
        return 0
    else
        print_status "ERROR" "$description schema '$schema_name' missing"
        return 1
    fi
}

# Function to verify table count
verify_table_count() {
    local container=$1
    local user=$2
    local database=$3
    local schema_name=$4
    local expected_count=$5
    local description=$6

    local count=$(docker exec "$container" psql -U "$user" -d "$database" -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='$schema_name';" 2>/dev/null | tr -d ' \n' | head -1)

    # Default to 0 if count is empty
    if [ -z "$count" ]; then
        count="0"
    fi

    if [ "$count" -eq "$expected_count" ]; then
        print_status "SUCCESS" "$description has $count/$expected_count tables"
        return 0
    else
        print_status "WARNING" "$description has $count/$expected_count tables"
        return 1
    fi
}

# Function to verify specific table existence
verify_table_exists() {
    local container=$1
    local user=$2
    local database=$3
    local schema_name=$4
    local table_name=$5
    local description=$6

    local count=$(docker exec "$container" psql -U "$user" -d "$database" -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='$schema_name' AND table_name='$table_name';" 2>/dev/null | tr -d ' \n' | head -1)

    # Default to 0 if count is empty
    if [ -z "$count" ]; then
        count="0"
    fi

    if [ "$count" -eq "1" ]; then
        print_status "SUCCESS" "$description table '$table_name' exists"
        return 0
    else
        print_status "ERROR" "$description table '$table_name' missing"
        return 1
    fi
}

# Main verification function
main() {
    echo -e "${BLUE}🔍 Starting comprehensive database health verification...${NC}\n"

    local overall_status=0

    # Test container availability
    print_status "INFO" "Checking Docker containers..."
    if ! docker ps --format "table {{.Names}}" | grep -q "pg-test-customer"; then
        print_status "ERROR" "Customer database container not running"
        overall_status=1
    fi

    if ! docker ps --format "table {{.Names}}" | grep -q "pg-test-internal"; then
        print_status "ERROR" "Internal database container not running"
        overall_status=1
    fi

    # Test database connections
    print_status "INFO" "Testing database connectivity..."
    test_db_connection "$CUSTOMER_HOST" "$CUSTOMER_PORT" "$CUSTOMER_USER" "$CUSTOMER_PASSWORD" "$CUSTOMER_DB" "Customer" || overall_status=1
    test_db_connection "$INTERNAL_HOST" "$INTERNAL_PORT" "$INTERNAL_USER" "$INTERNAL_PASSWORD" "$INTERNAL_DB" "Internal" || overall_status=1

    # Verify customer database schemas
    print_status "INFO" "Verifying customer database schemas..."
    verify_schema "pg-test-customer" "$CUSTOMER_USER" "$CUSTOMER_DB" "daana_dw" "Customer data warehouse" || overall_status=1
    verify_schema "pg-test-customer" "$CUSTOMER_USER" "$CUSTOMER_DB" "daana_metadata" "Customer metadata" || overall_status=1
    verify_schema "pg-test-customer" "$CUSTOMER_USER" "$CUSTOMER_DB" "stage" "Customer staging" || overall_status=1

    # Verify internal database schemas
    print_status "INFO" "Verifying internal database schemas..."
    verify_schema "pg-test-internal" "$INTERNAL_USER" "$INTERNAL_DB" "frontend" "Internal frontend" || overall_status=1
    verify_schema "pg-test-internal" "$INTERNAL_USER" "$INTERNAL_DB" "daana_metadata" "Internal metadata" || overall_status=1
    verify_schema "pg-test-internal" "$INTERNAL_USER" "$INTERNAL_DB" "daana_stage" "Internal staging" || overall_status=1
    verify_schema "pg-test-internal" "$INTERNAL_USER" "$INTERNAL_DB" "daana_dw" "Internal data warehouse" || overall_status=1

    # Verify critical table counts
    print_status "INFO" "Verifying table counts..."
    verify_table_count "pg-test-internal" "$INTERNAL_USER" "$INTERNAL_DB" "frontend" "12" "Frontend schema" || overall_status=1

    # Verify specific critical tables
    print_status "INFO" "Verifying critical tables..."
    verify_table_exists "pg-test-internal" "$INTERNAL_USER" "$INTERNAL_DB" "frontend" "bim" "Frontend BIM" || overall_status=1
    verify_table_exists "pg-test-internal" "$INTERNAL_USER" "$INTERNAL_DB" "frontend" "src_to_target" "Frontend mapping" || overall_status=1
    verify_table_exists "pg-test-internal" "$INTERNAL_USER" "$INTERNAL_DB" "frontend" "master_process" "Frontend process" || overall_status=1

    # Summary
    echo ""
    if [ $overall_status -eq 0 ]; then
        print_status "SUCCESS" "All database health checks passed!"
        echo -e "${GREEN}🎉 Database verification completed successfully${NC}"
    else
        print_status "ERROR" "Some database health checks failed"
        echo -e "${RED}💥 Database verification failed - see errors above${NC}"
    fi

    return $overall_status
}

# Run main function
main "$@"