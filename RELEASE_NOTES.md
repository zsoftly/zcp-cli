# zcp v0.0.24 Release Notes

## Deleting a VM now releases its public IP

`zcp instance delete` used to leave a deleted VM's auto-assigned public IP
allocated (and billable). It now frees the IP as part of the deletion. Deletion
is asynchronous — poll `zcp instance get <slug>` to confirm.

```bash
zcp instance delete my-vm --yes                            # deletes the VM and releases its IP
zcp instance delete my-vm --yes --delete-public-ip=false   # keep the IP allocated
```

The deprecated `--force` flag is now a hidden no-op (deletion is already immediate).

## Deleting a load balancer, and freeing its IP

`zcp loadbalancer delete` now deletes the way the Web UI does. Unlike a VM's IP, a
load balancer's public IP is a **separate, reusable resource** (you pick or acquire
it), so it is kept by default. Pass `--release-ip` to also free it.

```bash
zcp loadbalancer delete my-lb --yes                    # deletes the LB; its IP is kept (reusable)
zcp loadbalancer delete my-lb --yes --release-ip       # also release the LB's dedicated IP
zcp loadbalancer delete my-lb --yes --billing-cycle monthly   # for a monthly-billed LB
```

`--release-ip` never releases the network's **source-NAT** IP; if it can't confirm
an IP is safe to release, it skips it and prints the exact `zcp ip release <slug>`
command.

## Instances now show their public IP

`zcp instance list` and `zcp instance get` now display the public IP (previously
blank), and `get` also shows the billing cycle. Empty IP cells show `-`.
_Contributed by @cokerrd._

## `loadbalancer list` and `ip list` return all results

Both commands used to show only the first page. They now page through everything,
so large accounts see the full list.

## `ip allocate` validates `--vpc` / `--network`

Allocating an IP requires exactly one of `--vpc` or `--network`. Passing neither
used to fail with a raw API 500; it's now caught client-side with a clear message.
_Contributed by @cokerrd._

```bash
zcp ip allocate --plan ipv4-yul --billing-cycle hourly --network en-001001-0018
zcp ip allocate --plan ipv4-yul --billing-cycle hourly --vpc my-vpc
```

---

## Installation and upgrade

The install script installs the latest release and upgrades an existing
installation in place.

**Linux / macOS**

```bash
curl -fsSL https://github.com/zsoftly/zcp-cli/releases/latest/download/install.sh | bash
```

**Windows (PowerShell)**

```powershell
irm https://github.com/zsoftly/zcp-cli/releases/latest/download/install.ps1 | iex
```

**Manual download:** grab your platform's binary from the
[Releases](https://github.com/zsoftly/zcp-cli/releases) page, `chmod +x`, and
place it on your `PATH`.

**Verify:**

```bash
zcp version   # zcp version v0.0.24
```

First-time setup after installing:

```bash
zcp profile add default --region yul-1 --project default-9   # prompts for bearer token
zcp auth validate
```
