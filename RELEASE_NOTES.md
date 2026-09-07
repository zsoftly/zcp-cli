# zcp v0.0.28 Release Notes

Security fixes, working backup commands, a public-IP-first `instance ssh`, and
several output fixes. This release also carries everything prepared for v0.0.27,
which was never tagged or published. Upgrading from v0.0.26 picks up both sets
of changes.

## Security

**`--debug` output no longer leaks your API token.** Some list endpoints echo
the account's bearer token inside the response body. Debug output now redacts
that field, other credential fields such as `password` and `client_secret`, and
the configured token wherever it appears. So do error messages built from
unparseable response bodies. If you have pasted `--debug` output into a ticket
or a chat before this release, rotate the token.

**Dependencies and toolchain.** `golang.org/x/crypto` moved to v0.56.0, which
fixes two denial-of-service bugs in its SSH package (GO-2026-6354 and
GO-2026-6355). The shipped binary does not use that package. This closes the
advisories rather than a reachable hole. The Go toolchain moved from 1.26.5 to
1.26.8 across two steps. The 1.26.6 step, prepared for v0.0.27, cleared five
standard-library vulnerabilities reachable from the object storage commands and
one in `golang.org/x/net`. A vulnerability scan of this build reports nothing
reachable.

## Backups work again

**`backup list` decodes its own response.** The API returns the scheduled hour
as a string in list responses and as a number in create responses. The CLI only
accepted the number, so `backup list` failed in every region with a decode
error. Both forms are accepted now. The Terraform provider shares this code. Its
`zcp_volume_backup` read failure, which blocked `terraform plan`, clears once
the provider picks up this release.

**`vm-backup delete` works.** The API route for VM backup schedules rejects
`DELETE`, so the command had never succeeded. It now submits a
service-cancellation request, the same workflow `instance delete` uses. It
reports that the removal is running in the background.

**`--interval` is validated before the request.** The API accepts only `dailyAt`
and `hourlyAt`. `backup create` and `vm-backup create` now reject any other
value up front and list the accepted ones. `vm-backup create` no longer defaults
to `daily`, which the API always rejected.

**Clearer listings.** `backup list` shows the volume slug instead of a blank ID,
plus AT and SCHEDULED AT columns. `vm-backup list` shows the VM slug and the
same schedule columns. In JSON output, `backup list` renames `volume_id` to
`volume`, and `vm-backup list` drops the always-empty `id` and `state` keys.

```bash
zcp backup create --volume root-1234 --interval dailyAt --at 1 \
  --plan backup-yul --billing-cycle hourly --region yul-1 --project default-9
zcp backup list
zcp vm-backup delete backup-my-vm-dailyat --yes
```

## `instance ssh` prefers the public IP

The command always connected over the private address. That hangs for anyone
outside the VPC, even when a public IP was attached. It now uses the public IP
when one exists and falls back to the private IP otherwise. The new
`--use-public` and `--use-private` flags force that choice. An explicit
`--user root` is honoured instead of being replaced by the VM's reported
username.

```bash
zcp instance ssh my-vm                 # public IP when attached
zcp instance ssh my-vm --use-private   # over the VPC or VPN
zcp instance ssh my-vm --user ubuntu
```

## Output fixes

- `firewall list` shows each rule's state. The column was blank because the API
  reports the state only inside a nested object. Contributed by @cokerrd.
- `dns show` prints `-` for status instead of a fabricated `false`. The show
  endpoint does not return a status field. `dns list` still shows `true` or
  `false`.
- `autoscale policy delete` and `autoscale condition delete` print the numeric
  ID in their not-found message instead of a quoted character.

## From v0.0.27

**`instance create` supports VPC and existing networks.** `--network-type`
accepts `Isolated` (the default), `L2` and `Vpc`. Use `--vr-plan` to build a VPC
network from a virtual router plan. Use `--networks` to attach networks you
already have, and add `--default-network` when you attach more than one.
`--network-plan` is required only for `Isolated` and `L2` when `--networks` is
omitted. Contributed by @cokerrd.

```bash
zcp instance create --name my-vpc-vm --project default-9 --region yul-1 \
  --template ubuntu-2604-lts-1 --plan ca2sl --billing-cycle hourly \
  --network-type Vpc --vr-plan <router-plan> --storage-category pro-nvme
```

**`ip static-nat enable` needs a network.** The API rejects a static NAT request
without one, so the command failed every time. It now takes `--network`
alongside `--instance`. Scripts calling it without `--network` need updating.
Contributed by @cokerrd.

```bash
zcp ip static-nat enable 1036521143 --instance my-vm --network my-network
```

**`volume attach` and `volume detach` say what happened.** Both printed a row of
empty fields. They now show the API's status next to the slugs you passed.
Contributed by @cokerrd.

## For Terraform provider maintainers

The provider consumes `pkg/api` as a library. Signatures did not change, but
behaviour did. `vmbackup.Service.Delete` now posts a cancellation request
instead of `DELETE`. Success means the request was accepted, not that the
schedule is gone. A missing slug returns a 403 instead of a 404. The backup and
VM backup listings now walk every page, and both backup types decode the
scheduled hour from a number or a string. The changelog lists the new helper
methods.

## Known limitation

**`object-storage bucket encryption enable` is disabled.** The region's Ceph
RADOS Gateway has no encryption key backend configured. Enabling SSE-S3
default encryption makes every subsequent upload to the bucket fail until
encryption is disabled again. The command now refuses to run and explains
why. `status` and `disable` still work, so you can check or clear an
existing setting. See #54.

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
zcp version   # zcp version v0.0.28
```

First-time setup after installing:

```bash
zcp profile add default --region yul-1 --project default-9   # prompts for bearer token
zcp auth validate
```
