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
        echo "Connecting to MySQL"
        # `goose version` exits 1 with "no next version found" on a fresh DB
        # (goose_db_version table missing/empty), even though MySQL is
        # reachable. Treat that specific error as a successful probe so we
        # don't loop forever; `goose up` below bootstraps the table.
        version_output=$(go run ./cmd/migration/main.go version 2>&1)
        version_rc=$?

        if [[ ${version_rc} -eq 0 ]] || [[ "${version_output}" == *"no next version found"* ]]; then
            connected=1
            echo "Connected"
            break
        fi

        echo "${version_output}"
        let counter=$counter+3
        sleep 3
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
check_db_connection
# run db migrations
db_migrations

# run app

# CompileDaemon -polling-interval=10 -exclude-dir=.git -exclude-dir=vendor --build="gopherjs build ./internal/frontend/main --output "./web/frontend/js/frontend.js" --verbose && go build cmd/gateway/main.go -o gateway" --command=./gateway
go run ./cmd/gateway/main.go
# tail -f /dev/null
