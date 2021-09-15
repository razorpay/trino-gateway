#!/bin/bash

initialize() {
    # Init app secrets + envvars
    echo "Syncing app deps, if this takes time, update deps in the built image"
    make build
}

check_db_connection() {
    # Wait till db is available
    echo "GG"
}

db_migrations() {
    go run ./cmd/migration/main.go up
}

initialize
check_db_connection
# run db migrations
db_migrations

# run app

go run ./cmd/gateway/main.go
# tail -f /dev/null
