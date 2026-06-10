# zcp v0.0.14 Release Notes

## ⚠ Breaking Changes — API package paths changed

This release moves all API service packages and the HTTP client out of `internal/`
into `pkg/` so that external Go modules (such as the ZCP Terraform provider) can
import them directly.

**CLI end users are not affected.** The binary is identical to v0.0.12.

### Migration guide for Go library consumers

Run a global find-and-replace in your project:

```bash
# macOS / BSD sed
find . -name '*.go' | xargs sed -i '' \
  's|github.com/zsoftly/zcp-cli/internal/httpclient|github.com/zsoftly/zcp-cli/pkg/httpclient|g'
find . -name '*.go' | xargs sed -i '' \
  's|github.com/zsoftly/zcp-cli/internal/api/|github.com/zsoftly/zcp-cli/pkg/api/|g'

# Linux sed
find . -name '*.go' | xargs sed -i \
  's|github.com/zsoftly/zcp-cli/internal/httpclient|github.com/zsoftly/zcp-cli/pkg/httpclient|g'
find . -name '*.go' | xargs sed -i \
  's|github.com/zsoftly/zcp-cli/internal/api/|github.com/zsoftly/zcp-cli/pkg/api/|g'
```

Then update your `go.mod` to reference `v0.0.14`:

```bash
go get github.com/zsoftly/zcp-cli@v0.0.14
go mod tidy
```

### What moved

| Old                            | New                       |
| ------------------------------ | ------------------------- |
| `internal/httpclient`          | `pkg/httpclient`          |
| `internal/api/acl`             | `pkg/api/acl`             |
| `internal/api/affinitygroup`   | `pkg/api/affinitygroup`   |
| `internal/api/apierrors`       | `pkg/api/apierrors`       |
| `internal/api/autoscale`       | `pkg/api/autoscale`       |
| `internal/api/backup`          | `pkg/api/backup`          |
| `internal/api/billing`         | `pkg/api/billing`         |
| `internal/api/billingcycle`    | `pkg/api/billingcycle`    |
| `internal/api/cloudprovider`   | `pkg/api/cloudprovider`   |
| `internal/api/currency`        | `pkg/api/currency`        |
| `internal/api/dashboard`       | `pkg/api/dashboard`       |
| `internal/api/dns`             | `pkg/api/dns`             |
| `internal/api/egress`          | `pkg/api/egress`          |
| `internal/api/firewall`        | `pkg/api/firewall`        |
| `internal/api/instance`        | `pkg/api/instance`        |
| `internal/api/ipaddress`       | `pkg/api/ipaddress`       |
| `internal/api/iso`             | `pkg/api/iso`             |
| `internal/api/kubernetes`      | `pkg/api/kubernetes`      |
| `internal/api/loadbalancer`    | `pkg/api/loadbalancer`    |
| `internal/api/marketplace`     | `pkg/api/marketplace`     |
| `internal/api/monitoring`      | `pkg/api/monitoring`      |
| `internal/api/network`         | `pkg/api/network`         |
| `internal/api/objectstorage`   | `pkg/api/objectstorage`   |
| `internal/api/plan`            | `pkg/api/plan`            |
| `internal/api/portforward`     | `pkg/api/portforward`     |
| `internal/api/product`         | `pkg/api/product`         |
| `internal/api/project`         | `pkg/api/project`         |
| `internal/api/region`          | `pkg/api/region`          |
| `internal/api/response`        | `pkg/api/response`        |
| `internal/api/server`          | `pkg/api/server`          |
| `internal/api/snapshot`        | `pkg/api/snapshot`        |
| `internal/api/sshkey`          | `pkg/api/sshkey`          |
| `internal/api/storagecategory` | `pkg/api/storagecategory` |
| `internal/api/store`           | `pkg/api/store`           |
| `internal/api/support`         | `pkg/api/support`         |
| `internal/api/template`        | `pkg/api/template`        |
| `internal/api/userprofile`     | `pkg/api/userprofile`     |
| `internal/api/virtualrouter`   | `pkg/api/virtualrouter`   |
| `internal/api/vmbackup`        | `pkg/api/vmbackup`        |
| `internal/api/vmsnapshot`      | `pkg/api/vmsnapshot`      |
| `internal/api/volume`          | `pkg/api/volume`          |
| `internal/api/vpc`             | `pkg/api/vpc`             |
| `internal/api/vpn`             | `pkg/api/vpn`             |

### Packages that remain internal

These are CLI-only and are **not** part of the public API:

- `internal/commands`
- `internal/config`
- `internal/output`
- `internal/version`

---

## Note on release tag format

Starting with this release, tags use the `v` prefix (`v0.0.14`) to align with
Go module conventions and the Terraform Registry. Previous tags (`0.0.1`–`0.0.12`)
are preserved but the old format will not be used going forward.

---

## Installation

**macOS / Linux / WSL:**

```bash
curl -fsSL https://github.com/zsoftly/zcp-cli/releases/latest/download/install.sh | bash
```

**Windows (PowerShell):**

```powershell
irm https://github.com/zsoftly/zcp-cli/releases/latest/download/install.ps1 | iex
```

**Verify:**

```bash
zcp version
# zcp version v0.0.14
```

---

## Full Changelog

https://github.com/zsoftly/zcp-cli/compare/0.0.12...v0.0.14
