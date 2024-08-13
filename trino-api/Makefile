DOCKER_DIR := "./deployment/dev"
# Dir where build binaries are generated. The dir should be gitignored
BUILD_OUT_DIR := "bin/"

API_OUT       := "bin/api"
API_MAIN_FILE := "cmd/api/main.go"

GOVERSION=$(shell go version)
UNAME_OS=$(shell go env GOOS)
UNAME_ARCH=$(shell go env GOARCH)

GO       = go

MODULE   = $(shell $(GO) list -m)
SERVICE  = $(shell basename $(MODULE))

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

.PHONY: go-build-api ## Build the binary file for API server
go-build-api:
	@CGO_ENABLED=0 GOOS=$(UNAME_OS) GOARCH=$(UNAME_ARCH) go build -v -o $(API_OUT) $(API_MAIN_FILE)

.PHONY: local_dev_rebuild
local_dev_rebuild:
	docker-compose -f deployment/dev/docker-compose.yml up -d --build

.PHONY: local_dev_build
# Build the local_dev Docker image
local_dev_build:
	docker-compose -f $(DOCKER_DIR)/docker-compose.yml build


.PHONY: local_dev_run
# Run the docker container services
local_dev_run:
	docker-compose -f $(DOCKER_DIR)/docker-compose.yml up -d


.PHONY: local_dev_down
# Stop the docker container services
local_dev_down:
	docker-compose -f $(DOCKER_DIR)/docker-compose.yml down


.PHONY: local_dev_logs
# View logs for the docker services
local_dev_logs:
	docker-compose -f $(DOCKER_DIR)/docker-compose.yml logs -f


.PHONY: local_dev_test
# Run tests inside the API docker container
local_dev_test:
	docker-compose -f $(DOCKER_DIR)/docker-compose.yml exec api go test ./...


.PHONY: local_dev_clean
# Clean up Docker container resources
local_dev_clean:
	docker-compose -f $(DOCKER_DIR)/docker-compose.yml down --rmi all --volumes --remove-orphans
