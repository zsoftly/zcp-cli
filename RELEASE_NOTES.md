# zcp 0.0.3 Release Notes

## What's New

### API field type fixes

Fixed JSON unmarshal errors across multiple resources where the live API returns
different types than the OpenAPI spec documents:

- `volume` — `createdTimeStamp` (string to int64)
- `host` — `cpuCores`, `vmCount` (int to string)
- `kubernetes` — `minMemory`, `minCpuNumber` (string to int)
- `vpc` / `network` — CIDR field name corrected (`getcIDR` to `cIDR`)

### New `host list` command

```bash
zcp host list
```

Lists all hypervisor hosts with CPU cores, VM count, and status.

### Network create supports VPC tiers

```bash
zcp network create --name my-tier --offering <uuid> --vpc <uuid> \
  --gateway 10.1.1.1 --netmask 255.255.255.0
```

New flags: `--vpc`, `--gateway`, `--netmask`, `--acl`

### Resource quota subcommand

```bash
zcp resource quota
```

### Integration test suite

Full lifecycle tests against the live API:

```bash
go test -tags integration -v -timeout 30m ./tests/integration/
```

### Other fixes

- `snapshot-policy list` now requires `--volume` (matches API spec)
- `network.IsPublic` moved from body to query parameter (matches API spec)
- VPC `description` and `publicLoadBalancerProvider` now required in create

---

## Installation

### Quick Install (Recommended)

**Windows:**

```powershell
irm https://github.com/zsoftly/zcp-cli/releases/latest/download/install.ps1 | iex
```

**macOS/Linux/WSL:**

```bash
curl -fsSL https://github.com/zsoftly/zcp-cli/releases/latest/download/install.sh | bash
```

### Manual Install

Download the binary for your platform from the assets below, make it executable, and move it to your `PATH`.

## Platforms

| OS      | Architecture | Binary                  |
| ------- | ------------ | ----------------------- |
| Linux   | amd64        | `zcp-linux-amd64`       |
| Linux   | arm64        | `zcp-linux-arm64`       |
| macOS   | amd64        | `zcp-darwin-amd64`      |
| macOS   | arm64        | `zcp-darwin-arm64`      |
| Windows | amd64        | `zcp-windows-amd64.exe` |
| Windows | arm64        | `zcp-windows-arm64.exe` |

**Full Changelog**: https://github.com/zsoftly/zcp-cli/compare/0.0.2...0.0.3
