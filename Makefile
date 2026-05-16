# korego Makefile
# -------------------------------------------------------------------
# All Go is built with CGO_ENABLED=0 for scratch-container compatibility.

BINARY     := korego
CMD        := ./cmd/korego
MODULE     := github.com/ramayac/korego
VERSION    ?= $(shell git describe --tags --always 2>/dev/null || echo "dev")
LDFLAGS    := -ldflags "-s -w -X '$(MODULE)/pkg/common.Version=$(VERSION)' \
                              -X 'github.com/ramayac/korego.Version=$(VERSION)'"
DOCKER_IMG := korego:$(VERSION)

# Directories tested by the unit-test and coverage targets.
PKG_DIRS   := . \
              ./pkg/common/... \
              ./internal/dispatch/... \
              ./pkg/echo/... \
              ./pkg/truefalse/... \
              ./pkg/whoami/... \
              ./pkg/hostname/... \
              ./pkg/uname/... \
              ./pkg/pwd/... \
              ./pkg/printenv/... \
              ./pkg/env/... \
              ./pkg/yes/... \
              ./pkg/ls/... \
              ./pkg/cat/... \
              ./pkg/mkdir/... \
              ./pkg/rmdir/... \
              ./pkg/rm/... \
              ./pkg/cp/... \
              ./pkg/mv/... \
              ./pkg/touch/... \
              ./pkg/ln/... \
              ./pkg/stat/... \
              ./pkg/readlink/... \
              ./pkg/basename/... \
              ./pkg/dirname/... \
              ./pkg/head/... \
              ./pkg/tail/... \
              ./pkg/wc/... \
              ./pkg/tee/... \
              ./pkg/cut/... \
              ./pkg/tr/... \
              ./pkg/sort/... \
              ./pkg/uniq/... \
              ./pkg/grep/... \
              ./pkg/sed/... \
              ./internal/daemon/... \
              ./pkg/daemon/... \
              ./pkg/client/... \
              ./pkg/sleep/... \
              ./pkg/date/... \
              ./pkg/id/... \
              ./pkg/kill/... \
              ./pkg/df/... \
              ./pkg/du/... \
              ./pkg/find/... \
              ./pkg/ps/... \
              ./pkg/xargs/... \
              ./pkg/chmod/... \
              ./pkg/chown/... \
              ./pkg/chgrp/... \
              ./pkg/sha256sum/... \
              ./pkg/tar/... \
              ./internal/shell/... \
              ./pkg/printf/... \
              ./pkg/expr/... \
              ./pkg/testcmd/... \
              ./pkg/md5sum/... \
              ./pkg/gzip/... \
              ./pkg/diff/... \
              ./test/posix-json/...

.DEFAULT_GOAL := help

# -------------------------------------------------------------------
# help — list all targets with descriptions
# -------------------------------------------------------------------
.PHONY: help
help:
	@echo ""
	@echo "  korego — $(VERSION)"
	@echo ""
	@echo "  Usage: make <target>"
	@echo ""
	@echo "  Build"
	@echo "    build        Compile the korego binary (CGO_ENABLED=0)"
	@echo "    build-race   Compile with -race detector (dev only)"
	@echo "    install      Install korego to \$$GOPATH/bin"
	@echo ""
	@echo "  Test"
	@echo "    test         Run all unit tests"
	@echo "    test-v       Run all unit tests (verbose)"
	@echo "    cover        Run tests and open HTML coverage report"
	@echo "    cover-pct    Print per-package coverage percentages"
	@echo ""
	@echo "  Quality"
	@echo "    vet          Run go vet"
	@echo "    lint         Run staticcheck (installs if missing)"
	@echo "    fmt          Run gofmt -w on all Go files"
	@echo "    fmt-check    Check formatting without modifying files"
	@echo ""
	@echo "  Docker"
	@echo "    docker        Build production scratch image ($(DOCKER_IMG))"
	@echo "    docker-debug  Build Alpine debug image (korego:debug)"
	@echo "    smoke-docker  Run smoke tests inside the production container"
	@echo ""
	@echo "  Smoke"
	@echo "    smoke        Build + run manual integration smoke tests (local)"
	@echo "    symlink-test Test symlink dispatch (ln -s korego echo)"
	@echo ""
	@echo "  Housekeeping"
	@echo "    clean        Remove build artifacts and Docker image"
	@echo "    tidy         go mod tidy"
	@echo "    all          vet + test + build"
	@echo ""

