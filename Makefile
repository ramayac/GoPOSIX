# goposix Makefile
# -------------------------------------------------------------------
# All Go is built with CGO_ENABLED=0 for scratch-container compatibility.

BINARY     := goposix
CMD        := ./cmd/goposix
MODULE     := github.com/ramayac/goposix
VERSION    ?= $(shell git describe --tags --always 2>/dev/null || echo "dev")
LDFLAGS    := -ldflags "-s -w -X '$(MODULE)/pkg/common.Version=$(VERSION)' \
                              -X 'github.com/ramayac/goposix.Version=$(VERSION)'"
DOCKER_IMG     := goposix:$(VERSION)
DOCKER_IMG_CLI := goposix:cli-$(VERSION)

# Directories tested by the unit-test and coverage targets.
PKG_DIRS   := . \
              ./pkg/common/... \
              ./internal/dispatch/... \
              ./pkg/echo/... \
              ./pkg/awk/... \
              ./pkg/truefalse/... \
              ./pkg/whoami/... \
              ./pkg/hostname/... \
              ./pkg/hostid/... \
              ./pkg/factor/... \
              ./pkg/sha3sum/... \
              ./pkg/uname/... \
              ./pkg/pwd/... \
              ./pkg/printenv/... \
              ./pkg/paste/... \
              ./pkg/patch/... \
              ./pkg/pidof/... \
              ./pkg/nl/... \
              ./pkg/env/... \
              ./pkg/yes/... \
              ./pkg/ls/... \
              ./pkg/cat/... \
              ./pkg/mkdir/... \
              ./pkg/rmdir/... \
              ./pkg/rm/... \
              ./pkg/cp/... \
              ./pkg/comm/... \
              ./pkg/mv/... \
              ./pkg/touch/... \
              ./pkg/tree/... \
              ./pkg/ln/... \
              ./pkg/stat/... \
              ./pkg/readlink/... \
              ./pkg/realpath/... \
              ./pkg/rev/... \
              ./pkg/uptime/... \
              ./pkg/basename/... \
              ./pkg/cal/... \
              ./pkg/dirname/... \
              ./pkg/head/... \
              ./pkg/tail/... \
              ./pkg/wc/... \
              ./pkg/wget/... \
              ./pkg/which/... \
              ./pkg/tee/... \
              ./pkg/cut/... \
              ./pkg/tr/... \
              ./pkg/sort/... \
              ./pkg/tsort/... \
              ./pkg/seq/... \
              ./pkg/uniq/... \
              ./pkg/unexpand/... \
              ./pkg/grep/... \
              ./pkg/fold/... \
              ./pkg/sed/... \
              ./internal/daemon/... \
              ./pkg/daemon/... \
              ./pkg/client/... \
              ./pkg/sleep/... \
              ./pkg/date/... \
              ./pkg/dd/... \
              ./pkg/id/... \
              ./pkg/join/... \
              ./pkg/kill/... \
              ./pkg/df/... \
              ./pkg/du/... \
              ./pkg/find/... \
              ./pkg/ps/... \
              ./pkg/xargs/... \
              ./pkg/chmod/... \
              ./pkg/chown/... \
              ./pkg/chgrp/... \
              ./pkg/sha1sum/... \
              ./pkg/sha256sum/... \
              ./pkg/sha512sum/... \
              ./pkg/tar/... \
              ./internal/shell/... \
              ./pkg/shell/... \
              ./pkg/printf/... \
              ./pkg/expr/... \
              ./pkg/expand/... \
              ./pkg/testcmd/... \
              ./pkg/md5sum/... \
              ./pkg/gzip/... \
              ./pkg/diff/... \
              ./pkg/cksum/... \
              ./pkg/cmp/... \
              ./pkg/strings/... \
              ./pkg/sum/... \
              ./pkg/link/... \
              ./pkg/logger/... \
              ./pkg/logname/... \
              ./pkg/mkfifo/... \
              ./pkg/nice/... \
              ./pkg/nohup/... \
              ./pkg/od/... \
              ./pkg/split/... \
              ./pkg/tty/... \
              ./pkg/unlink/... \
              ./pkg/who/... \
              ./pkg/bunzip2/... \
              ./pkg/bzcat/... \
              ./pkg/unlzma/... \
              ./pkg/uncompress/... \
              ./pkg/unzip/... \
              ./pkg/uuencode/... \
              ./pkg/uudecode/... \
              ./pkg/taskset/... \
              ./pkg/start-stop-daemon/... \
              ./pkg/cryptpw/... \
              ./pkg/makedevs/... \
              ./pkg/ar/... \
              ./pkg/cpio/... \
              ./pkg/mount/... \
              ./pkg/mdev/... \
              ./test/posix-json/...


.DEFAULT_GOAL := help

