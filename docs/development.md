# Development Guide

This guide explains how to set up a development environment for ZCP CLI, understand the repository structure, run tests, and contribute new commands.

---

## Prerequisites

| Tool | Minimum Version | Notes                                             |
| ---- | --------------- | ------------------------------------------------- |
| Go   | 1.26.1          | As declared in `go.mod` toolchain directive       |
| Make | Any             | GNU Make for build targets                        |
| Git  | Any             | Required for version embedding via `git describe` |

Install Go from [https://go.dev/dl/](https://go.dev/dl/). Verify your installation:

```bash
go version
# go version go1.26.1 linux/amd64
```

---

## Repository Structure

```
zcp-cli/
├── cmd/
│   └── zcp/
│       ├── main.go          # Entry point; wires root command
│       └── root/
│           └── root.go      # Root Cobra command; persistent flags; command registration
├── internal/
│   ├── config/              # Config file loading, saving, profile resolution
│   ├── httpclient/          # Shared HTTP client with auth header injection
│   ├── output/              # Table / JSON / YAML output rendering
│   ├── version/             # Version string (set via ldflags at build time)
│   ├── commands/            # Cobra command implementations (instance, region, dns, etc.)
│   └── api/
│       ├── apierrors/       # API error types and response parsing
│       ├── response/        # Generic response envelope types
│       ├── region/          # Region service
│       ├── instance/        # Virtual machine service
│       ├── plan/            # Service plan listing
│       ├── template/        # Template service
│       └── ...              # 30+ service packages (volume, network, dns, billing, etc.)
├── docs/                    # Markdown documentation
├── scripts/                 # Install scripts (install.sh, install.ps1)
├── Makefile                 # All build, test, and quality targets
├── go.mod                   # Module definition and dependencies
└── go.sum                   # Dependency checksums
```

### Key Packages

- `internal/config` — manages `~/.config/zcp/config.yaml`, profile resolution, and URL precedence logic.
- `internal/httpclient` — a single `Client` struct used by all API service packages. It injects the `Authorization: Bearer` header, sets `User-Agent`, and delegates error parsing to `apierrors`. Also provides `GetEnvelope`/`PostEnvelope`/`PutEnvelope` helpers for unwrapping the `{status, data}` response envelope.
- `internal/output` — the `Printer` type renders tabular data in table, JSON, or YAML format. All commands use this for consistent output.
- `internal/api/apierrors` — parses ZCP API error envelopes into typed `APIError` values.
- `internal/api/response` — generic response envelope types (`Envelope[T]`, `Single[T]`) for the paginated API format.
- `internal/commands` — one file per command group (e.g., `instance.go`, `region.go`, `dns.go`). Each file registers Cobra subcommands and implements `RunE` functions.

---

## Build Targets

All targets are defined in the `Makefile`. Run `make help` for a full listing.

```bash
make build        # Build bin/zcp for the current OS/arch
make dev          # Alias for build
make build-all    # Cross-compile: Linux/Darwin/Windows × amd64/arm64
make build-linux  # Linux amd64 + arm64
make build-darwin # macOS amd64 + arm64
make build-windows# Windows amd64 + arm64 (.exe)

make test         # Run all tests: go test -v ./...
make test-race    # Run tests with race detector: go test -race ./...

make fmt          # Format all Go files with gofmt
make vet          # Run go vet ./...
make tidy         # go mod tidy
make lint         # Run staticcheck (must be installed separately)

make install      # Copy bin/zcp to /usr/local/bin/zcp
make clean        # Remove the bin/ directory
make release-checksums  # Generate SHA256 checksums for all release binaries
```

The version string is embedded at build time:

```makefile
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X $(VERSION_PKG).Version=$(VERSION)"
```

When building outside a git repository, the version defaults to `dev`.

---

## Running Tests

```bash
# All tests, verbose
make test

# With race detector (recommended before submitting a PR)
make test-race

# A specific package
go test -v ./internal/config/...

# A single test function
go test -v -run TestSaveAndLoad ./internal/config/...
```

Tests use only the standard library's `net/http/httptest` for server mocking. No external test frameworks are used.

### Test Conventions

- Test files use the `_test` package suffix (e.g., `package config_test`) for black-box testing.
- Tests that write config files use `t.TempDir()` and `t.Setenv("XDG_CONFIG_HOME", ...)` to isolate state.
- HTTP handler tests use `httptest.NewServer` to spin up a local server and assert on request headers, query parameters, and response decoding.
- Waiter tests use short `WithPollInterval` and `WithWaitTimeout` values to keep test duration minimal.

---

## Adding a New Command

The following steps add a new top-level command group. The example adds `zcp network list`.

### 1. Create the API service package

Create `internal/api/network/network.go`:

```go
package network

import (
    "context"
    "github.com/zsoftly/zcp-cli/internal/httpclient"
)

type Network struct {
    UUID string `json:"uuid"`
    Name string `json:"name"`
}

type Service struct {
    client *httpclient.Client
}

func NewService(client *httpclient.Client) *Service {
    return &Service{client: client}
}

func (s *Service) List(ctx context.Context) ([]Network, error) {
    // implement API call
    return nil, nil
}
```

### 2. Write tests for the service

Create `internal/api/network/network_test.go` following the pattern used in `internal/api/region/region_test.go`:

- Spin up an `httptest.Server` that returns fixture JSON.
- Assert on the path, query parameters, and decoded result.
- Assert that HTTP error responses surface as errors.

### 3. Create the command file

Create `internal/commands/network.go`:

```go
package commands

import (
    "github.com/spf13/cobra"
    "github.com/zsoftly/zcp-cli/internal/api/network"
    "github.com/zsoftly/zcp-cli/internal/config"
    "github.com/zsoftly/zcp-cli/internal/httpclient"
    "github.com/zsoftly/zcp-cli/internal/output"
)

func NewNetworkCmd(flags *config.GlobalFlags) *cobra.Command {
    cmd := &cobra.Command{
        Use:   "network",
        Short: "Manage network resources",
    }
    cmd.AddCommand(newNetworkListCmd(flags))
    return cmd
}

func newNetworkListCmd(flags *config.GlobalFlags) *cobra.Command {
    return &cobra.Command{
        Use:   "list",
        Short: "List networks",
        RunE: func(cmd *cobra.Command, args []string) error {
            cfg, err := config.Load()
            if err != nil {
                return err
            }
            profile, err := config.ResolveProfile(cfg, flags.Profile)
            if err != nil {
                return err
            }
            client := httpclient.New(httpclient.Options{
                BaseURL:   config.ActiveAPIURL(profile, flags.APIURL),
                APIKey:    profile.APIKey,
                SecretKey: profile.SecretKey,
            })
            svc := network.NewService(client)
            networks, err := svc.List(cmd.Context())
            if err != nil {
                return err
            }
            p := output.NewPrinter(cmd.OutOrStdout(), output.ParseFormat(flags.Output), flags.NoColor)
            headers := []string{"UUID", "NAME"}
            rows := make([][]string, len(networks))
            for i, n := range networks {
                rows[i] = []string{n.UUID, n.Name}
            }
            return p.PrintTable(headers, rows)
        },
    }
}
```

### 4. Register the command in root

In `cmd/zcp/root/root.go`, add:

```go
rootCmd.AddCommand(commands.NewNetworkCmd(flags))
```

### 5. Verify

```bash
make build
./bin/zcp network list --help
make test
```

---

## Code Style

- Follow standard Go formatting enforced by `gofmt` (`make fmt`).
- All exported types, functions, and constants must have a doc comment.
- Return `error` from `RunE` rather than calling `os.Exit` directly; Cobra handles printing the error.
- Do not use `log.Fatal` or `fmt.Println` in command implementations — use the `output.Printer` or `cmd.ErrOrStderr()`.
- Use `context.Context` for all HTTP calls to support timeout and cancellation.
- Keep API service packages free of Cobra and output dependencies. Services receive a `*httpclient.Client` and return domain types and errors.

---

## Dependency Management

Dependencies are managed with Go modules.

```bash
# Add a new dependency (imported in code first)
go get github.com/some/package@v1.2.3

# Tidy up unused dependencies
make tidy

# Verify the dependency graph
go mod verify
```

Current direct dependencies:

| Package                             | Purpose                        |
| ----------------------------------- | ------------------------------ |
| `github.com/spf13/cobra`            | CLI framework                  |
| `github.com/olekukonko/tablewriter` | Terminal table rendering       |
| `gopkg.in/yaml.v3`                  | YAML marshalling/unmarshalling |

Prefer the standard library where possible. Introduce new dependencies only when they provide significant value over a standard library implementation.