# -------------------------------------------------------------------
# Build targets
# -------------------------------------------------------------------
.PHONY: build
build:
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BINARY) $(CMD)

.PHONY: build-race
build-race:
	go build -race $(LDFLAGS) -o $(BINARY)-race $(CMD)

.PHONY: install
install:
	CGO_ENABLED=0 go install $(LDFLAGS) $(CMD)

# -------------------------------------------------------------------
# Test targets
# -------------------------------------------------------------------
.PHONY: test
test:
	CGO_ENABLED=0 go test $(PKG_DIRS)

.PHONY: test-v
test-v:
	CGO_ENABLED=0 go test -v $(PKG_DIRS)

.PHONY: cover
cover:
	CGO_ENABLED=0 go test -coverprofile=coverage.out $(PKG_DIRS)
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report written to coverage.html"
	@command -v xdg-open >/dev/null 2>&1 && xdg-open coverage.html || true

.PHONY: cover-pct
cover-pct:
	CGO_ENABLED=0 go test -cover $(PKG_DIRS)

.PHONY: cover-pkg
cover-pkg:
	@echo "=== Per-package coverage (target: ≥70%) ==="
	@CGO_ENABLED=0 go test -cover $(PKG_DIRS) 2>&1 | grep -E 'coverage:|FAIL' | while read line; do \
		pct=$$(echo "$$line" | grep -oP '\d+\.\d+%' | head -1 | tr -d '%'); \
		if [ -n "$$pct" ]; then \
			status="OK"; \
			if [ "$$(echo "$$pct < 5.0" | bc -l 2>/dev/null || echo 0)" = "1" ]; then status="CRITICAL"; fi; \
			printf "  %-6s %s\n" "$$pct%" "$$line"; \
		else \
			echo "  ----   $$line"; \
		fi; \
	done

# CI coverage gate: fails if overall coverage < threshold.
COVERAGE_THRESHOLD := 70
.PHONY: cover-gate
cover-gate:
	@echo "Checking coverage ≥ $(COVERAGE_THRESHOLD)%..."
	@CGO_ENABLED=0 go test -coverprofile=/tmp/korego_ci_cover.out $(PKG_DIRS) > /dev/null 2>&1 || true
	@total=$$(go tool cover -func=/tmp/korego_ci_cover.out 2>/dev/null | grep '^total:' | awk '{print $$NF}' | tr -d '%'); \
	if [ -z "$$total" ]; then echo "FAIL: could not parse coverage"; exit 1; fi; \
	if [ "$$(echo "$$total < $(COVERAGE_THRESHOLD)" | bc -l)" = "1" ]; then \
		echo "FAIL: coverage $$total% < $(COVERAGE_THRESHOLD)% threshold"; exit 1; \
	else \
		echo "PASS: coverage $$total% ≥ $(COVERAGE_THRESHOLD)%"; \
	fi

# -------------------------------------------------------------------
# Quality targets
# -------------------------------------------------------------------
.PHONY: vet
vet:
	CGO_ENABLED=0 go vet ./...

.PHONY: lint
lint:
	@command -v staticcheck >/dev/null 2>&1 || \
		(echo "Installing staticcheck..." && go install honnef.co/go/tools/cmd/staticcheck@latest)
	staticcheck ./...

.PHONY: fmt
fmt:
	gofmt -w $$(find . -name '*.go' -not -path './.git/*')

.PHONY: fmt-check
fmt-check:
	@diff=$$(gofmt -l $$(find . -name '*.go' -not -path './.git/*')); \
	if [ -n "$$diff" ]; then \
		echo "The following files are not gofmt-compliant:"; \
		echo "$$diff"; \
		exit 1; \
	fi
	@echo "All files are gofmt-compliant."

# -------------------------------------------------------------------
# Docker targets
# -------------------------------------------------------------------
.PHONY: docker
docker:
	docker build \
	  --build-arg VERSION=$(VERSION) \
	  -t $(DOCKER_IMG) \
	  -f docker/Dockerfile .

.PHONY: docker-debug
docker-debug: ## Build debug alpine docker image
	docker build -t korego:debug -f docker/Dockerfile.debug .

