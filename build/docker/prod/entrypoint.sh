#!/bin/bash

initialize() {
    # Init app secrets + envvars
    echo "GG"
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
