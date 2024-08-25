SCRIPT_DIR := "./scripts"
TRINO_REST_DOCKER_DIR := "./build/docker/dev/trino_rest"
BUILD_OUT_DIR := "bin/"
API_OUT := "bin/api"
API_MAIN_FILE := "cmd/trino_rest/main.go"

GOVERSION=$(shell go version)
UNAME_OS=$(shell go env GOOS)
UNAME_ARCH=$(shell go env GOARCH)

GO = go

MODULE 	=$(shell $(GO) list -m)
SERVICE =$(shell basename $(MODULE))

BIN 	 = $(CURDIR)/bin
PKGS     = $(or $(PKG),$(shell $(GO) list ./...))

$(BIN)/%: | $(BIN) ; $(info $(M) building package: $(PACKAGE)…)
	tmp=$$(mktemp -d); \
	   env GOBIN=$(BIN) go install $(PACKAGE) \
		|| ret=$$?; \
	   rm -rf $$tmp ; exit $$ret

$(BIN)/golint: PACKAGE=golang.org/x/lint/golint

GOLINT = $(BIN)/golint

.PHONY: lint
lint: | $(GOLINT) ; $(info $(M) running golint…) @ ## Run golint
	$Q $(GOLINT) -set_exit_status $(PKGS)

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

.PHONY: lcoal_trino_rest_build
lcoal_trino_rest_build:
	docker-compose -f $(TRINO_REST_DOCKER_DIR)/docker-compose.yml build

.PHONY: lcoal_trino_rest_rebuild
lcoal_trino_rest_rebuild:
	docker-compose -f $(TRINO_REST_DOCKER_DIR)/docker-compose.yml up -d --build

.PHONY: lcoal_trino_rest_run
lcoal_trino_rest_run:
	docker-compose -f $(TRINO_REST_DOCKER_DIR)/docker-compose.yml up -d

.PHONY: lcoal_trino_rest_down
lcoal_trino_rest_down:
	docker-compose -f $(TRINO_REST_DOCKER_DIR)/docker-compose.yml down

.PHONY: lcoal_trino_rest_logs
lcoal_trino_rest_logs:
	docker-compose -f $(TRINO_REST_DOCKER_DIR)/docker-compose.yml logs -f

.PHONY: lcoal_trino_rest_test
lcoal_trino_rest_test:
	docker-compose -f $(TRINO_REST_DOCKER_DIR)/docker-compose.yml exec api $(GO) test ./..

.PHONY: lcoal_trino_rest_clean
lcoal_trino_rest_clean:
	docker-compose -f $(TRINO_REST_DOCKER_DIR)/docker-compose.yml down --rmi all --volumes --remove-orphans
