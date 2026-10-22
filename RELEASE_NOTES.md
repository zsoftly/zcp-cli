# zcp v1.1.0 Release Notes

zcp v1.1.0 adds scheduled snapshots for block storage volumes. Create a policy
for a volume, then list, inspect, update, pause, resume, run, or delete it from
the CLI. The examples use the region and project in your active profile; replace
each quoted placeholder with a value from your account.

## Added

`zcp snapshot schedule` supports hourly, daily, weekly, and monthly policies.
Schedules use the selected region and project, an IANA timezone, a whole-hour
time, and a retention count. `--immediate` requests the first snapshot when the
policy is created. `run-now` requests another snapshot without waiting for its
next scheduled time.

Creation first confirms that the volume is in the selected region and project.
Later actions confirm that the policy is a block-storage snapshot policy in that
same region and project.

```bash
zcp snapshot schedule create "<volume-slug>" \
  --name nightly \
  --interval dailyAt \
  --at 01:00 \
  --timezone America/Toronto \
  --retention 7 \
  --immediate

zcp snapshot schedule list
zcp snapshot schedule update "<schedule-id>" \
  --interval weeklyOn --day 0 --at 02:00 --timezone UTC --retention 4
zcp snapshot schedule pause "<schedule-id>"
zcp snapshot schedule resume "<schedule-id>"
zcp snapshot schedule run-now "<schedule-id>" --yes
```

For `weeklyOn`, use weekday `0` through `6`. For `monthlyOn`, use day `1`
through `28`. `--at` accepts whole hours in `HH:00` format. Use `pause` to keep
existing snapshots while stopping future runs. The platform rejects deletion of
a schedule while its snapshots remain, so remove those snapshots before running
`zcp snapshot schedule delete "<schedule-id>" --yes`.

## Fixed

### DNS record operations

`zcp dns record-create` prints only the value it created. Use `--show-zone` on
record creation or deletion to print the resulting zone. If the API accepts a
create request but the CLI cannot verify the new value, the command returns an
error and tells you to inspect the zone with `zcp dns show "<domain-slug>"`
before retrying. It does not retry the create request.

`zcp dns record-delete` normally prints a labelled pre-delete snapshot. Table
output adds a `SOURCE` column. JSON and YAML set `source` to
`pre_delete_snapshot`. This is a read taken before deletion, not a guarantee of
which values were removed. A warning explains that concurrent changes can make
the snapshot differ from the deleted values. It deletes the values from its
initial snapshot one by one. If a deletion fails, it stops and reports partial
progress. Values added after the read are left alone. Deleting an already absent
record is idempotent.

CAA values must quote their value:

```bash
zcp dns record-create --domain "<domain-slug>" --name @ --type CAA \
  --content '0 issue "letsencrypt.org"'
```

### Build security

The release build uses Go 1.26.10, which fixes the reachable
certificate-verification vulnerability reported by `govulncheck`.

## Installation and upgrade

The install scripts install the latest release and upgrade an existing
installation in place.

**Linux / macOS**

```bash
curl -fsSL https://github.com/zsoftly/zcp-cli/releases/latest/download/install.sh | bash
```

**Windows (PowerShell)**

```powershell
irm https://github.com/zsoftly/zcp-cli/releases/latest/download/install.ps1 | iex
```

**Manual download:** download the binary or archive for your platform from the
[Releases](https://github.com/zsoftly/zcp-cli/releases) page. Extract an archive
first. Rename the platform binary to `zcp` (or `zcp.exe` on Windows). On Linux
or macOS, run `chmod +x zcp`, then place it on your `PATH`.

**Verify:**

```bash
zcp version   # zcp version v1.1.0
```
