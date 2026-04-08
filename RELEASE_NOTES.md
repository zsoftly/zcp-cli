# zcp 0.0.7 Release Notes

## What's New

### 8 new commands

| Command                     | Description                                                                  |
| --------------------------- | ---------------------------------------------------------------------------- |
| `zcp region list`           | List available regions (replaces `zone list`)                                |
| `zcp profile-info`          | User profile, company details, 2FA, time settings, API access, activity logs |
| `zcp vm-backup list/create` | VM backup operations                                                         |
| `zcp cloud-provider list`   | List available cloud providers                                               |
| `zcp server list`           | List available servers                                                       |
| `zcp currency list`         | List available currencies                                                    |
| `zcp billing-cycle list`    | List available billing cycles                                                |
| `zcp storage-category list` | List available storage categories                                            |

### Dead code removed

11 commands and 13 API packages that still pointed at old `/restapi/` endpoints have been removed. These commands were broken since v0.0.6 and would return 403 errors:

`zone`, `offering`, `resource`, `host`, `cost`, `usage`, `internal-lb`, `snapshot-policy`, `security-group`, `tag`, `admin`

Use the STKCNSL replacements instead:

| Old command               | Replacement                 |
| ------------------------- | --------------------------- |
| `zcp zone list`           | `zcp region list`           |
| `zcp offering compute`    | `zcp plan vm`               |
| `zcp offering storage`    | `zcp plan storage`          |
| `zcp cost summary`        | `zcp billing costs`         |
| `zcp usage list`          | `zcp billing monthly-usage` |
| `zcp tag create`          | `zcp instance tag-create`   |
| `zcp admin list-accounts` | Not available via API       |

### Auth validate fixed

`zcp auth validate` now correctly hits the STKCNSL region API instead of the dead zone API.

---

## 42 total commands

The CLI now has 42 commands, all backed by the STKCNSL API with zero legacy code remaining.

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

**Full Changelog**: https://github.com/zsoftly/zcp-cli/compare/0.0.6...0.0.7