.PHONY: docker-shell
docker-shell: docker-debug ## Run an interactive shell in the docker image
	docker run -it --rm korego:debug sh

.PHONY: docker-run
docker-run: docker ## Run a command in the production scratch container (e.g., make docker-run CMD="ls -la")
	docker run --rm $(DOCKER_IMG) $(CMD)

# smoke-docker: run smoke checks inside the production scratch container.
.PHONY: smoke-docker
smoke-docker: docker
	@echo ""
	@echo "--- Docker smoke tests ($(DOCKER_IMG)) ---"
	docker run --rm $(DOCKER_IMG) true
	@echo "true: exit=0 OK"
	docker run --rm $(DOCKER_IMG) false; [ $$? -eq 1 ] && echo "false: exit=1 OK"
	docker run --rm $(DOCKER_IMG) echo smoke test passed
	docker run --rm $(DOCKER_IMG) echo --json smoke test
	docker run --rm $(DOCKER_IMG) whoami --json
	docker run --rm $(DOCKER_IMG) hostname --json
	docker run --rm $(DOCKER_IMG) uname --json
	docker run --rm $(DOCKER_IMG) pwd --json
	docker run --rm $(DOCKER_IMG) --help
	@echo ""
	@echo "=== ALL DOCKER SMOKE TESTS PASSED ==="

# -------------------------------------------------------------------
# Smoke / integration tests (local binary)
# -------------------------------------------------------------------
.PHONY: smoke
smoke: build
	@echo ""
	@echo "--- true / false ---"
	./$(BINARY) true;  echo "true  exit=$$?"
	./$(BINARY) false; echo "false exit=$$?"
	@echo ""
	@echo "--- echo ---"
	./$(BINARY) echo hello world
	./$(BINARY) echo --json hello world
	@echo ""
	@echo "--- uname ---"
	./$(BINARY) uname
	./$(BINARY) uname --json
	@echo ""
	@echo "--- whoami ---"
	./$(BINARY) whoami
	./$(BINARY) whoami --json
	@echo ""
	@echo "--- pwd ---"
	./$(BINARY) pwd
	./$(BINARY) pwd --json
	@echo ""
	@echo "--- hostname ---"
	./$(BINARY) hostname
	./$(BINARY) hostname --json
	@echo ""
	@echo "--- printenv HOME ---"
	./$(BINARY) printenv HOME
	./$(BINARY) printenv --json HOME
	@echo ""
	@echo "--- env -i FOO=bar ---"
	./$(BINARY) env -i FOO=bar
	./$(BINARY) env --json -i FOO=bar
	@echo ""
	@echo "--- unknown command (expect 127) ---"
	./$(BINARY) nonexist; echo "exit=$$?"
	@echo ""
	@echo "--- --help ---"
	./$(BINARY) --help
	@echo ""
	@echo "--- --version ---"
	./$(BINARY) --version
	@echo ""
	@echo "smoke: all checks done"

.PHONY: symlink-test
symlink-test: build
	@echo "Creating symlink: echo -> korego"
	ln -sf ./$(BINARY) ./echo
	@echo "Running ./echo via symlink..."
	./echo symlink dispatch works
	./echo --json symlink json
	rm -f ./echo
	@echo "symlink-test: PASS"

# -------------------------------------------------------------------
# Housekeeping
# -------------------------------------------------------------------
.PHONY: clean
clean:
	rm -f $(BINARY) $(BINARY)-race coverage.out coverage.html
	-docker rmi $(DOCKER_IMG) korego:debug 2>/dev/null || true
	@echo "clean: done"

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: all
all: vet test build
	@echo "all: vet + test + build complete"

.PHONY: testsuite
testsuite: build
	@echo "--- BusyBox Test Suite ---"
	cd test/busybox_testsuite && ./runtest

.PHONY: validate-schemas
validate-schemas: build
	@echo "--- Validate JSON output against schemas ---"
	bash test/validate_schemas.sh

.PHONY: example-agent
example-agent: build
	@echo "--- Running agent integration example ---"
	go run ./examples/agent/main.go

.PHONY: bench
bench:
	@echo "--- Running benchmarks ---"
	go test -bench=. -benchmem ./test/benchmark/...

.PHONY: ci
ci: vet test build docker smoke-docker cover-gate
	@echo "ci: full pipeline complete (coverage ≥ $(COVERAGE_THRESHOLD)%)"
