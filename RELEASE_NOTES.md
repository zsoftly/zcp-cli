# zcp 0.0.5 Release Notes

## What's New

### Delete verification

Delete commands now verify the resource is actually removed. Previously, the API could
return success (HTTP 204) while silently failing — the CLI would report "deleted" when
the resource was still there. Now affected commands check after delete and warn if the
resource still exists.

Applies to: `vpc delete`, `network delete`, `volume delete`, `security-group delete`

### Volume list deduplication

The API sometimes returns duplicate entries for the same volume. The CLI now deduplicates
by UUID before displaying results.

### Friendlier error messages

- `snapshot create` on a detached volume now says:
  `volume must be attached to a running instance before taking a snapshot`
  instead of the raw CloudStack error.
- `firewall list` on accounts with no IP addresses now returns an empty table
  instead of `API error 412: Invalid IpAddress Details`.

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

**Full Changelog**: https://github.com/zsoftly/zcp-cli/compare/0.0.4...0.0.5
