# Proto repo info
PROTO_GIT_URL := "https://github.com/razorpay/proto"

# Accept a branch from which we need to checkout proto files. Useful for dev testing.
PROTO_BRANCH ?= master

PROTO_REMOTE_DIR := common/error

# Proto gen info
PROTO_SRC := proto/
PROTO_DEST := response/

# This is the only variable that ever should change.
# This can be a branch, tag, or commit.
# When changed, the given version of Prototool will be installed to
# .tmp/$(uname -s)/(uname -m)/bin/prototool
PROTOTOOL_VERSION := v1.9.0
PROTOC_GEN_GO_VERSION := v1.4.2

VERBOSE = 0
Q 		= $(if $(filter 1,$VERBOSE),,@)
M 		= $(shell printf "\033[34;1m▶\033[0m")

$(BIN)/%: | $(BIN) ; $(info $(M) building package: $(PACKAGE)…)
	tmp=$$(mktemp -d); \
	   env GOBIN=$(BIN) go get $(PACKAGE) \
		|| ret=$$?; \
	   rm -rf $$tmp ; exit $$ret

.PHONY: proto-fetch ## Fetch proto files frrm remote repo
proto-fetch: ## Fetch proto files from remote repo
	@echo "\n + Fetching proto files from branch: $(PROTO_BRANCH) \n"
	@mkdir $(PROTO_SRC) && \
	cd $(PROTO_SRC) && \
	git init --quiet && \
	git config core.sparseCheckout true && \
	echo $(PROTO_REMOTE_DIR) > .git/info/sparse-checkout && \
	git remote add origin $(PROTO_GIT_URL)  && \
	git fetch origin $(PROTO_BRANCH) --quiet && \
	git checkout origin/$(PROTO_BRANCH) --quiet

.PHONY: deps
deps:
	@echo "\n + Fetching dependencies \n"
	@go get google.golang.org/grpc
	@go get github.com/golang/protobuf/protoc-gen-go@$(PROTOC_GEN_GO_VERSION)

.PHONY: proto-generate
proto-generate:
	@echo "\n + Generating protobuf language bindings\n"
	@protoc --proto_path=proto --go_out=response proto/common/error/*/*

.PHONY: proto-refresh ## Download and re-compile protobuf
proto-refresh: clean proto-fetch proto-generate ## Fetch proto files from remote repo

.PHONY: clean ## Remove previous builds, protobuf files, and proto compiled code
clean:
	@echo " + Removing cloned and generated files\n"
	@rm -rf $(PROTO_SRC)
