# ZCP CLI

The official command-line interface for the ZSoftly Cloud Platform

[![CI](https://github.com/zsoftly/zcp-cli/actions/workflows/build.yml/badge.svg)](https://github.com/zsoftly/zcp-cli/actions/workflows/build.yml)
![Go](https://img.shields.io/badge/Go-1.26.4-blue)

---

## Overview

ZCP CLI (`zcp`) is a full-featured command-line tool for managing resources on the ZSoftly Cloud Platform. It covers the complete lifecycle of cloud infrastructure: compute instances, block storage, snapshots, networks, VPCs, firewalls, load balancers, VPN gateways, SSH keys, Kubernetes clusters, DNS, object storage, S3-compatible buckets, backups, autoscale policies, monitoring, projects, sub-users, roles and permissions, billing, and support. All commands support table, JSON, and YAML output, making the CLI equally suited for interactive use and automation pipelines.

---

## Installation

### Quick Install — Linux / macOS

```bash
curl -fsSL https://github.com/zsoftly/zcp-cli/releases/latest/download/install.sh | bash
```

The script installs `zcp` to `/usr/local/bin`. You may be prompted for `sudo` access.

### PowerShell — Windows

```powershell
irm https://github.com/zsoftly/zcp-cli/releases/latest/download/install.ps1 | iex
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

Requirements: Go 1.26.4+, GNU Make.

---

## Quick Start

```bash
# 1. Add your first profile (prompts for bearer token). Every account starts
#    with the project default-9 (like us-east-1 on AWS); if you use another
#    project, find its slug with 'zcp project list'.
zcp profile add default --region yul-1 --project default-9

# 2. Confirm your credentials work
zcp auth validate

# 3. Discover available regions
zcp region list

# 4. List your instances
zcp instance list
```

Your compute cloud provider is auto-detected and saved to your profile by `zcp auth validate` (and `zcp profile add`), so compute commands normally don't need it; `--cloud-provider` / `ZCP_CLOUD_PROVIDER` remain as overrides (e.g. for CI that skips `auth validate`). DNS and object storage select their own providers automatically.

Region and project stored on the profile are picked up automatically by scoped commands, including create. Two groups manage their own scope instead: `dns create` (fixed `default` region; pass `--project` explicitly) and `object-storage` (object-storage regions `os-yul`/`os-yow`). For CI or scripts without a profile, export the defaults instead:

```bash
export ZCP_REGION=yul-1
export ZCP_PROJECT=default-9
```

---

## Configuration

Profiles store your bearer token (and optional API URL) and are written to a `0600` config file:

| Platform    | Path                        |
| ----------- | --------------------------- |
| Linux/macOS | `~/.config/zcp/config.yaml` |
| Windows     | `%AppData%\zcp\config.yaml` |

```bash
zcp profile add default --region yul-1 --project default-9
zcp profile add staging --bearer-token TOKEN --region yul-1 --project default-9
zcp profile list                              # list profiles
zcp profile use staging                       # switch active profile
```

See **[docs/configuration.md](docs/configuration.md)** for full profile management, all environment variables, and precedence rules.

---

## Commands

Run `zcp <command> --help` for the full flag list of any command, and `zcp --help` for the complete command tree.

For copy-paste examples covering every resource — compute, storage, networking, VPCs, Kubernetes, object storage, DNS, billing, and more — see **[docs/commands.md](docs/commands.md)**.

> **Object storage is partly CLI-only.** Instance and basic bucket management go through the ZCP REST API and are mirrored in the Web UI, but the advanced S3 operations (versioning, policy, tagging, encryption, lifecycle, CORS, presigned URLs, copy/move, multipart cleanup, and all object uploads/downloads) talk directly to the Ceph RADOS Gateway over the S3 protocol and are available **only through this CLI** — the CMP has not yet exposed them via the REST API or Web UI. See [docs/commands.md](docs/commands.md#two-backends--and-what-is-cli-only).

Every listing command supports `--output table|json|yaml` (`-o` for short).

---

## Global Flags

These flags are available on every command:

| Flag             | Short | Default                    | Description                                        |
| ---------------- | ----- | -------------------------- | -------------------------------------------------- |
| `--profile`      |       | active profile from config | Profile name to use for this invocation            |
| `--output`       | `-o`  | `table`                    | Output format: `table`, `json`, `yaml`             |
| `--auto-approve` | `-y`  | `false`                    | Skip all confirmation prompts (useful for CI)      |
| `--api-url`      |       | from profile config        | Override the API base URL                          |
| `--timeout`      |       | `30`                       | HTTP request timeout in seconds                    |
| `--debug`        |       | `false`                    | Enable debug output (requests/responses to stderr) |
| `--no-color`     |       | `false`                    | Disable ANSI color in table output                 |
| `--pager`        |       | `false`                    | Pipe table output through a pager (`less`)         |

---

## Shell Completions

`zcp` ships with completion scripts for Bash, Zsh, Fish, and PowerShell:

```bash
source <(zcp completion bash)    # Bash (add to ~/.bashrc)
source <(zcp completion zsh)     # Zsh  (add to ~/.zshrc)
zcp completion fish | source     # Fish
zcp completion powershell | Out-String | Invoke-Expression   # PowerShell
```

See [docs/completions.md](docs/completions.md) for details.

---

## Development

```bash
make build        # Build for the current platform → bin/zcp
make build-all    # Cross-compile for Linux, macOS, Windows (amd64 + arm64)
make test         # Run all tests with -v
make test-race    # Run all tests with the race detector
make fmt          # Format all Go source files with gofmt
make vet          # Run go vet
make lint         # Run staticcheck (must be installed separately)
make install      # Install zcp to /usr/local/bin
```

Requirements: Go 1.26.4, GNU Make, Git. See **[docs/development.md](docs/development.md)** for the full development guide.

---

## License

Copyright (c) ZSoftly. All rights reserved.