# -------------------------------------------------------------------
# help — list all targets with descriptions
# -------------------------------------------------------------------
.PHONY: help
help:
	@echo ""
	@echo "  goposix — $(VERSION)"
	@echo ""
	@echo "  Usage: make <target>"
	@echo ""
	@echo "  Build"
	@echo "    build        Compile the goposix binary (CGO_ENABLED=0)"
	@echo "    build-race   Compile with -race detector (dev only)"
	@echo "    install      Install goposix to \$$GOPATH/bin"
	@echo "    image        Build the default (daemon) Docker image"
	@echo "    image-cli    Build the CLI-only (scratch) Docker image"
	@echo "    image-debug  Build the debug (Alpine+shell) Docker image"
	@echo ""
	@echo "  Test"
	@echo "    test         Run all unit tests"
	@echo "    test-v       Run all unit tests (verbose)"
	@echo "    test-race    Run unit tests with race detector (dev only, ~10x slower)"
	@echo "    cover        Run tests and open HTML coverage report"
	@echo "    cover-pct    Print per-package coverage percentages"
	@echo ""
	@echo "  Quality"
	@echo "    vet          Run go vet"
	@echo "    lint         Run staticcheck (installs if missing)"
	@echo "    fmt          Run gofmt -w on all Go files"
	@echo "    fmt-check    Check formatting without modifying files"
	@echo ""
	@echo "  Container"
	@echo "    docker        Build default daemon image ($(DOCKER_IMG))"
	@echo "    docker-cli    Build CLI-only scratch image (goposix:cli)"
	@echo "    docker-debug  Build Alpine debug image (goposix:debug)"
	@echo "    smoke-docker  Run smoke tests in CLI container"
	@echo ""
	@echo "  Smoke"
	@echo "    smoke        Build + run manual integration smoke tests (local)"
	@echo "    symlink-test Test symlink dispatch (ln -s goposix echo)"
	@echo ""
	@echo "  Performance"
	@echo "    bench-image   Build benchmark Docker image"
	@echo "    bench-quick   Quick benchmark (Cat A+H+F, SCALE=0.1 for CI)"
	@echo "    bench-all     Full benchmark suite (SCALE=1.0)"
	@echo "    bench-smoke   CI smoke (SCALE=0.1, ~30s)"
	@echo "    bench-pub     Publication quality (SCALE=5.0, ~40min)"
	@echo "    bench-stress  Stress test (SCALE=25.0, ~3h)"
	@echo "    bench-shell   Interactive shell in bench container"
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

# test-race runs all unit tests with the Go race detector enabled.
# The race detector instruments memory accesses and detects concurrent
# read/write conflicts on shared variables, maps, and slices. It catches
# data races that can cause silent corruption or fatal panics in production.
#
# Race detection is ~10x slower than normal tests and uses ~10x more memory.
# Use this target during development when touching concurrent code (daemon,
# shell, session manager, client SDK), before merging PRs that modify
# goroutine coordination, or when debugging flaky test failures.
#
# Not included in the default 'test' or 'ci' targets due to overhead.
.PHONY: test-race
test-race:
	go test -race $(PKG_DIRS)

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
	@echo "=== Per-package coverage (target: ≥80%) ==="
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
COVERAGE_THRESHOLD := 80
.PHONY: cover-gate
cover-gate:
	@echo "Checking coverage ≥ $(COVERAGE_THRESHOLD)%..."
	@tmp=$$(mktemp /tmp/goposix_ci_cover.XXXXXX.out); \
	CGO_ENABLED=0 go test -coverprofile=$$tmp $(PKG_DIRS) > /dev/null 2>&1 || true; \
	total=$$(go tool cover -func=$$tmp 2>/dev/null | grep '^total:' | awk '{print $$NF}' | tr -d '%'); \
	rm -f $$tmp; \
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
# Container targets
# -------------------------------------------------------------------

# Default image: persistent daemon (goposix:latest)
.PHONY: docker
docker:
	docker build \
	  --build-arg VERSION=$(VERSION) \
	  --target daemon \
	  -t $(DOCKER_IMG) \
	  -t goposix:latest \
	  -f docker/Dockerfile .

# Convenience aliases for clarity.
.PHONY: image
image: docker

# CLI-only scratch image (goposix:cli)
.PHONY: docker-cli
docker-cli:
	docker build \
	  --build-arg VERSION=$(VERSION) \
	  --target cli \
	  -t $(DOCKER_IMG_CLI) \
	  -t goposix:cli \
	  -f docker/Dockerfile .

.PHONY: image-cli
image-cli: docker-cli

.PHONY: docker-debug
docker-debug: ## Build debug alpine docker image
	docker build \
	  --target debug \
	  -t goposix:debug \
	  -f docker/Dockerfile .

.PHONY: image-debug
image-debug: docker-debug

# Go-Alpine Distro MVP (GoPOSIX-powered Alpine Userland)
.PHONY: docker-alpine
docker-alpine:
	docker build \
	  --target alpine-mvp \
	  -t go-alpine \
	  -f docker/Dockerfile .

.PHONY: image-alpine
image-alpine: docker-alpine

.PHONY: docker-shell
docker-shell: docker-debug ## Run an interactive shell in the docker image
	docker run -it --rm goposix:debug sh

