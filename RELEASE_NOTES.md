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
