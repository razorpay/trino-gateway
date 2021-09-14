SCRIPT_DIR := "./scripts"

BUILD_OUT_DIR := "bin/"

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
	$(SCRIPT_DIR)/setup.sh

.PHONY: dev-build
dev-build:
	$(SCRIPT_DIR)/dev.sh

.PHONY: dev-docker
dev-docker-up:
	$(SCRIPT_DIR)/dev.sh


.PHONY: dev-migration
dev-migration:
	go build ./cmd/migration/main.go -o migration.go
	./migration.go up
