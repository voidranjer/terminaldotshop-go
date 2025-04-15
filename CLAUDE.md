# Terminal Go Client - Development Guide

## Build & Test Commands

- `make build`: Build the application binary
- `make run`: Build and run the application
- `make run/live`: Run with live reloading
- `make test`: Run all tests with race detection
- `make test/cover`: Run tests with coverage
- `go test -v ./pkg/api`: Run tests in specific package
- `make tidy`: Format code and tidy go.mod
- `make audit`: Run quality checks (vet, staticcheck, govulncheck)
- `go run ./cmd/cli/main.go`: Run CLI directly
- `bun dev`: Run SSH server with SST

## Code Style Guidelines

- **Imports**: Standard library first, followed by external dependencies, then local packages
- **Types**: Define types at the top of files, use custom structs for domain models
- **Naming**: PascalCase for exported identifiers, camelCase for private
- **Error Handling**: Explicit error returns with context, central error display mechanism
- **Modules**: Organized by logical domain (api, tui, resource)
- **Testing**: `_test.go` files with context-based testing
- **Documentation**: Use Go doc comments for public functions and types
- **UI Components**: Composition-based Bubble Tea components with Model-View pattern
- **Formatting**: Standard Go formatting with `go fmt`