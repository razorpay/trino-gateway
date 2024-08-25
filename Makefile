SCRIPT_DIR := "./scripts"
BUILD_OUT_DIR := "bin/"
API_OUT := "bin/api"
API_MAIN_FILE := "cmd/trino_rest/main.go"

GOVERSION=$(shell go version)
UNAME_OS=$(shell go env GOOS)
UNAME_ARCH=$(shell go env GOARCH)

GO = go

PWD = $(shell pwd)

.PHONY: setup
setup:
	$(SCRIPT_DIR)/setup.sh

.PHONY: dev-setup
dev-setup: setup config/dev.toml


config/dev.toml:
	touch $(PWD)/config/dev.toml

.PHONY: build
build:
	$(SCRIPT_DIR)/compile.sh

.PHONY: build-frontend
build-frontend: web/frontend/js/frontend.js

web/frontend/js/frontend.js:
	echo "Compiling frontend"
	gopherjs build ./internal/frontend --output "./web/frontend/js/frontend.js" --minify --verbose

# .PHONY: dev-build
# dev-build:
# 	$(SCRIPT_DIR)/dev.sh

.PHONY: dev-docker-up
dev-docker-up:
	$(SCRIPT_DIR)/docker.sh up

.PHONY: dev-docker-down
dev-docker-down:
	$(SCRIPT_DIR)/docker.sh down

.PHONY: dev-docker-run-example ## Runs bundled example in dev docker env
dev-docker-run-example:
	$(SCRIPT_DIR)/run-example.sh

.PHONY: dev-migration
dev-migration:
	go build ./cmd/migration/main.go -o migration.go
	./migration.go up

.PHONY: test-integration
test-integration:
	go test -tags=integration ./test/it -v -count=1

.PHONY: test-unit
test-unit:
	go test

.PHONY: go-build-trino-rest-api ## Build trino rest api
go-build-trino-rest-api:
	@CGO_ENABLED=0 GOOS=$(UNAME_OS) GOARCH=$(UNAME_ARCH) $(GO) build -v -o $(API_OUT) $(API_MAIN_FILE)

.PHONY: local_trino_rest_up
local_trino_rest_up:
	$(SCRIPT_DIR)/trino_rest_docker.sh up

.PHONY: local_trino_rest_down
local_trino_rest_down:
	$(SCRIPT_DIR)/trino_rest_docker.sh down

.PHONY: local_trino_rest_logs
local_trino_rest_logs:
	$(SCRIPT_DIR)/trino_rest_docker.sh logs

.PHONY: local_trino_rest_clean
local_trino_rest_clean:
	$(SCRIPT_DIR)/trino_rest_docker.sh clean
