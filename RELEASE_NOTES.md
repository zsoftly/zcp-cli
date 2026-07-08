# zcp v0.0.23 Release Notes

## `instance --wait` now reports the VM's real state

`zcp instance create --wait` (and `start --wait` / `stop --wait`) previously
polled the CMP's cached list/show endpoint, which can keep reporting `Starting`
for many minutes after a VM is actually `Running`. On affected deployments
`--wait` could hang until it timed out even though the VM was already up.

`--wait` now polls the live `GET /virtual-machines/{slug}/meta` endpoint, which
performs a real-time reconcile against the underlying platform (CloudStack/APC)
and returns the authoritative state. Verified live end to end: with `--wait`,
`create` returned `Running` from `/meta` while the plain `instance list` still
showed `Starting`.

This is a client-side workaround for a CMP background state-sync issue (the
platform's own reconciliation is unreliable; state only refreshes on demand).
The on-demand `/meta` sync is authoritative, so the CLI polls it.

## `instance delete --delete-public-ip` help corrected — the flag is currently a no-op

The flag advertised that deleting a VM releases its auto-assigned public IP, but
that never worked against the live API: the `DELETE` endpoint ignores it, and
the IP-releasing `PUT .../destroy` endpoint currently rejects API-token auth (a
CMP bug, reported and under fix). Until that lands, the help text, confirmation
prompt, and command examples now state plainly that the IP is **not** released
automatically, and that you must free it manually:

```bash
zcp instance delete my-vm --yes
zcp ip release <ip-slug> --yes
```

No behavior change — this corrects misleading messaging only. The real fix
(routing `instance delete` through `PUT .../destroy`) is implemented and
verified at the request level, and is held until the API accepts token auth.

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
zcp version   # zcp version v0.0.23
```

First-time setup after installing:

```bash
zcp profile add default --region yul-1 --project default-9   # prompts for bearer token
zcp auth validate
```

---

## Fixed

### `instance --wait` reflects the real state

```bash
# Create and wait: returns when the VM is actually Running (polled via /meta),
# even while `instance list` still reports Starting.
zcp instance create --name my-vm --project default-9 --region yul-1 \
  --template ubuntu-2604-lts-1 --plan ca2m --billing-cycle hourly \
  --network-plan pnet-yul --storage-category premium-ssd --wait
```

SDK consumers get a new `instance.Service.Meta(ctx, slug)` method that returns
the live, hypervisor-synced view (authoritative `state`).

### `instance delete --delete-public-ip` messaging

```bash
# The auto-assigned public IP is NOT released automatically yet (known CMP API
# bug, under fix). Free it manually after deleting:
zcp instance delete my-vm --yes && zcp ip release <ip-slug> --yes
```
