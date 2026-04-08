# zcp 0.0.6 Release Notes

## What's New

### API backend migration

The CLI now communicates with the STKCNSL API backend, replacing the previous STKBILL backend. This is a breaking change for configuration files — existing profiles using `apikey`/`secretkey` must be recreated with `bearer_token`.

### Bearer token authentication

Authentication now uses a single bearer token instead of separate API key and secret key. Update your profiles:

```bash
zcp profile add default --bearer-token YOUR_TOKEN
```

Config files now use `bearer_token` instead of `apikey`/`secretkey`.

### 15 new command groups

| Command              | Description                                            |
| -------------------- | ------------------------------------------------------ |
| `zcp dns`            | DNS domain and record management                       |
| `zcp project`        | Project management with users and dashboards           |
| `zcp monitoring`     | Global and per-VM resource monitoring                  |
| `zcp billing`        | Costs, invoices, subscriptions, coupons, budget alerts |
| `zcp support`        | Support tickets, replies, feedback, FAQs               |
| `zcp autoscale`      | VM autoscale groups with policies and conditions       |
| `zcp dashboard`      | Account service counts overview                        |
| `zcp plan`           | Service plans for all resource types                   |
| `zcp store`          | Store items and checkout                               |
| `zcp marketplace`    | Marketplace app listing                                |
| `zcp product`        | Product categories and listing                         |
| `zcp iso`            | ISO image management                                   |
| `zcp affinity-group` | Affinity group management                              |
| `zcp backup`         | VM and block storage backups                           |
| `zcp region`         | Region listing                                         |

### Expanded existing commands

- **instance**: reboot, reset, tags, change-hostname, change-password, change-plan, change-OS, add-network, addons
- **instance create**: now requires `--blockstorage-plan` flag (e.g. `50-gb-2`, `100gb`)
- **project**: added `delete` subcommand with confirmation prompt
- **billing cancel-service**: now requires `--service` flag and supports `--reason`, `--type`
- **network**: egress firewall rule management
- **vpc**: ACL management, VPN gateway management
- **loadbalancer**: rule creation, VM attachment to rules
- **Discovery**: cloud-providers, currencies, storage-categories, billing-cycles, unit-pricings

### Auto-approve for CI/CD

All destructive commands now respect the global `--auto-approve` (or `-y`) flag, skipping confirmation prompts. Useful for scripting and automation pipelines:

```bash
zcp -y project delete my-project
zcp -y billing cancel-service my-vm --service "Virtual Machine"
```

### RESTful API with pagination

All endpoints now use clean RESTful paths with slug identifiers. List responses include pagination metadata (`current_page`, `per_page`, `total`).

### VM creation example

```bash
zcp instance create \
  --name my-vm \
  --cloud-provider nimbo \
  --project my-project \
  --region noida \
  --template ubuntu-22f \
  --plan bp-4vc-8gb \
  --billing-cycle hourly \
  --storage-category nvme \
  --blockstorage-plan 50-gb-2 \
  --ssh-key my-key
```

---

## Breaking Changes

- **Config format**: `apikey`/`secretkey` replaced by `bearer_token`. Run `zcp profile add` to reconfigure.
- **Zone commands**: `zcp zone list` still works but `zcp region list` is the new canonical command.
- **UUID flags**: Flags like `--zone-uuid`, `--uuid` replaced by slug-based identifiers.

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

**Full Changelog**: https://github.com/zsoftly/zcp-cli/compare/0.0.5...0.0.6
