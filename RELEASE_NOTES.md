# zcp v0.0.24 Release Notes

## Deleting a VM now releases its public IP

`zcp instance delete` previously called an endpoint that ignored the
`delete_public_ip` request, so a deleted VM left its auto-assigned public IP
`Allocated` (and billable). It now uses the same service-cancellation workflow as
the CMP Web UI (`POST /billing/service-cancel-requests/{slug}`), which releases
the IP as part of the cancellation. Verified live end to end: the VM is deleted
and its public IP is released with no orphan left behind.

```bash
zcp instance delete my-vm --yes                       # releases the auto-assigned IP too
zcp instance delete my-vm --yes --delete-public-ip=false   # keep the IP allocated
```

Deletion is asynchronous — a successful response means the request was accepted,
not that the VM is already gone; poll `zcp instance get <slug>` to confirm. The
deprecated `--force` flag is now a hidden no-op (deletion is already immediate).

## Deleting a load balancer, and freeing its IP

`zcp loadbalancer delete` now goes through the same service-cancellation workflow
(matching the Web UI, with consistent async deletion). A load balancer's public
IP is a **separate, reusable resource** — as in the Web UI, you Choose an existing
IP or Acquire a new one — so it is kept by default. Pass `--release-ip` to also
free it.

```bash
zcp loadbalancer delete my-lb --yes                   # deletes the LB; its IP is kept (reusable)
zcp loadbalancer delete my-lb --yes --release-ip      # also release the LB's dedicated IP
zcp loadbalancer delete my-lb --yes --billing-cycle monthly   # for a monthly-billed LB
```

`--release-ip` never releases the network's **source-NAT** IP (that would break
the network); if it can't confirm the IP is safe, it skips the release and prints
the exact `zcp ip release <slug>` command. Verified live: a dedicated LB IP is
released, a source-NAT IP is correctly left alone.

## `loadbalancer list` and `ip list` return all results

Both commands previously showed only the first page of results. They now page
through the full set, so large accounts see everything (and `--release-ip` can
find a load balancer that lands on a later page).

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
