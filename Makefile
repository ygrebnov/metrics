ROOT_PATH := $(dir $(realpath $(lastword $(MAKEFILE_LIST))))
COVERAGE_PATH := $(ROOT_PATH).coverage/

# Force go to use the local toolchain instead of auto-downloading a toolchain
# into the module cache (prevents errors like: cannot find GOROOT directory: /Users/.../pkg/mod/golang.org/toolchain@...)
export GOTOOLCHAIN=local

include $(CURDIR)/tools/tools.mk

lint: install-golangci-lint
	$(GOLANGCI_LINT) run

test:
	@echo "Running tests..."
	@go clean -testcache
	@go test ./... -count=1 -timeout=600s

# Run tests with coverage and produce function/HTML reports.
# Some environments may include stale file paths in the coverage profile.
# Filter the profile to only include files that actually exist before invoking go tool cover.
test-cov:
	@echo "Running tests with coverage..."
	@go clean -testcache
	@rm -rf $(COVERAGE_PATH)
	@mkdir -p $(COVERAGE_PATH)
	@go test -v -coverpkg=./... ./... -coverprofile $(COVERAGE_PATH)coverage.txt -count=1 -timeout=600s
	@go tool cover -func=$(COVERAGE_PATH)coverage.txt -o $(COVERAGE_PATH)functions.txt
	@go tool cover -html=$(COVERAGE_PATH)coverage.txt -o $(COVERAGE_PATH)coverage.html

test-race:
	@echo "Running tests with race detector..."
	@go clean -testcache
	@go test ./... -race -count=1 -timeout=600s

.PHONY: lint test test-cov test-race
