#!/bin/bash

initialize() {
    # Init app secrets + envvars
    echo "Syncing app deps, if this takes time, update deps in the built image"
    go mod download
}

check_db_connection() {
    # Wait till db is available
    echo "GG"
}

db_migrations() {
    echo "GG"
}

initialize
check_db_connection
# run db migrations
db_migrations

# run app


tail -f /dev/null
