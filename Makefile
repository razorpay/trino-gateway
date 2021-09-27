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
	$(SCRIPT_DIR)/compile.sh

.PHONY: build-frontend
build-frontend: web/frontend/js/frontend.js

web/frontend/js/frontend.js:
	echo "Compiling frontend"
	gopherjs build ./internal/frontend/main --output "./web/frontend/js/frontend.js" --verbose

# .PHONY: dev-build
# dev-build:
# 	$(SCRIPT_DIR)/dev.sh

.PHONY: dev-docker-up
dev-docker-up:
	$(SCRIPT_DIR)/docker.sh up

.PHONY: dev-docker-down
dev-docker-down:
	$(SCRIPT_DIR)/docker.sh down


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