.PHONY: docker-run
docker-run: docker-cli ## Run a command in the CLI scratch container (e.g., make docker-run CMD="ls -la")
	docker run --rm goposix:cli $(CMD)

.PHONY: docker-run-daemon
docker-run-daemon: docker ## Start the daemon container
	docker run -d --name goposix goposix:latest
	@echo "Daemon running. Socket: /var/run/goposix.sock"
	@echo "Stop: docker rm -f goposix"

# smoke-docker: run smoke checks inside the CLI container.
.PHONY: smoke-docker
smoke-docker: docker-cli
	@echo ""
	@echo "--- Docker smoke tests (goposix:cli) ---"
	docker run --rm goposix:cli true
	@echo "true: exit=0 OK"
	docker run --rm goposix:cli false; [ $$? -eq 1 ] && echo "false: exit=1 OK"
	docker run --rm goposix:cli echo smoke test passed
	docker run --rm goposix:cli echo --json smoke test
	docker run --rm goposix:cli whoami --json
	docker run --rm goposix:cli hostname --json
	docker run --rm goposix:cli uname --json
	docker run --rm goposix:cli pwd --json
	docker run --rm goposix:cli --help
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
	@echo "Creating symlink: echo -> goposix"
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
	-docker rmi $(DOCKER_IMG) goposix:cli goposix:debug 2>/dev/null || true
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

.PHONY: example-rpc
example-rpc: build
	@echo "--- Running RPC integration example ---"
	go run ./examples/rpc_client/main.go

.PHONY: bench
bench:
	@echo "--- Running benchmarks ---"
	go test -bench=. -benchmem ./test/benchmark/...

# =============================================================================
# Performance Benchmarking (GoPOSIX vs BusyBox) — see wiki/19_performance_benchmarking.md
# =============================================================================
SCALE ?= 1.0

.PHONY: bench-image
bench-image:
	docker build -t goposix:bench -f test/benchmark/Dockerfile.bench .

.PHONY: daemon-image
daemon-image:
	docker build --target daemon -t goposix:daemon -f docker/Dockerfile .

.PHONY: bench-daemon
bench-daemon: daemon-image
	@echo "Starting daemon container..."
	@docker rm -f goposix-bench-daemon 2>/dev/null || true
	docker run -d --name goposix-bench-daemon --privileged \
	  -v goposix-bench-data:/data \
	  goposix:daemon
	@sleep 2
	@echo "Daemon running. Socket: /home/goposix/goposix.sock (inside container)"
	@echo "Test: docker exec goposix-bench-daemon /bin/goposix echo hello"
	@echo "Bench: docker exec goposix-bench-daemon /bench/bench_client -op echo 1000"
	@echo "Stop:  docker rm -f goposix-bench-daemon"

.PHONY: bench-all
bench-all: bench-image
	docker run --rm --privileged \
	  -e BENCH_SCALE=$(SCALE) \
	  -v goposix-bench-data:/data \
	  goposix:bench --all

.PHONY: bench-cat
bench-cat: bench-image
	docker run --rm --privileged \
	  -e BENCH_SCALE=$(SCALE) \
	  -v goposix-bench-data:/data \
	  goposix:bench --cat $(CAT)

.PHONY: bench-quick
bench-quick: bench-image
	docker run --rm --privileged \
	  -e BENCH_SCALE=$(SCALE) \
	  -v goposix-bench-data:/data \
	  goposix:bench --quick

.PHONY: bench-smoke bench-pub bench-stress
bench-smoke: SCALE=0.1
bench-smoke: bench-all
bench-pub: SCALE=5.0
bench-pub: bench-all
bench-stress: SCALE=25.0
bench-stress: bench-all

.PHONY: bench-report
bench-report:
	@latest=$$(ls -t test/benchmark/results/ 2>/dev/null | grep -v latest | head -1); \
	if [ -n "$$latest" ]; then \
		test/benchmark/lib/report.sh test/benchmark/results/$$latest; \
	else \
		echo "No results found in test/benchmark/results/."; \
		echo "Results are stored in Docker volume 'goposix-bench-data'."; \
		echo "Use 'make bench-fetch' to copy the latest results locally."; \
	fi

.PHONY: bench-fetch
bench-fetch:
	@mkdir -p test/benchmark/results
	@cid=$$(docker create goposix:bench true 2>/dev/null); \
	docker cp $$cid:/data/results/. test/benchmark/results/ 2>/dev/null || true; \
	docker rm $$cid >/dev/null 2>&1 || true; \
	echo "Results fetched to test/benchmark/results/"

.PHONY: bench-shell
bench-shell: bench-image
	docker run --rm -it --privileged \
	  -e BENCH_SCALE=$(SCALE) \
	  -v goposix-bench-data:/data \
	  --entrypoint /bin/sh \
	  goposix:bench

.PHONY: ci
ci: vet test build docker smoke-docker cover-gate testsuite
	@echo "ci: full pipeline complete (coverage ≥ $(COVERAGE_THRESHOLD)%)"
