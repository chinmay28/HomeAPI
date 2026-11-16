.PHONY: build clean test frontend backend dev test-unit test-integration test-e2e release version icons icons-check

# Build the single static binary with embedded frontend
build: frontend backend

# Install frontend deps if missing or stale relative to the lockfile.
frontend/node_modules: frontend/package.json frontend/package-lock.json
	cd frontend && npm ci
	@touch frontend/node_modules

# Build frontend
frontend: frontend/node_modules
	cd frontend && npm run build
	rm -rf cmd/homeapi/frontend_build
	cp -r frontend/build cmd/homeapi/frontend_build

# Build Go binary. The version is the commit's month and the repo's commit
# count, which a compiled binary can't read — stamp them in (see
# internal/version). CGO_ENABLED=0: the SQLite driver is pure Go, and a static
# binary is what lets the same build run on any Linux, and on a phone.
LDFLAGS = $(shell node scripts/version.mjs --ldflags)

backend: frontend
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o homeapi ./cmd/homeapi

# Static linux/<arch> release binaries with the frontend embedded, as the
# release workflow publishes them: dist/homeapi-linux-amd64, -arm64.
release: frontend
	mkdir -p dist
	for arch in amd64 arm64; do \
		CGO_ENABLED=0 GOOS=linux GOARCH=$$arch go build -trimpath -ldflags "$(LDFLAGS)" -o dist/homeapi-linux-$$arch ./cmd/homeapi || exit 1; \
	done

# The Android app's icons, derived from frontend/public/icon.svg (see
# cmd/homeapi-icons). icons-check fails when a committed copy has drifted.
icons:
	go run ./cmd/homeapi-icons

icons-check:
	go run ./cmd/homeapi-icons -check

# Print the version this tree builds as.
version:
	@node scripts/version.mjs

# Development: run Go server only (frontend via npm start separately)
dev:
	go run ./cmd/homeapi

# Run all tests
test:
	go test ./internal/... -v -count=1
	go test ./cmd/homeapi-icons -count=1
	go test ./tests/integration/... -v -count=1
	go test ./tests/e2e/... -v -count=1

# Unit tests only
test-unit:
	go test ./internal/... -v -count=1

# Integration tests only
test-integration:
	go test ./tests/integration/... -v -count=1

# E2E tests only
test-e2e:
	go test ./tests/e2e/... -v -count=1

# Clean build artifacts
clean:
	rm -f homeapi
	rm -rf dist
	rm -rf cmd/homeapi/frontend_build
	rm -rf frontend/build
