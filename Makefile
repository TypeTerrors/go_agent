# Makefile for cds.agent.app

APP_NAME=agent
CMD_DIR=cmd/$(APP_NAME)
BIN_DIR=bin
BIN_PATH=$(BIN_DIR)/$(APP_NAME)
PKG=./...
TEST_PKG=./tests
RUN_ARGS?=--help
TEST_RUN?=
GOFILES=$(shell git ls-files '*.go')

RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[0;33m
BLUE=\033[0;34m
NO_COLOR=\033[0m

.PHONY: all init build build-mac build-linux build-windows run clean fmt fmt-check lint vet test testv test-one race tidy tools help

all: build

init:
	@echo "\n${BLUE}Ensuring Go modules are tidy...${NO_COLOR}"
	@go mod tidy

build:
	@mkdir -p $(BIN_DIR)
	@echo "\n${BLUE}Building $(APP_NAME)...${NO_COLOR}"
	@go build -o $(BIN_PATH) $(CMD_DIR)
	@echo "${GREEN}Built:${NO_COLOR} $(BIN_PATH)"

build-mac:
	@mkdir -p $(BIN_DIR)
	GOOS=darwin GOARCH=amd64 go build -o $(BIN_PATH)-darwin-amd64 $(CMD_DIR)
	GOOS=darwin GOARCH=arm64 go build -o $(BIN_PATH)-darwin-arm64 $(CMD_DIR)

build-linux:
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(BIN_PATH)-linux-amd64 $(CMD_DIR)
	GOOS=linux GOARCH=arm64 go build -o $(BIN_PATH)-linux-arm64 $(CMD_DIR)

build-windows:
	@mkdir -p $(BIN_DIR)
	GOOS=windows GOARCH=amd64 go build -o $(BIN_PATH)-windows-amd64.exe $(CMD_DIR)

run: build
	@echo "\n${BLUE}Running $(APP_NAME) $(RUN_ARGS)...${NO_COLOR}"
	@$(BIN_PATH) $(RUN_ARGS)

# VHS demo generation
.PHONY: demos

demos: docs/assets/agent-help.gif docs/assets/agent-list.gif docs/assets/agent-read.gif docs/assets/agent-write.gif docs/assets/agent-delete.gif

# Render rule
# Example: docs/assets/agent-help.gif depends on docs/vhs/agent-help.tape

docs/assets/%.gif: docs/vhs/%.tape
	@mkdir -p docs/assets
	vhs $<

clean:
	@echo "\n${YELLOW}Cleaning...${NO_COLOR}"
	@rm -rf $(BIN_DIR)

fmt:
	@gofmt -w .
	@goimports -w . || true

fmt-check:
	@diff -u <(echo -n) <(gofmt -l .) || true

lint:
	@echo "Running static checks"
	@go vet $(PKG)
	@staticcheck $(PKG) || true

vet:
	@go vet $(PKG)

test:
	@go test $(PKG)

race:
	@go test -race -v $(PKG)

# Run tests with verbose output limited to tests package
testv:
	@go test -v $(TEST_PKG)

# Run a single test or regex: make test-one TEST_RUN='^TestWriteReadFile$'
test-one:
	@test -n "$(TEST_RUN)" || (echo "Set TEST_RUN, e.g.: make test-one TEST_RUN='^TestWriteReadFile$'" && exit 1)
	@go test -v $(TEST_PKG) -run "$(TEST_RUN)"

tidy:
	@go mod tidy

# Install optional tools if missing
TOOLS := staticcheck goimports vhs

.PHONY: tools-install
tools: tools-install

tools-install:
	@for t in $(TOOLS); do \
		if ! command -v $$t >/dev/null 2>&1; then \
			echo "Installing $$t"; \
			case $$t in \
				staticcheck) go install honnef.co/go/tools/cmd/staticcheck@latest ;; \
				goimports) go install golang.org/x/tools/cmd/goimports@latest ;; \
			esac; \
		fi; \
	done

help:
	@echo "Targets:"
	@echo "  make build           Build CLI to $(BIN_PATH)"
	@echo "  make run RUN_ARGS=   Run CLI with args (default --help)"
	@echo "  make test            Run all tests"
	@echo "  make testv           Run tests in $(TEST_PKG) verbose"
	@echo "  make test-one TEST_RUN='^TestName$'  Run a single test"
	@echo "  make race            Run tests with -race"
	@echo "  make lint|vet        Static checks"
	@echo "  make fmt|fmt-check   Format or check formatting"
	@echo "  make tidy            go mod tidy"
	@echo "  make clean           Remove $(BIN_DIR)"
	@echo "  make build-<os>      Cross-compile (mac/linux/windows)"
	@echo "  make demos           Render VHS GIFs to docs/assets"
