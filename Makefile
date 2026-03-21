.PHONY: build clean test frontend backend dev

# Build the single static binary with embedded frontend
build: frontend backend

# Build frontend
frontend:
	cd frontend && npm run build
	rm -rf cmd/homeapi/frontend_build
	cp -r frontend/build cmd/homeapi/frontend_build

# Build Go binary
backend: frontend
	go build -o homeapi ./cmd/homeapi

# Development: run Go server only (frontend via npm start separately)
dev:
	go run ./cmd/homeapi

# Run all tests
test:
	go test ./internal/... -v -count=1
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
	rm -rf cmd/homeapi/frontend_build
	rm -rf frontend/build
