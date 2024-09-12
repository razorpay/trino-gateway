#!/bin/bash

initialize() {
    # Init app secrets + envvars
    echo "Initializing app"
}

check_db_connection() {
    # Wait till db is available
    connected=0
    counter=0

    echo "Wait 60 seconds for connection to MySQL"
    while [[ ${counter} -lt 60 ]]; do
        {
            echo "Connecting to MySQL" && go run ./cmd/migration/main.go version &&
            connected=1

        } || {
            let counter=$counter+3
            sleep 3
        }
        if [[ ${connected} -eq 1 ]]; then
            echo "Connected"
            break;
        fi
    done

    if [[ ${connected} -eq 0 ]]; then
        echo "MySQL connection failed."
        exit;
    fi
}

db_migrations() {
    go run ./cmd/migration/main.go up
}

initialize
# check_db_connection
# run db migrations
# db_migrations

# run app

# CompileDaemon -polling-interval=10 -exclude-dir=.git -exclude-dir=vendor --build="gopherjs build ./internal/frontend/main --output "./web/frontend/js/frontend.js" --verbose && go build cmd/gateway/main.go -o gateway" --command=./gateway
go run ./cmd/trino_rest/main.go
# tail -f /dev/null
