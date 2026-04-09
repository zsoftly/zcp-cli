# zcp 0.0.8 Release Notes

## What's New

### VPC create fixed

VPC creation now works with the correct payload structure:

```bash
zcp vpc create \
  --name my-vpc \
  --cloud-provider nimbo \
  --region noida \
  --project default-124 \
  --plan vpc-1 \
  --cidr 10.1.0.1 \
  --size 16 \
  --billing-cycle hourly \
  --storage-category nvme
```

Key: `--cidr` is the network address (not CIDR notation), `--size` is the mask separately.

### ACL list creation fixed

`zcp vpc acl-create-rule` and `zcp acl create` now correctly create ACL lists:

```bash
zcp vpc acl-create-rule my-vpc --name allow-web --description "Allow HTTP"
zcp acl create my-vpc --name private-acl --description "Deny all inbound"
```

### Create commands gain required flags

`--cloud-provider`, `--region`, `--project` added to: network, vpc, virtualrouter, dns, vpn, autoscale create commands.

### Volume Size type fix

Volume list no longer fails when the API returns size as a number.

### Roadmap published

See `docs/roadmap.md` for what's working, what's coming, and what's blocked on the platform.

---

## Known limitations (blocked on platform)

These require API changes from the STKCNSL team:

- **No DELETE endpoints** for VPCs, networks, virtual routers, IP addresses, or ACL lists
- **No ACL rule CRUD** — can create ACL lists but not rules inside them
- **Network create (isolated)** — `networkofferingid` not resolvable for nimbo/noida
- **DNS create** — needs admin-side `cloud_provider_setup` provisioning
- **billing cancel-service for VPCs** — returns "service not found"

See `docs/roadmap.md` for full details.

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

**Full Changelog**: https://github.com/zsoftly/zcp-cli/compare/0.0.7...0.0.8
