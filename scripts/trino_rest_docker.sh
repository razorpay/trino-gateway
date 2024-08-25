#!/bin/bash

if [[ $@ == "up" ]]; then
    docker-compose -f build/docker/dev/trino_rest/docker-compose.yml up -d --build
elif [[ $@ == "down" ]]; then
    docker-compose -f build/docker/dev/trino_rest/docker-compose.yml down
elif [[ $@ == "logs" ]]; then
    docker-compose -f build/docker/dev/trino_rest/docker-compose.yml logs -f
else
    docker-compose -f build/docker/dev/trino_rest/docker-compose.yml down --rmi all --volumes --remove-orphans
fi
