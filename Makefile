export SERVICE_NAME=klubyorg
export GOBIN := $(PWD)/bin
export PATH := $(GOBIN):$(PATH)

SHELL := env PATH=$(PATH) /bin/sh

INSTALL_TOOL_CMD=go install -modfile tools/go.mod

tools/go.mod:

./bin:
	mkdir -p ./bin

./bin/buf: tools/go.mod | ./bin
	$(INSTALL_TOOL_CMD) github.com/bufbuild/buf/cmd/buf

.PHONY: generate
generate:  ./bin/buf
	@go generate ./...
	@go mod tidy

.PHONY: serve
serve:
	@docker build . -t $(SERVICE_NAME)
	@docker run -p 8080:8080 $(SERVICE_NAME)
