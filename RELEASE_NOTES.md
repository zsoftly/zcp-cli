# zcp 0.0.11 Release Notes

## Bug Fixes

### SSH key import sends the wrong field

`zcp ssh-key import` was sending `cloud_provider` in the request body. The API (and portal) expect `region`. The `--cloud-provider` flag has been replaced with `--region`:

```bash
# before (broken — key created without region binding)
zcp ssh-key import --name mykey --public-key "ssh-ed25519 ..." --cloud-provider nimbo

# after (correct)
zcp ssh-key import --name mykey --public-key "ssh-ed25519 ..." --region yul-1
```

---

### `instance get` / `instance list` — Private IP always blank

The top-level `private_ip` API field is `null` for CloudStack-backed VMs; the real value is in `networks[].pivot.ipaddress`. Both commands now resolve the private IP from the network pivot, preferring the default network attachment before falling back to the first available network.

---

### `instance get` — transient 403 after VM creation

Immediately after creation, the CMP routing layer hasn't yet indexed the new VM slug, and `instance get` returns a 403:

```
The route virtual-machines/<slug> could not be found.
```

`instance get` now recognises this as a transient condition and retries up to 5 times with exponential backoff (2 s → 4 s → 8 s → 16 s) before surfacing the error. All retry attempts emit a message to stderr so you can see progress:

```
VM routing not ready yet, retrying in 2s...
VM routing not ready yet, retrying in 4s...
```

---

### `dns record-delete` — record ID printed with wrong format

The "already deleted" log message was formatting the integer record ID with `%q` (string-quote verb), producing output like `DNS record '42' not found`. Corrected to `%d`.

---

## New

### `instance create` — user-data support

Pass a cloud-init / user-data script at VM creation time:

```bash
# inline script
zcp instance create ... --user-data "#!/bin/bash\napt-get update"

# from a file
zcp instance create ... --user-data-file ./cloud-init.yaml
```

`--user-data` and `--user-data-file` are mutually exclusive — passing both returns an error immediately.

---

## Changed

### `instance create --blockstorage-plan` is now optional

The flag was previously marked required, but the backend auto-assigns a block storage plan when the field is omitted (consistent with portal behaviour). Existing scripts that pass the flag are unaffected.

### `instance create --network-plan` — corrected example values

Help text examples updated from `inet-yow` / `inet-yul` to `pnet-yow` / `pnet-yul` to match the actual network plan slugs.

### `template account-delete --yes` gains `-f` shorthand

`-y` is reserved globally for `--auto-approve`. The confirmation-skip flag on destructive template commands now uses `-f` (force).

### `volume create --size 0` gives a clearer error

Explicitly passing `--size 0` now returns `--size must be > 0` instead of the misleading `--plan or --size is required`. `--plan` and `--size` are mutually exclusive even when `--size 0` is supplied.

---

## Installation

**macOS / Linux / WSL:**

```bash
curl -fsSL https://github.com/zsoftly/zcp-cli/releases/latest/download/install.sh | bash
```

**Windows (PowerShell):**

```powershell
irm https://github.com/zsoftly/zcp-cli/releases/latest/download/install.ps1 | iex
```

**Verify:**

```bash
zcp version
# zcp version 0.0.11
```

---

## Full Changelog

https://github.com/zsoftly/zcp-cli/compare/0.0.10...0.0.11
