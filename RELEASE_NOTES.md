# zcp v0.0.29 Release Notes

Volume listings now retrieve every page returned by the API. `zcp volume list`
and Go library consumers receive the complete collection instead of only the
first page.

## Fixed

**Pagination failures now stop with an error.** If the API ignores a requested
page, omits pagination metadata on a later page, or reports an incomplete page,
the CLI returns an error instead of silently returning duplicate or partial
volume results.

```bash
zcp volume list
```

**VPC subnet limit failures now explain how to proceed.** The platform allows
eight subnets per VPC by default. When the platform rejects `zcp network create
--vpc` because the VPC has reached that limit, the CLI directs you to open a
support ticket. Rerun the command after the platform applies the quota increase.

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
zcp version   # zcp version v0.0.29
```

First-time setup after installing:

```bash
zcp profile add default --region yul-1 --project default-9   # prompts for bearer token
zcp auth validate
```
