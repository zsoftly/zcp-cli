# ZCP CLI

The official command-line interface for the ZSoftly Cloud Platform

[![CI](https://github.com/zsoftly/zcp-cli/actions/workflows/build.yml/badge.svg)](https://github.com/zsoftly/zcp-cli/actions/workflows/build.yml)
![Go](https://img.shields.io/badge/Go-1.26.1-blue)

---

## Overview

ZCP CLI (`zcp`) is a full-featured command-line tool for managing resources on the ZSoftly Cloud Platform. It covers the complete lifecycle of cloud infrastructure: compute instances, block storage, snapshots, networks, VPCs, firewalls, load balancers, VPN gateways, SSH keys, Kubernetes clusters, and billing. All commands support table, JSON, and YAML output, making the CLI equally suited for interactive use and automation pipelines.

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

## Quick Start

```bash
# 1. Install (see above)

# 2. Add your first profile — prompts for API key and secret key
zcp profile add default

# 3. Confirm your credentials work
zcp auth validate

# 4. Discover available zones
zcp zone list

# 5. List your instances
zcp instance list --zone <zone-uuid>
```

---

## Configuration

### Adding a Profile

Run the interactive setup to create your first profile:

```bash
zcp profile add default
```

You will be prompted for:
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

# Show details of the active profile (credentials are masked)
zcp profile show

# Show details of a specific profile
zcp profile show staging

# Switch to a different active profile
zcp profile use staging

# Update credentials or API URL on an existing profile
zcp profile update prod --api-key <new-key>
zcp profile update prod --secret-key <new-secret>
zcp profile update prod --api-url-override https://custom.api.url

# Rename a profile
zcp profile rename staging prod

# Delete a profile
zcp profile delete old-profile
```

---

## Commands Reference

Use `zcp <command> --help` for the full flag list of any command.

### Discovery

```bash
# Zones
zcp zone list
zcp zone list --uuid <zone-uuid>

# Compute, storage, network, and VPC offerings
zcp offering compute
zcp offering compute --zone <zone-uuid>
zcp offering storage
zcp offering storage --zone <zone-uuid>
zcp offering network
zcp offering vpc

# VM templates
zcp template list
zcp template list --zone <zone-uuid>

# Resource availability
zcp resource available
zcp resource available --zone <zone-uuid>
```

### Compute

```bash
# List and inspect
zcp instance list --zone <zone-uuid>
zcp instance get <uuid> --zone <zone-uuid>
zcp instance status <uuid>

# Create — use --wait to block until the instance is Running
zcp instance create \
  --zone <zone-uuid> \
  --name my-vm \
  --template <template-uuid> \
  --compute-offering <offering-uuid> \
  --network <network-uuid>

zcp instance create ... --ssh-key mykey --wait

# Lifecycle
zcp instance start <uuid>
zcp instance start <uuid> --wait
zcp instance stop <uuid>
zcp instance stop <uuid> --force --wait
zcp instance reboot <uuid>
zcp instance delete <uuid> [--yes] [--expunge]

# Resize — change compute offering (instance must be stopped)
zcp instance resize <uuid> --offering <offering-uuid>
zcp instance resize <uuid> --offering <offering-uuid> --cpu 4 --memory 8192

# Rename display name
zcp instance rename <uuid> --display-name "My Web Server"

# Recover from error state
zcp instance recover <uuid>

# List attached networks and passwords
zcp instance network-list <uuid>
zcp instance password-list --zone <zone-uuid>
zcp instance password-list --zone <zone-uuid> --instance <uuid>

# Open an SSH session directly from the CLI
zcp instance ssh <uuid>
zcp instance ssh <uuid> --user ubuntu
zcp instance ssh <uuid> --user root --identity-file ~/.ssh/my-key.pem --port 2222
```

The `--wait` flag on `create`, `start`, and `stop` polls the API until the instance reaches the target state, printing progress to stderr.

### Storage

```bash
# Volumes
zcp volume list --zone <zone-uuid>
zcp volume create --zone <zone-uuid> --name my-disk --storage-offering <uuid>
zcp volume attach <uuid> --instance <uuid>
zcp volume detach <uuid>
zcp volume delete <uuid> [--yes]

# Snapshots
zcp snapshot list
zcp snapshot create --volume <uuid> --zone <zone-uuid> --name my-snapshot
zcp snapshot delete <uuid> [--yes]

# VM snapshots (whole-instance checkpoint)
zcp vm-snapshot list
zcp vm-snapshot create --zone <zone-uuid> --name my-checkpoint --instance <uuid>
zcp vm-snapshot revert <uuid> [--yes]

# Snapshot policies (automated scheduling)
zcp snapshot-policy list
zcp snapshot-policy create \
  --volume <uuid> \
  --interval daily \
  --time 02:00 \
  --timezone America/Toronto \
  --max-snapshots 7
```

### Networking

```bash
# Networks
zcp network list --zone <zone-uuid>
zcp network create --zone <zone-uuid> --name my-net --offering <uuid>
zcp network delete <uuid> [--yes]

# Public IP addresses
zcp ip list --zone <zone-uuid>
zcp ip allocate --network <uuid>
zcp ip release <uuid> [--yes]
zcp ip static-nat enable <ip-uuid> --instance <uuid> --network <uuid>

# Firewall rules (ingress)
zcp firewall list --zone <zone-uuid>
zcp firewall create --ip <uuid> --protocol tcp --start-port 80 --end-port 80

# Egress rules
zcp egress list --zone <zone-uuid>
zcp egress create --network <uuid> --protocol tcp

# Port forwarding
zcp portforward list --zone <zone-uuid>
zcp portforward create \
  --ip <uuid> \
  --protocol tcp \
  --public-port 2222 \
  --private-port 22 \
  --instance <uuid>

# Resource tags
zcp tag list
zcp tag create --zone <zone-uuid> --resource <uuid> --type Instance --key env --value prod
```

### Advanced Networking

```bash
# VPCs
zcp vpc list
zcp vpc create --zone <zone-uuid> --name my-vpc --offering <vpc-offering-uuid> --cidr 10.0.0.0/16
zcp vpc delete <uuid> [--yes]

# Network ACLs
zcp acl list
zcp acl create --vpc <uuid> --name my-acl
zcp acl delete <uuid> [--yes]

# Public load balancers
zcp loadbalancer list --zone <zone-uuid>
zcp loadbalancer create --ip <uuid> --name my-lb --algorithm roundrobin
zcp loadbalancer delete <uuid> [--yes]

# Internal load balancers (VPC-scoped)
zcp internal-lb list --zone <zone-uuid>
zcp internal-lb create --network <uuid> --name my-internal-lb
zcp internal-lb delete <uuid> [--yes]

# VPN gateways and connections
zcp vpn list --zone <zone-uuid>
zcp vpn create --vpc <uuid> --name my-vpn
zcp vpn delete <uuid> [--yes]
```

### Security and Access

```bash
# SSH keys
zcp ssh-key list
zcp ssh-key create --name mykey --public-key "$(cat ~/.ssh/id_rsa.pub)"
zcp ssh-key delete <uuid> [--yes]

# Security groups
zcp security-group list
zcp security-group create --name my-sg --description "Web tier"
zcp security-group delete <uuid> [--yes]
```

### Kubernetes

```bash
# 'k8s' is accepted as an alias for 'kubernetes'
zcp kubernetes list --zone <zone-uuid>
zcp kubernetes get <uuid>
zcp kubernetes create --zone <zone-uuid> --name my-cluster --offering <uuid>
zcp kubernetes delete <uuid> [--yes]
zcp kubernetes kubeconfig <uuid>   # Download kubeconfig for kubectl access
```

### Billing and Admin

```bash
# Usage records
zcp usage list
zcp usage list --zone <zone-uuid>
zcp usage list --output csv

# Cost summary
zcp cost summary
zcp cost summary --zone <zone-uuid>

# Admin operations (requires elevated permissions)
zcp admin list-accounts
zcp admin get-account <uuid>
```

### Auth

```bash
# Validate that the active profile credentials are accepted by the API
zcp auth validate
```

---

## Output Formats

All listing commands support three output formats controlled by the `--output` (or `-o`) flag.

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

| Flag          | Default                          | Description                                        |
|---------------|----------------------------------|----------------------------------------------------|
| `--profile`   | active profile from config       | Profile name to use for this invocation            |
| `--output`    | `table`                          | Output format: `table`, `json`, `yaml`             |
| `--api-url`   | `https://cloud.zcp.zsoftly.ca`   | Override the API base URL                          |
| `--timeout`   | `30`                             | HTTP request timeout in seconds                    |
| `--debug`     | `false`                          | Enable debug output (requests/responses to stderr) |
| `--no-color`  | `false`                          | Disable ANSI color in table output                 |
| `--pager`     | `false`                          | Pipe table output through a pager (`less`)         |

The `-o` shorthand is accepted for `--output`.

---

## Shell Completions

`zcp` ships with completion scripts for Bash, Zsh, Fish, and PowerShell.

```bash
# Bash (add to ~/.bashrc)
source <(zcp completion bash)

# Zsh (add to ~/.zshrc)
source <(zcp completion zsh)

# Fish
zcp completion fish | source

# PowerShell
zcp completion powershell | Out-String | Invoke-Expression
```

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
