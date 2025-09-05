# Makefile for cds.agent.app
#
# NOTE FOR NODE.JS DEVs:
# This Makefile plays a role similar to npm scripts in package.json.
# Think of targets like:
#   - make build   => npm run build
#   - make test    => npm test
#   - make lint    => npm run lint
#   - make run     => npm start (with RUN_ARGS for CLI flags)
# Use `make help` to see common tasks.

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
	@if command -v gum >/dev/null 2>&1; then \
		gum style --foreground 33 --bold "\nBuilding $(APP_NAME)..."; \
	else \
		echo "\n${BLUE}Building $(APP_NAME)...${NO_COLOR}"; \
	fi
	@go build -o $(BIN_PATH) $(CMD_DIR)
	@if command -v gum >/devnull 2>&1; then \
		gum style --foreground 82 --bold "Built: $(BIN_PATH)"; \
	else \
		echo "${GREEN}Built:${NO_COLOR} $(BIN_PATH)"; \
	fi

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
	@if command -v gum >/dev/null 2>&1; then \
		gum style --foreground 33 --bold "\nRunning $(APP_NAME) $(RUN_ARGS)..."; \
	else \
		echo "\n${BLUE}Running $(APP_NAME) $(RUN_ARGS)...${NO_COLOR}"; \
	fi
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
	@if command -v gum >/dev/null 2>&1; then \
		gum style --foreground 214 --bold "\nCleaning..."; \
	else \
		echo "\n${YELLOW}Cleaning...${NO_COLOR}"; \
	fi
	@rm -rf $(BIN_DIR)

fmt:
	@gofmt -w .
	@goimports -w . || true

fmt-check:
	@diff -u <(echo -n) <(gofmt -l .) || true

lint:
	@if command -v gum >/dev/null 2>&1; then \
		gum style --foreground 213 --bold "Running static checks"; \
	else \
		echo "Running static checks"; \
	fi
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
# Attempts in order: if Go is present, install via `go install`.
# Otherwise try OS package managers (brew on macOS, scoop/choco on Windows, apt/yum/pacman on Linux).
TOOLS := staticcheck goimports vhs gum

.PHONY: tools-install
tools: tools-install

tools-install:
	@echo "Detecting platform and installing tools as needed..."
	@for t in $(TOOLS); do \
		if command -v $$t >/dev/null 2>&1; then \
			echo "Already installed: $$t"; \
			continue; \
		fi; \
		if command -v go >/dev/null 2>&1; then \
			case $$t in \
				staticcheck) echo "Installing staticcheck via go"; go install honnef.co/go/tools/cmd/staticcheck@latest ;; \
				goimports) echo "Installing goimports via go"; go install golang.org/x/tools/cmd/goimports@latest ;; \
				vhs) echo "Installing vhs via go"; go install github.com/charmbracelet/vhs@latest ;; \
				gum) echo "Installing gum via go"; go install github.com/charmbracelet/gum@latest ;; \
			esac; \
			continue; \
		fi; \
		UNAME_S=$$(uname -s 2>/dev/null || echo Unknown); \
		case "$$UNAME_S" in \
			Darwin*) \
				if command -v brew >/dev/null 2>&1; then \
					echo "Installing $$t via brew"; \
					brew install $$t || true; \
				else \
					echo "Homebrew not found. Install Go from https://go.dev/dl/ then re-run 'make tools'"; \
					false; \
				fi ;; \
			Linux*) \
				if command -v apt-get >/dev/null 2>&1; then sudo apt-get update && sudo apt-get install -y $$t || true; \
				elif command -v dnf >/dev/null 2>&1; then sudo dnf install -y $$t || true; \
				elif command -v yum >/dev/null 2>&1; then sudo yum install -y $$t || true; \
				elif command -v pacman >/dev/null 2>&1; then sudo pacman -Sy --noconfirm $$t || true; \
				else echo "No supported package manager found. Install Go from https://go.dev/dl/ then re-run 'make tools'"; false; \
				fi ;; \
			*) \
				# Windows or unknown shell; try scoop/choco
				if command -v scoop >/dev/null 2>&1; then \
					echo "Installing $$t via scoop"; \
					scoop install $$t || true; \
				elif command -v choco >/dev/null 2>&1; then \
					echo "Installing $$t via choco"; \
					choco install -y $$t || true; \
				else \
					echo "On Windows, install Go (https://go.dev/dl/) or Scoop (https://scoop.sh) then re-run 'make tools'"; \
					false; \
				fi ;; \
		esac; \
	done || true
	@echo "Tools installation attempted. If some tools failed, please install Go from https://go.dev/dl/ and re-run 'make tools'"

help:
	@echo "Makefile scripts (like npm scripts):"
	@echo "  make build             Build CLI to $(BIN_PATH)"
	@echo "  make run RUN_ARGS=     Run CLI with args (default --help)"
	@echo "  make test              Run all tests"
	@echo "  make testv             Run tests in $(TEST_PKG) verbose"
	@echo "  make test-one TEST_RUN='^TestName$'  Run a single test"
	@echo "  make race              Run tests with -race"
	@echo "  make lint|vet          Static checks (go vet, staticcheck)"
	@echo "  make fmt|fmt-check     Format or check formatting (gofmt/goimports)"
	@echo "  make tidy              go mod tidy"
	@echo "  make clean             Remove $(BIN_DIR)"
	@echo "  make build-<os>        Cross-compile (mac/linux/windows)"
	@echo "  make demos             Render VHS GIFs to docs/assets"
	@echo "\nTIP: Install gum for pretty output: https://github.com/charmbracelet/gum"}]} />}니다
