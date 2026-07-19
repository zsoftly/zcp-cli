# zcp v0.0.26 Release Notes

Bug fixes for port forwarding, firewall, and SSH key commands.

## `portforward list` shows the ports again

The public and private port columns in `zcp portforward list` were blank. The
response was decoded from the wrong field names, so the ports never populated.
They now show correctly. The rules themselves were always fine, so no action is
needed beyond upgrading.

## `portforward create` and `firewall create` report clearly

Both commands used to print a table of empty fields after a successful create.
The API accepts these requests asynchronously and returns no rule object, so
there was nothing to show yet. They now confirm the request was accepted and
point you to the matching `list` command.

```bash
zcp portforward create --ip <slug> --protocol tcp \
  --public-port 22 --public-end-port 22 --private-port 22 --private-end-port 22 \
  --instance my-vm
# Port forwarding rule creation accepted. Run 'zcp portforward list --ip <slug>' to confirm.
```

## `ssh-key delete` accepts the ID, name, or slug

`zcp ssh-key delete` only worked with the key's slug. The ID (UUID) shown by
`zcp ssh-key list` was rejected as not found. You can now pass any of the ID,
name, or slug, and the command resolves it to the slug before deleting. Deleting
a key that is already gone is a no-op instead of an error.

```bash
zcp ssh-key delete my-key          # slug
zcp ssh-key delete a24bee1f-...    # ID from 'ssh-key list' now works too
```

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
zcp version   # zcp version v0.0.26
```

First-time setup after installing:

```bash
zcp profile add default --region yul-1 --project default-9   # prompts for bearer token
zcp auth validate
```
