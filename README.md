# ZCP CLI

The official command-line interface for the ZSoftly Cloud Platform

[![CI](https://github.com/zsoftly/zcp-cli/actions/workflows/build.yml/badge.svg)](https://github.com/zsoftly/zcp-cli/actions/workflows/build.yml)
![Go](https://img.shields.io/badge/Go-1.26.1-blue)

---

## Overview

ZCP CLI (`zcp`) is a full-featured command-line tool for managing resources on the ZSoftly Cloud Platform. It covers the complete lifecycle of cloud infrastructure: compute instances, block storage, snapshots, networks, VPCs, firewalls, load balancers, VPN gateways, SSH keys, Kubernetes clusters, DNS, backups, autoscale policies, monitoring, projects, billing, and support. All commands support table, JSON, and YAML output, making the CLI equally suited for interactive use and automation pipelines.

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

# 2. Add your first profile — prompts for bearer token
zcp profile add default

# 3. Confirm your credentials work
zcp auth validate

# 4. Discover available regions
zcp region list

# 5. List your instances
zcp instance list
```

---

## Configuration

### Adding a Profile

Run the interactive setup to create your first profile:

```bash
zcp profile add default
```

You will be prompted for:

- Bearer token
- API URL (leave blank to use the default)

To add a named profile non-interactively:

```bash
zcp profile add staging --bearer-token YOUR_TOKEN
```

### Config File Location

| Platform    | Path                        |
| ----------- | --------------------------- |
| Linux/macOS | `~/.config/zcp/config.yaml` |
| Windows     | `%AppData%\zcp\config.yaml` |

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
zcp profile update prod --bearer-token <new-token>
zcp profile update prod --api-url-override https://custom.api.url

# Rename a profile
zcp profile rename staging prod

# Delete a profile
zcp profile delete old-profile
```

### Environment Variables

You can override configuration and flags using environment variables:

- `ZCP_BEARER_TOKEN`: Overrides profile credentials.
- `ZCP_API_URL`: Overrides the API base URL.
- `ZCP_PROJECT`: Sets the default project slug.
- `ZCP_REGION`: Sets the default region slug.
- `ZCP_CLOUD_PROVIDER`: Sets the default cloud provider.
- `ZCP_OUTPUT`: Sets the default output format (`json`, `yaml`, `table`).
- `ZCP_DEBUG`: Set to `true` to enable verbose debug output.

See [docs/configuration.md](docs/configuration.md) for the full list and usage examples.

---

## Commands Reference

Use `zcp <command> --help` for the full flag list of any command.

### Discovery

```bash
# Regions
zcp region list

# Plans by service type (preferred over legacy 'offering' commands)
zcp plan vm                # Virtual Machine plans
zcp plan storage           # Block Storage plans
zcp plan kubernetes        # Kubernetes plans
zcp plan lb                # Load Balancer plans
zcp plan router            # Virtual Router plans
zcp plan ip                # IP Address plans
zcp plan vm-snapshot       # VM Snapshot plans
zcp plan template          # My Template plans
zcp plan iso               # ISO plans
zcp plan backup            # Backup plans

# Discovery helpers
zcp cloud-provider list    # Available cloud providers
zcp server list            # Available servers
zcp currency list          # Available currencies
zcp billing-cycle list     # Available billing cycles
zcp storage-category list  # Available storage categories

# VM templates
zcp template list

# Marketplace
zcp marketplace list

# ISO images
zcp iso list

# Store
zcp store list
```

### Compute

```bash
# List and inspect
zcp instance list
zcp instance get <slug>

# Create — use --wait to block until the instance is Running
zcp instance create \
  --name my-vm \
  --cloud-provider zcp \
  --project my-project \
  --region yow-1 \
  --template ubuntu-22f \
  --plan bp-4vc-8gb \
  --billing-cycle hourly \
  --storage-category nvme \
  --blockstorage-plan 50-gb-2 \
  --ssh-key mykey

zcp instance create ... --wait

# Lifecycle
zcp instance start <slug>
zcp instance stop <slug>
zcp instance reboot <slug>
zcp instance reset <slug>            # Hard reset (prompts for confirmation)

# Change plan (instance must be stopped)
zcp instance change-plan <slug> --plan <new-plan> --billing-cycle hourly

# Change hostname
zcp instance change-hostname <slug> --hostname new-hostname

# Change OS (DESTRUCTIVE — reinstalls the VM)
zcp instance change-os <slug> --template ubuntu-22f

# Change startup script
zcp instance change-script <slug> --user-data "#!/bin/bash\napt update"

# Change password
zcp instance change-password <slug> --password "newSecureP@ss"

# Add a network to a running instance
zcp instance add-network <slug> --network <network-slug>

# Activity logs
zcp instance logs <slug>

# Tags
zcp instance tag-create <slug> --key env --value prod
zcp instance tag-delete <slug> --key env

# Addons
zcp instance addons <slug>

# Open an SSH session directly from the CLI
zcp instance ssh <slug>
zcp instance ssh <slug> --user ubuntu
zcp instance ssh <slug> --user root --identity-file ~/.ssh/my-key.pem --port 2222

# To cancel/delete an instance, use billing cancel-service:
zcp billing cancel-service <subscription-slug> --service "Virtual Machine" --reason not_needed_anymore
```

The `--wait` flag on `create`, `start`, and `stop` polls the API until the instance reaches the target state, printing progress to stderr.

### Storage

```bash
# Volumes
zcp volume list
zcp volume create \
  --name my-disk \
  --project my-project \
  --cloud-provider zcp \
  --region yow-1 \
  --billing-cycle hourly \
  --storage-category nvme \
  --plan 50-gb-2
zcp volume create ... --vm <vm-slug>   # Attach on creation
zcp volume attach <volume-slug> --vm <vm-slug>
zcp volume detach <volume-slug>

# Snapshots
zcp snapshot list
zcp snapshot create \
  --volume <slug> \
  --name my-snapshot \
  --plan snapshot-per-gb \
  --cloud-provider zcp \
  --region yow-1 \
  --billing-cycle hourly \
  --project my-project
zcp snapshot revert <snapshot-slug> --volume <volume-slug>

# VM snapshots (whole-instance checkpoint)
zcp vm-snapshot list
zcp vm-snapshot create \
  --vm <slug> \
  --name my-checkpoint \
  --plan basic \
  --billing-cycle hourly \
  --project my-project \
  --cloud-provider zcp \
  --region yow-1
zcp vm-snapshot revert <slug>
```

### Networking

```bash
# Networks
zcp network list
zcp network categories
zcp network create --name my-net --category <slug> --cloud-provider zcp --region yow-1 --project my-project
zcp network update <slug> --name "New Name"

# VPC tier networks
zcp network create --name public-tier --cloud-provider zcp --region yow-1 --project my-project \
  --vpc <vpc-slug> --type Vpc --gateway 10.1.1.1 --netmask 255.255.255.0 --acl-id <acl-id>

# Public IP addresses
zcp ip list
zcp ip allocate --network <slug>
zcp ip release <slug>
zcp ip static-nat enable <slug> --instance <slug> --network <slug>

# Firewall rules (ingress)
zcp firewall list
zcp firewall create --ip <slug> --protocol tcp --start-port 80 --end-port 80

# Egress rules
zcp egress list
zcp egress create --network <slug> --protocol tcp

# Port forwarding
zcp portforward list
zcp portforward create \
  --ip <slug> \
  --protocol tcp \
  --public-port 2222 \
  --private-port 22 \
  --instance <slug>

```

### Advanced Networking

```bash
# VPCs
zcp vpc list
zcp vpc create \
  --name my-vpc \
  --cloud-provider zcp \
  --region yow-1 \
  --project my-project \
  --plan vpc-1 \
  --network-address 10.1.0.1 \
  --size 16 \
  --billing-cycle hourly \
  --storage-category nvme

# Network ACL lists
zcp acl list <vpc-slug>
zcp acl create <vpc-slug> --name my-acl --description "Allow web traffic"

# Public load balancers
zcp loadbalancer list
zcp loadbalancer create --ip <slug> --name my-lb --algorithm roundrobin
zcp loadbalancer delete <slug>

# VPN gateways and connections
zcp vpn list
zcp vpn create --vpc <slug> --name my-vpn
zcp vpn delete <slug>
```

### Security and Access

```bash
# SSH keys
zcp ssh-key list
zcp ssh-key create --name mykey --public-key "$(cat ~/.ssh/id_rsa.pub)"
zcp ssh-key delete <slug>

# Affinity groups
zcp affinity-group list
zcp affinity-group create --name my-ag --type host-affinity
zcp affinity-group delete <slug>
```

### DNS

```bash
# Domains
zcp dns list
zcp dns show <slug>

# Create a domain
zcp dns create --name example.com --project my-project --cloud-provider zcp --region yow-1 --dns-provider dns-provider

# Create a record
zcp dns record-create --domain <domain-slug> --name www --type A --content 192.0.2.1
zcp dns record-create --domain <domain-slug> --name mail --type MX --content mail.example.com --ttl 3600

# Delete a record or domain
zcp dns record-delete --domain <domain-slug> --record-id 42
zcp dns delete <domain-slug>
```

### Backup

```bash
zcp backup list
zcp backup get <slug>
zcp backup create --instance <slug> --name my-backup
zcp backup restore <slug>
zcp backup delete <slug>
```

### Autoscale

```bash
zcp autoscale list
zcp autoscale get <slug>
zcp autoscale create --name my-policy --min 1 --max 5 --cloud-provider zcp --region yow-1 --project my-project
zcp autoscale delete <slug>
```

### Monitoring

```bash
zcp monitoring list
zcp monitoring get <slug>
zcp monitoring create --instance <slug> --type cpu --threshold 80
zcp monitoring delete <slug>
```

### Project

```bash
zcp project list
zcp project create --name my-project --icon cloud-15 --purpose "Development"
zcp project update <slug> --name "New Name" --description "Updated description"
zcp project delete <slug>
zcp project dashboard <slug>

# Project users
zcp project user list <slug>
zcp project user add <slug> --email alice@example.com --role admin

# Project icons
zcp project icon list
```

### Kubernetes

```bash
# 'k8s' is accepted as an alias for 'kubernetes'
zcp kubernetes list
zcp kubernetes create \
  --name my-cluster \
  --version v1.28.4 \
  --plan k8s-plan-1 \
  --region yow-1 \
  --project my-project \
  --cloud-provider zcp \
  --billing-cycle monthly \
  --workers 3 \
  --ssh-key mykey

# HA cluster with multiple control nodes
zcp kubernetes create \
  --name ha-cluster \
  --version v1.28.4 \
  --plan k8s-plan-1 \
  --region yow-1 \
  --project my-project \
  --cloud-provider zcp \
  --billing-cycle monthly \
  --workers 3 \
  --control-nodes 3 \
  --ha \
  --ssh-key mykey

# Start / stop / upgrade
zcp kubernetes start <slug>
zcp kubernetes stop <slug>
zcp kubernetes upgrade <slug> --plan k8s-plan-2

# To cancel/delete a cluster, use billing cancel-service:
zcp billing cancel-service <subscription-slug> --service "Kubernetes" --reason not_needed_anymore
```

### Billing and Admin

```bash
# Account balance and costs
zcp billing balance
zcp billing costs
zcp billing monthly-usage
zcp billing usage
zcp billing credit-limit
zcp billing service-counts
zcp billing free-credits

# Invoices and payments
zcp billing invoices
zcp billing invoices --page 2
zcp billing invoices-count
zcp billing payments

# Subscriptions
zcp billing subscriptions active
zcp billing subscriptions inactive
zcp billing contracts
zcp billing trials

# Cancel a service (instances, volumes, IPs, etc.)
zcp billing cancel-service <subscription-slug> --service "Virtual Machine" --reason not_needed_anymore
zcp billing cancel-service <subscription-slug> --service "Block Storage" --reason not_needed_anymore --type Immediate
zcp billing cancel-requests

# Coupons
zcp billing coupons
zcp billing redeem-coupon SAVE50

# Budget alerts
zcp billing budget-alert
zcp billing budget-alert-set --amount 500 --threshold 80 --enabled

```

### Support

```bash
zcp support list
zcp support get <ticket-id>
zcp support create --subject "Issue title" --description "Details"
zcp support close <ticket-id>
```

### Dashboard

```bash
zcp dashboard summary
zcp dashboard status
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
zcp region list
```

```
SLUG       NAME      COUNTRY  ACTIVE
----       ----      -------  ------
toronto    Toronto   Canada   true
montreal   Montreal  Canada   true
```

**JSON**

```bash
zcp region list --output json
```

```json
[
  {
    "slug": "toronto",
    "name": "Toronto",
    "country_name": "Canada",
    "is_active": "true"
  }
]
```

**YAML**

```bash
zcp region list --output yaml
```

```yaml
- slug: toronto
  name: Toronto
  country_name: Canada
  is_active: "true"
```

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
