# zcp 0.0.4 Release Notes

## What's New

### VPC tier network commands

Create and update networks inside a VPC using the dedicated StackBill endpoint
(`/restapi/vpc/createVpcNetwork`):

```bash
zcp vpc create-network --vpc <uuid> --name my-tier --offering <uuid> \
  --gateway 10.1.1.1 --netmask 255.255.255.0 --acl <uuid>

zcp vpc update-network <network-uuid> --offering <uuid> --name new-name
```

This resolves the VPC tier creation issue — the previous `network create` endpoint
(`/restapi/network/createNetwork`) is for isolated networks only.

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

**Full Changelog**: https://github.com/zsoftly/zcp-cli/compare/0.0.3...0.0.4
