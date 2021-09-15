#!/bin/bash

if [[ $@ == "up" ]]; then
    docker-compose -f build/docker/dev/docker-compose.yml up -d --build 
else
    docker-compose -f build/docker/dev/docker-compose.yml down
fi