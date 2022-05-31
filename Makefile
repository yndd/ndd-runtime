# ====================================================================================
# Setup Project

PROJECT_NAME ?= ndd-runtime
REPO ?= yndd
PROJECT_REPO ?= github.com/$(REPO)/$(PROJECT_NAME)

.PHONY: all
all: generate

.PHONY: generate
generate: protoc-gen-gofast protoc-gen-golang-deepcopy fmt vet
	go mod vendor
	protoc -I . -I ./vendor $(shell find ./apis/ -name '*.proto') --gofast_out=. --gofast_opt=paths=source_relative  --golang-deepcopy_out=. --golang-deepcopy_opt=paths=source_relative

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./... | grep -v vendor/ && exit 1 || exit 0

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
PROTOC_GO_FAST ?= $(LOCALBIN)/protoc-gen-gofast
PROTOC_GO_DEEPCOPY ?= $(LOCALBIN)/protoc-gen-golang-deepcopy

## Tool Versions
CONTROLLER_TOOLS_VERSION ?= v0.8.0
PROTOC_GO_FAST_VERSION ?= latest
PROTOC_GO_DEEPCOPY_VERSION ?= latest

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: protoc-gen-gofast
protoc-gen-gofast: $(PROTOC_GO_FAST) ## Download protoc-gen-gofast locally if necessary.
$(PROTOC_GO_FAST): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install github.com/gogo/protobuf/protoc-gen-gofast@$(PROTOC_GO_FAST_VERSION)

.PHONY: protoc-gen-golang-deepcopy
protoc-gen-golang-deepcopyt: $(PROTOC_GO_DEEPCOPY) ## Download protoc-gen-golang-deepcopy locally if necessary.
$(PROTOC_GO_DEEPCOPY): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install github.com/protobuf-tools/protoc-gen-deepcopy@$(PROTOC_GO_DEEPCOPY_VERSION)



