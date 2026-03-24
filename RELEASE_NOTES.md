# zcp 0.0.2 Release Notes

## What's New

### Default zone per profile

Stop passing `--zone` on every command. Set it once:

```bash
zcp zone list                  # find your zone UUID
zcp zone use <uuid>            # save it to your active profile
```

Every command that previously required `--zone` now falls back to the profile default
automatically. You can still override per-command:

```bash
zcp instance list              # uses default zone
zcp instance list --zone <uuid>  # overrides for this call
```

To clear the default:

```bash
zcp zone use ""
```

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

**Full Changelog**: https://github.com/zsoftly/zcp-cli/compare/0.0.1...0.0.2
