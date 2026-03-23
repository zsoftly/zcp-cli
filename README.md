# ZCP CLI

The official command-line interface for the ZSoftly Cloud Platform

![CI](https://img.shields.io/badge/CI-passing-brightgreen)
![Go](https://img.shields.io/badge/Go-1.26.1-blue)

---

## Overview

ZCP CLI (`zcp`) is a command-line tool for managing resources on the ZSoftly Cloud Platform. It supports authentication, zone discovery, compute and storage offerings, template management, and resource availability checks. All commands support table, JSON, and YAML output formats, making the CLI suitable for both interactive use and scripting.

---

## Installation

### Quick Install — Linux / macOS

```bash
curl -fsSL https://raw.githubusercontent.com/zsoftly/zcp-cli/main/scripts/install.sh | bash
```

The script installs `zcp` to `/usr/local/bin`. You may be prompted for `sudo` access.

### PowerShell — Windows

```powershell
irm https://raw.githubusercontent.com/zsoftly/zcp-cli/main/scripts/install.ps1 | iex
```

### Manual Download

Download the appropriate binary for your platform from the [Releases](https://github.com/zsoftly/zcp-cli/releases) page, make it executable, and place it on your `PATH`:

```bash
# Example: Linux amd64
curl -Lo zcp https://github.com/zsoftly/zcp-cli/releases/latest/download/zcp-linux-amd64
chmod +x zcp
sudo mv zcp /usr/local/bin/zcp
```

### Build From Source

```bash
git clone https://github.com/zsoftly/zcp-cli.git
cd zcp-cli
make build
# Binary is written to bin/zcp
```

Requirements: Go 1.26.1+, GNU Make.

---

## Configuration

### Adding a Profile

Run the interactive setup to create your first profile:

```bash
zcp profile add
```

You will be prompted for:
- Profile name (default: `default`)
- API key
- Secret key
- API URL (leave blank to use the default: `https://cloud.zcp.zsoftly.ca`)

To add a named profile non-interactively:

```bash
zcp profile add staging --api-key YOUR_KEY --secret-key YOUR_SECRET
```

### Config File Location

| Platform     | Path                              |
|--------------|-----------------------------------|
| Linux/macOS  | `~/.config/zcp/config.yaml`       |
| Windows      | `%AppData%\zcp\config.yaml`       |

The file is created with `0600` permissions (owner read/write only) to protect credentials.

### Managing Profiles

```bash
# List all configured profiles
zcp profile list

# Switch to a different active profile
zcp profile use staging

# Show the currently active profile
zcp profile show
```

---

## Usage

### Version

```bash
zcp version
```

### Validate Authentication

```bash
zcp auth validate
```

Verifies that the active profile credentials are accepted by the API.

### Zones

```bash
# List all availability zones
zcp zone list

# Filter to a specific zone by UUID
zcp zone list --uuid <zone-uuid>
```

### Compute Offerings

```bash
# List available compute service offerings
zcp offering compute

# Filter by zone
zcp offering compute --zone-uuid <zone-uuid>
```

### Storage Offerings

```bash
# List available disk offerings
zcp offering storage

# Filter by zone
zcp offering storage --zone-uuid <zone-uuid>
```

### Network Offerings

```bash
# List available network offerings
zcp offering network
```

### VPC Offerings

```bash
# List available VPC offerings
zcp offering vpc
```

### Templates

```bash
# List available templates
zcp template list

# Filter by zone
zcp template list --zone-uuid <zone-uuid>
```

### Resource Availability

```bash
# Check available resources in all zones
zcp resource available

# Check a specific zone
zcp resource available --zone-uuid <zone-uuid>
```

---

## Phase 2 Commands

### Compute

```
zcp instance list --zone <uuid>
zcp instance get <uuid> --zone <uuid>
zcp instance create --zone <uuid> --name <name> --template <uuid> --compute-offering <uuid> --network <uuid>
zcp instance start <uuid>
zcp instance stop <uuid> [--force]
zcp instance reboot <uuid>
zcp instance delete <uuid> [--yes] [--expunge]
zcp instance resize <uuid> --offering <uuid>
zcp instance network-list <uuid>
zcp instance status <uuid>
```

### Storage

```
zcp volume list --zone <uuid>
zcp volume create --zone <uuid> --name <name> --storage-offering <uuid>
zcp volume attach <uuid> --instance <uuid>
zcp volume detach <uuid>
zcp volume delete <uuid> [--yes]
zcp snapshot list
zcp snapshot create --volume <uuid> --zone <uuid> --name <name>
zcp snapshot delete <uuid> [--yes]
zcp vm-snapshot list
zcp vm-snapshot create --zone <uuid> --name <name> --instance <uuid>
zcp vm-snapshot revert <uuid> [--yes]
zcp snapshot-policy list
zcp snapshot-policy create --volume <uuid> --interval <hourly|daily|weekly|monthly> --time <HH:MM> --timezone <tz> --max-snapshots <n>
```

### Networking

```
zcp network list --zone <uuid>
zcp network create --zone <uuid> --name <name> --offering <uuid>
zcp network delete <uuid> [--yes]
zcp ip list --zone <uuid>
zcp ip allocate --network <uuid>
zcp ip release <uuid> [--yes]
zcp ip static-nat enable <ip-uuid> --instance <uuid> --network <uuid>
zcp firewall list --zone <uuid>
zcp firewall create --ip <uuid> --protocol tcp --start-port 80 --end-port 80
zcp egress list --zone <uuid>
zcp egress create --network <uuid> --protocol tcp
zcp portforward list --zone <uuid>
zcp portforward create --ip <uuid> --protocol tcp --public-port 2222 --private-port 22 --instance <uuid>
zcp tag list
zcp tag create --zone <uuid> --resource <uuid> --type Instance --key env --value prod
```

---

## Output Formats

All listing commands support three output formats controlled by the `--output` flag.

**Table (default)**

```bash
zcp zone list
```

```
UUID                                  NAME      COUNTRY  ACTIVE
----                                  ----      -------  ------
3a7c1e2d-...                          Toronto   Canada   true
b91f4a8c-...                          Montreal  Canada   true
```

**JSON**

```bash
zcp zone list --output json
```

```json
[
  {
    "uuid": "3a7c1e2d-...",
    "name": "Toronto",
    "country_name": "Canada",
    "is_active": "true"
  }
]
```

**YAML**

```bash
zcp zone list --output yaml
```

```yaml
- uuid: 3a7c1e2d-...
  name: Toronto
  country_name: Canada
  is_active: "true"
```

---

## Global Flags

These flags are available on every command:

| Flag              | Default                             | Description                                |
|-------------------|-------------------------------------|--------------------------------------------|
| `--profile`       | active profile from config          | Profile name to use for this invocation    |
| `--output`        | `table`                             | Output format: `table`, `json`, `yaml`     |
| `--api-url`       | `https://cloud.zcp.zsoftly.ca`      | Override the API base URL                  |
| `--timeout`       | `30`                                | HTTP request timeout in seconds            |
| `--debug`         | `false`                             | Enable debug output (requests/responses)   |
| `--no-color`      | `false`                             | Disable ANSI color in table output         |

---

## Development

### Requirements

- Go 1.26.1 (toolchain, as specified in `go.mod`)
- GNU Make
- Git

### Build Targets

```bash
make build        # Build for the current platform → bin/zcp
make build-all    # Cross-compile for Linux, macOS, Windows (amd64 + arm64)
make test         # Run all tests with -v
make test-race    # Run all tests with the race detector
make fmt          # Format all Go source files with gofmt
make vet          # Run go vet
make tidy         # Tidy go.mod / go.sum
make lint         # Run staticcheck (must be installed separately)
make install      # Install zcp to /usr/local/bin
make clean        # Remove the bin/ directory
```

See `docs/development.md` for the full development guide.

---

## License

Copyright (c) ZSoftly. All rights reserved.
