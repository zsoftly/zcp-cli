# zcp v0.0.25 Release Notes

Hotfix: `zcp dns record-create` can now create `MX` records.

## `dns record-create` now supports MX records

Creating an `MX` record with `zcp dns record-create` used to fail with a 403. The
command never sent the record's priority, which the API requires for `MX`. (The
Web UI worked because it sends `priority` as its own field.) The command now takes
a `--priority` flag: put the mail server in `--content` and the preference number
in `--priority`.

```bash
zcp dns record-create --domain example-com-1 --name @ --type MX --content mail.example.com. --priority 10
```

`--priority` is required for `MX` and must be between 0 and 65535. Leaving it off
stops with a clear error instead of a server-side 403. A `0` preference is sent
correctly. Other record types (A, AAAA, CNAME, TXT) are unchanged and do not take
a priority.

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
zcp version   # zcp version v0.0.25
```

First-time setup after installing:

```bash
zcp profile add default --region yul-1 --project default-9   # prompts for bearer token
zcp auth validate
```
