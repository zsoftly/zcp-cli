# zcp v0.0.17 Release Notes

## Clearer errors, and no backend technology in output

This release is a CLI-usability and information-hygiene pass:

- **Actionable argument errors** across every command that takes positional arguments
  (128 subcommands) — what's missing, the usage line, and the command's own examples.
- **Unknown subcommands now error** (non-zero exit) instead of silently printing help.
- **Backend technology names are no longer exposed** in any command output.
- **`--cloud-provider` is auto-detected** and saved to your profile — you no longer pass it.

---

## Added

### `object-storage object download`

Objects can now be downloaded to a local file over the S3 protocol — previously the CLI
could upload, list, delete, and show metadata, but not fetch an object's contents.

```bash
zcp object-storage object download my-store my-bucket report.pdf            # → ./report.pdf
zcp object-storage object download my-store my-bucket images/logo.png --dest ./logo.png
```

`--dest` accepts a file path or a directory (writes the object's base name into it); it
defaults to the base name in the current directory.

### Object versioning and raw bucket policies

- `zcp object-storage bucket versioning enable|suspend|status` — S3 object versioning.
- `zcp object-storage bucket policy get|set|delete` — read/set/remove a bucket's raw S3
  policy (`set --file policy.json`, or `--file -` for stdin) for fine-grained access.

Both verified live against Ceph RGW.

### Tags, encryption, lifecycle, and a versioned-bucket fix

- `bucket tag` / `object tag` — set/get/delete tags (`--tag key=value`).
- `bucket encryption status|enable|disable` — default SSE-S3 encryption.
- `bucket lifecycle expire --days N [--prefix P]` (plus `get`/`delete`) — auto-expire objects.
- `bucket empty` and `bucket delete --purge` — remove all objects **and versions**. This
  fixes a real gap: a bucket that ever had versioning enabled couldn't be deleted, because
  its object versions/delete-markers blocked the REST delete.

- `bucket cors set --origin --method [--header --max-age]` (plus `get`/`delete`) —
  cross-origin rules for browser apps.

RGW support for each was verified by probing the live endpoint with the S3 client before
building.

### Versioning workflows, copy/move, stat, presigned upload, multipart cleanup

The object-storage S3 surface is now feature-complete (everything Ceph RGW supports
except object-lock, which needs a backend change at bucket creation). All of these
operations are **CLI-only** — they talk directly to the Ceph RADOS Gateway over the
S3 protocol and are not yet available via the ZCP REST API or the Web UI (only
instance and basic bucket CRUD are REST-backed and mirrored in the Web UI):

- **Versioning is usable:** `object versions`, `object download/delete --version-id`,
  and `object restore` (undelete).
- **`object copy` / `object move`** — server-side, no round-trip.
- **`object stat`** (full S3 metadata via HEAD) and `object put --metadata key=value`.
- **`object put-url`** — pre-signed upload URL (`curl -T file "<url>"`).
- **`bucket uploads list|abort`** — reclaim storage from failed large uploads.
- **`bucket lifecycle expire --noncurrent-days / --abort-multipart-days`** — expire old
  versions and clean up stalled uploads.
- `policy/lifecycle/cors get` now honor `-o yaml`.

### Public buckets, shareable object URLs, and plan discovery

- `zcp object-storage bucket set-acl --acl public-read` makes a bucket's objects
  anonymously downloadable (via an S3 bucket policy); `--acl private` reverts it.
- `zcp object-storage object url <slug> <bucket> <key> [--expires 24h]` mints a
  pre-signed, time-limited link a client can use without credentials — even on a
  private bucket (max 7 days).
- `zcp plan object-storage` lists the Object Storage plan slugs for `create --plan`.

These were verified end-to-end against live Ceph storage in YOW (create → bucket →
upload → make public → anonymous download → pre-signed URL → make private → delete).

### Object-storage create is simpler

`object-storage create --plan <slug>` no longer needs `--storage-category` — the CLI
derives it from the plan (a mismatch previously failed with `Invalid Storage Category`).
Use an object-storage region (`os-yul`/`os-yow`) and a plan from `zcp plan object-storage`.

## Changed

### Helpful errors instead of `accepts 1 arg(s), received 0`

Before:

```
❯ zcp profile add
Error: accepts 1 arg(s), received 0
```

After:

```
❯ zcp profile add
Error: missing required argument: <name>

Usage:
  zcp profile add <name> [flags]

Examples:
  zcp profile add default
  zcp profile add prod --bearer-token <token>
```

Multi-argument commands name each missing placeholder:

```
❯ zcp acl create-rule
Error: missing required arguments: <vpc-slug>, <acl-name-or-id>

Usage:
  zcp acl create-rule <vpc-slug> <acl-name-or-id> [flags]

Examples:
  zcp acl create-rule my-vpc web-acl --number 1 --protocol tcp --start-port 80 --end-port 80 --cidr 0.0.0.0/0
  ...
```

Supplying too many arguments now reports `too many arguments: expected N, got M` with the
same usage/examples block.

This is purely a messaging change: which commands accept how many arguments is unchanged,
and all existing behavior on the success path is identical.

### Unknown subcommands error instead of silently printing help

Running a command group with a bad subcommand used to print that group's help and exit `0`,
hiding the typo and letting scripts treat a mistake as success. It now errors to stderr and
exits non-zero, with a message that adapts to the group. A group with one subcommand points
straight at it:

```
❯ zcp region lists
Error: unknown subcommand "lists" for "zcp region"

Run this instead:
  zcp region list    List available regions

Example:
  zcp region list
```

A group with several lists the valid subcommands and suggests the closest match:

```
❯ zcp profile shw
Error: unknown subcommand "shw" for "zcp profile"

Did you mean this?
	show

Available commands:
  add             Add or update a profile
  ...

Run 'zcp profile --help' for usage and examples.
```

Running a group with no arguments still prints help and exits `0`.

### `--cloud-provider` is auto-detected

Customers no longer pass a cloud provider. `zcp auth validate` (and `zcp profile add`)
detect the account's compute provider — the one whose catalog includes "Virtual
Machine" (`nimbo` in production) — and save it to the profile; every create command
reads it automatically. Object storage and DNS default to their own providers (`ceph`
and `dns`) automatically. The flag is hidden from help but remains available as an
override (along with `ZCP_CLOUD_PROVIDER`).

```
❯ zcp auth validate
Credentials are valid.
Cloud provider detected and saved to profile "default": nimbo
```

If it can't be determined, create commands say so with guidance instead of the old
terse "--cloud-provider is required".

All three provider values were confirmed against the live API: each region maps to
exactly one provider (`yow-1`/`yul-1`→`nimbo`, `default`→`dns`, `os-yow`/`os-yul`→`ceph`),
and real existing instances/volumes store `cloud_provider: nimbo`. As part of this,
`zcp dns create` now defaults `--region` to `default` (the DNS provider's only region)
so it is fully hands-off, and object-storage examples use object-storage regions
(`os-yul`/`os-yow`) — the previous `yow-1`/`yul-1` DNS and object-storage examples were
wrong and would have failed.

## Removed

### Backend technology is no longer exposed in output

Display-only columns that revealed the underlying platform ("Cloud Stack", "Ceph", "Dns")
have been removed:

- `region list` — dropped `PROVIDER` and `COMING SOON`
- `backup list`, `snapshot list`, `vm-backup list` — dropped `SERVICE`
- `instance get` — dropped the `Service` row
- `cloud-provider list` — dropped `DISPLAY NAME`
- `dns list` / `dns show` / `dns create` — dropped the `DNS PROVIDER` column/row, and hid the `--dns-provider` flag (which named the backend, e.g. `powerdns`)

These were informational only. Resource creation uses the provider/region **slug**, which is
retained, so no workflow changes. Billing/dashboard/project "SERVICE" columns are untouched —
they name billing categories (e.g. "Virtual Machine"), not backend technology.

## Internal

- New `internal/commands/args.go` provides drop-in `exactArgs` / `minArgs` / `maxArgs` /
  `rangeArgs` validators that replace `cobra.ExactArgs` / `cobra.MaximumNArgs` package-wide.
  Missing-argument names are derived from each command's `Use` line and examples from its
  `Example` field, so no per-command wiring is needed and new commands get the behavior for
  free by using the helpers.
- `EnforceSubcommandErrors` (same file) walks the command tree once — called from `root.go`
  after all subcommands are registered — and installs the unknown-subcommand handler on every
  command group, so the behavior applies uniformly and to future groups automatically.
