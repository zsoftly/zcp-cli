# zcp v0.0.22 Release Notes

## DNS records now display and delete correctly

The live DNS backend (PowerDNS) models records as record **sets** addressed by
name and type — it exposes no record IDs, and returns values in a `contents`
array. The CLI previously decoded neither, so record tables printed blank ID
and CONTENT columns, and `dns record-delete` demanded a numeric `--record-id`
that does not exist anywhere. Record deletion was impossible.

This release aligns the CLI with how the backend actually works, verified live
end to end (create → show → delete → confirm gone).

Highlights:

- **Record content is visible again.** `zcp dns show` and `record-create`
  tables show real values (multi-value sets joined, e.g.
  `ns1.zsoftly.ca., ns2.zsoftly.ca.`), and the dead ID column is gone.
- **`dns record-delete` works — by name and type.**
- **Record names are relative.** The backend appends the zone; the help text
  now says so (passing an FQDN used to silently create
  `www.example.com.example.com.`).
- **`egress create` retries its lookup and reports honestly** when the backend
  silently drops an accepted rule (a platform-side issue found while testing).
- **`docs/commands.md` is now machine-validated:** all 264 examples checked
  against the built CLI — six sections documented commands that did not exist
  and are rewritten to the real trees.

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
zcp version   # zcp version v0.0.22
```

First-time setup after installing:

```bash
zcp profile add default --region yul-1 --project default-9   # prompts for bearer token
zcp auth validate
```

---

## Fixed

### DNS record display and deletion

```bash
# Records show their content; sets are addressed by NAME + TYPE (no IDs)
zcp dns show example-com
# NAME               TYPE  CONTENT                           TTL
# www.example.com.   A     192.0.2.50                        3600
# example.com.       NS    ns1.zsoftly.ca., ns2.zsoftly.ca.  3600

# Create with a RELATIVE name — the zone is appended by the backend
zcp dns record-create --domain example-com --name www --type A --content 192.0.2.50

# Delete by name and type (relative or fully qualified both work)
zcp dns record-delete --domain example-com --name www --type A
```

The legacy `--record-id` flag remains for deployments whose DNS backend exposes
record IDs. SDK consumers get `DeleteRecordByName`, `CanonicalRecordFQDN`, and
`Record.Contents`; the ID-based `DeleteRecord` is deprecated.

### Egress rule creation reporting

The create endpoint returns no body, so the CLI resolves the new rule from the
rule list. It now retries that lookup (3 attempts over ~4s) before giving up,
and when the rule never appears — the API can return 200 yet create nothing on
some networks — the error says the backend may have dropped the rule, pointing
at the platform rather than the CLI.

### Command reference corrected and machine-validated

Six sections of `docs/commands.md` documented commands that do not exist
(`monitoring create`, `vpn create --vpc`, `support close`, `dashboard status`,
among others) or missed required flags (`ip allocate` without `--plan`/
`--billing-cycle`). All are rewritten to the real command trees — including
the previously undocumented `kubernetes scale/get-config/upgrade-version/delete`
and `loadbalancer attach-vm/detach-vm/delete-rule` — and every example in the
reference is now validated automatically against the CLI (command paths and
flags; 264 examples).
