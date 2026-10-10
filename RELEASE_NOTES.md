# Unreleased

The next release will include the following fixes. No release version or date
has been selected.

## Kubernetes resource plans and billing

`zcp kubernetes create` now requires separate resource selections for the
control plane, worker nodes, and one root-volume plan used for every node:

```bash
zcp kubernetes create --name my-cluster --version v1.37.0 \
  --control-plane-plan k8s-cpi-yul --worker-plan k8s-li-yul \
  --storage-plan b2g1 --storage-category pro-nvme \
  --region yul-1 --project default-9 --billing-cycle hourly \
  --workers 3 --enable-csi --ssh-key mykey
```

The cluster object has no recurring charge. The selected billing cycle applies
to the subscriptions created for each control-plane VM, worker VM, and root
volume. Network, load balancer, and public-IP subscriptions depend on the
platform packages and feature flags enabled for those resources.

`zcp kubernetes get` now shows resource totals, separate control-plane and
worker configuration, root-volume count, network, and autoscaling status when
the API returns them. `zcp kubernetes scale` supports manual worker counts,
enabling autoscaling with `--min-workers` and `--max-workers`, and disabling
autoscaling with an explicit `--workers` count.

Use `--enable-csi` to request Cloud Storage Integration for a supported
cluster. Creating a high-availability cluster requires at least two control
nodes.

## VM backup schedules

`zcp vm-backup schedule` manages the scheduler-backed VM backup policies shown
in the cloud portal. Create policies with `hourly`, `dailyAt`,
`everyOtherDay`, `weeklyOn`, or `monthlyOn`, an IANA timezone, a retention
count, and an optional request for an immediate backup. Weekly policies use Sunday `0`
through Saturday `6`. Monthly policies use day `1` through `28`, matching the
provider's supported range.

```bash
zcp vm-backup schedule create my-vm --name nightly --interval dailyAt --at 01:00 \
  --timezone America/Toronto --retention 7 --immediate
zcp vm-backup schedule list
zcp vm-backup schedule pause <id>
zcp vm-backup schedule run-now <id>
```

Policies support get, update, pause, resume, run-now, and delete operations.
Deleting a policy leaves backups it has already created in place.
Policy output includes the last-run result and any reported error; an active
policy does not by itself confirm a completed backup.

The current platform rejects `everyOtherDay` after persisting a
policy. If it returns an error, list policies before retrying and remove any
unintended policy.

## VM backup create output

`zcp vm-backup create` will print the created schedule, including its slug. It
will honor `--output json` and `--output yaml` without mixing status text into
structured output. When the create endpoint provides only an acknowledgement,
the command will identify one new matching schedule from a scoped snapshot and
read-only lookup. It will not repeat an accepted create request. If it cannot
identify the schedule, it will tell you to list schedules before trying again.

## DNS status and pagination

`zcp dns show <slug>` will preserve a status supplied by the detail endpoint.
When that endpoint omits status, it will use the exactly matching entry from
`zcp dns list`. If no status is available, the display remains `-`. If the
lookup fails, the command returns an error.

`dns.Service.List` will retrieve every reported page and return an error rather
than a partial result when a later page fails or pagination metadata is
inconsistent.

---

# zcp v0.0.31 Release Notes

v0.0.31 fixes ACL rule listings that stopped at the first API page, adds
bounded ACL rule listing controls, and includes security updates to the Go
toolchain and networking dependencies.

## Highlights

**List every rule in an ACL.** `zcp acl rules` now retrieves every API page, so
ACLs with more than the default page size return their complete rule set.

```bash
zcp acl rules <vpc-slug> <acl-name-or-id>
```

**Safer API pagination.** If a later page fails or reports inconsistent
pagination metadata, the command returns an error instead of a partial rule
list. A response that explicitly reports `total: 0` must contain no rules.

**Choose how many ACL rules to return.** Use `--max-items` for a bounded
result, `--starting-token` to resume it, `--page-size` to set the API request
size, or `--no-paginate` to request one API page. Bounded and single-page JSON
and YAML output use an object with a `rules` array and optional `next_token`.
Table output prints the continuation token on standard error.

```bash
zcp acl rules <vpc-slug> <acl-name-or-id> --max-items 25 --output json
zcp acl rules <vpc-slug> <acl-name-or-id> --max-items 25 --starting-token '<next_token>' --output json
zcp acl rules <vpc-slug> <acl-name-or-id> --no-paginate --output yaml
```

`--no-paginate` cannot be combined with the other pagination flags. A
continuation token applies only to the same ACL and retains its page size.

**Security updates.** The selected Go toolchain is now Go 1.26.9 and
`golang.org/x/net` is now v0.60.0. These updates resolve the reachable
standard-library and networking vulnerabilities reported by `govulncheck`.

## Go library consumers

`acl.Service.ListRules` now returns rules from every page. Callers receive an
error and no partial rule list when a later page cannot be retrieved or its
pagination metadata is inconsistent. Terraform provider maintainers must adopt
this `zcp-cli` module release to receive this behavior.

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
zcp version   # zcp version v0.0.31
```

First-time setup after installing:

```bash
zcp profile add default --region yul-1 --project default-9   # prompts for bearer token
zcp auth validate
```
