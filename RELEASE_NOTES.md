# zcp 0.0.9 Release Notes

## What's New

### Environment variable support

All create commands now respect environment variables for the three most commonly repeated flags. Set them once and every command picks them up:

```bash
export ZCP_CLOUD_PROVIDER=zcp
export ZCP_REGION=yow-1
export ZCP_PROJECT=my-project

# Before: every command needed all 3 flags
zcp instance create --name my-vm --cloud-provider zcp --region yow-1 --project my-project --template ubuntu-22f --plan bp-4vc-8gb --billing-cycle hourly --storage-category nvme --blockstorage-plan 50-gb-2

# Now: just the resource-specific flags
zcp instance create --name my-vm --template ubuntu-22f --plan bp-4vc-8gb --billing-cycle hourly --storage-category nvme --blockstorage-plan 50-gb-2
```

Works across all create commands: instance, volume, vpc, network, kubernetes, dns, loadbalancer, autoscale, snapshot, vm-snapshot, vm-backup, virtual-router, vpn, iso, affinity-group, template, backup.

### Zero-config mode

The CLI can now run with just environment variables — no config file needed:

```bash
export ZCP_BEARER_TOKEN=your-token
export ZCP_API_URL=https://api.zcp.zsoftly.ca
zcp region list
```

### All new environment variables

| Variable             | Overrides                                 | Example                          |
| -------------------- | ----------------------------------------- | -------------------------------- |
| `ZCP_BEARER_TOKEN`   | Profile `bearer_token`                    | `export ZCP_BEARER_TOKEN=abc123` |
| `ZCP_API_URL`        | Profile `api_url`                         | `export ZCP_API_URL=https://...` |
| `ZCP_PROFILE`        | Active profile (when `--profile` not set) | `export ZCP_PROFILE=staging`     |
| `ZCP_PROJECT`        | `--project` flag                          | `export ZCP_PROJECT=my-project`  |
| `ZCP_REGION`         | `--region` flag                           | `export ZCP_REGION=yow-1`        |
| `ZCP_CLOUD_PROVIDER` | `--cloud-provider` flag                   | `export ZCP_CLOUD_PROVIDER=zcp`  |
| `ZCP_OUTPUT`         | `--output` flag                           | `export ZCP_OUTPUT=json`         |
| `ZCP_DEBUG`          | `--debug` flag                            | `export ZCP_DEBUG=true`          |

Precedence: CLI flag > environment variable > profile config > default.

### Bug fix: Kubernetes create

`--billing-cycle` validation was accidentally removed in v0.0.8. Restored — the API requires it (confirmed via Postman collection).

---

## Installation

**macOS/Linux/WSL:**

```bash
curl -fsSL https://github.com/zsoftly/zcp-cli/releases/latest/download/install.sh | bash
```

**Windows:**

```powershell
irm https://github.com/zsoftly/zcp-cli/releases/latest/download/install.ps1 | iex
```

**Full Changelog**: https://github.com/zsoftly/zcp-cli/compare/0.0.8...0.0.9
