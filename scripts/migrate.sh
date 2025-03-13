#!/bin/bash

# Load environment variables from .env file if it exists
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

# Default values
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_USER=${DB_USER:-media_center}
DB_NAME=${DB_NAME:-media_center}
DB_PASSWORD=${DB_PASSWORD:-media_center}

# Function to run SQL file
run_sql_file() {
    local file=$1
    echo "Running $file..."
    PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f "$file"
}

# Function to check if database exists
check_database() {
    PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='$DB_NAME'" | grep -q 1
}

# Function to create database if it doesn't exist
create_database() {
    if ! check_database; then
        echo "Creating database $DB_NAME..."
        PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -c "CREATE DATABASE $DB_NAME"
    fi
}

# Function to handle errors
handle_error() {
    echo "Error: $1"
    exit 1
}

# Main migration function
migrate() {
    local action=$1
    local migrations_dir="database/migrations"

    # Create migrations directory if it doesn't exist
    mkdir -p "$migrations_dir"

    # Ensure database exists
    create_database || handle_error "Failed to create database"

    if [ "$action" = "down" ]; then
        # Run down migrations in reverse order
        echo "Running down migrations..."
        for file in $(ls -r "$migrations_dir"/*_down.sql 2>/dev/null); do
            run_sql_file "$file" || handle_error "Failed to run down migration: $file"
        done
    else
        # Run up migrations in order
        echo "Running up migrations..."
        for file in $(ls "$migrations_dir"/*.sql 2>/dev/null | grep -v '_down.sql$'); do
            run_sql_file "$file" || handle_error "Failed to run up migration: $file"
        done
    fi

    echo "Migrations completed successfully"
}

# Parse command line arguments
case "$1" in
    "up")
        migrate "up"
        ;;
    "down")
        migrate "down"
        ;;
    "reset")
        echo "Resetting database..."
        migrate "down"
        migrate "up"
        ;;
    *)
        echo "Usage: $0 {up|down|reset}"
        exit 1
        ;;
esac 