#!/bin/bash

initialize() {
    # Init app secrets + envvars
}

check_db_connection() {
    # Wait till db is available
}

db_migrations() {

}

initialize
check_db_connection
# run db migrations
db_migrations

# run app
