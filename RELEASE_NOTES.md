# zcp v0.0.20 Release Notes

## Deleting a VM now releases its auto-assigned public IP

When a VM is created public, the CMP auto-assigns it a 1:1 public IP. Until now,
deleting the VM left that IP `Allocated` — orphaned and still billing — until someone
released it by hand. `zcp instance delete` now releases it as part of the delete, matching
the "Delete auto-assigned public IPs when deleting VM" option in the portal.

Highlights:

- **`zcp instance delete` releases the VM's auto-assigned public IP by default** (sends
  `delete_public_ip=true`).
- **Manually-acquired and source-NAT IPs are untouched** — they release only when their
  network/IP is removed.
- **Opt out with `--delete-public-ip=false`** to keep the IP (e.g. when it's reused by NAT, a
  load balancer, or a shared network).

---

## Added

### `zcp instance delete` releases the auto-assigned public IP

```bash
zcp instance delete my-vm --yes                       # VM gone, its auto-assigned IP released
zcp instance delete my-vm --force --yes               # also expunge from the hypervisor immediately
zcp instance delete my-vm --delete-public-ip=false    # VM gone, keep the IP allocated
```

`--delete-public-ip` defaults to **true**. It only releases public IPs that the CMP assigned
when the VM was created — manually-acquired IPs and the network's source-NAT IP are never
touched by this flag (those release when you delete the network/IP). The interactive
confirmation prompt now says when the IP will be released:

```
WARNING: Delete "my-vm" is permanent and cannot be undone. Its auto-assigned public IP will also be released. [y/N]:
```

## Changed

- **Behavior change:** `zcp instance delete` used to leave the VM's auto-assigned public IP
  allocated (and billing) after the VM was deleted. It is now released by default. Add
  `--delete-public-ip=false` to keep the old behavior.

## Upgrade notes

If you have scripts or automation that delete VMs and then **reuse** the freed public IP (for a
NAT rule, load balancer, or shared network), pass `--delete-public-ip=false` so the IP stays
allocated. Otherwise no changes are needed — the new default cleans up the leaked IPs you'd
previously have had to release manually with `zcp ip release`.

---

## Installation

### Linux / macOS / WSL (one-liner)

```bash
curl -fsSL https://github.com/zsoftly/zcp-cli/releases/latest/download/install.sh | bash
```

Installs `zcp` to `/usr/local/bin` (you may be prompted for `sudo`). Set `INSTALL_DIR` to
choose another location, e.g. `INSTALL_DIR="$HOME/.local/bin"`.

### Windows (PowerShell)

```powershell
irm https://github.com/zsoftly/zcp-cli/releases/latest/download/install.ps1 | iex
```

Installs `zcp.exe` to `%LOCALAPPDATA%\Programs\zcp`.

### Manual download

Grab the binary for your platform from the
[Releases page](https://github.com/zsoftly/zcp-cli/releases), make it executable, and put it on
your `PATH`.

| OS      | Arch          | Asset                   |
| ------- | ------------- | ----------------------- |
| Linux   | x86_64        | `zcp-linux-amd64`       |
| Linux   | ARM64         | `zcp-linux-arm64`       |
| macOS   | Intel         | `zcp-darwin-amd64`      |
| macOS   | Apple Silicon | `zcp-darwin-arm64`      |
| Windows | x86_64        | `zcp-windows-amd64.exe` |
| Windows | ARM64         | `zcp-windows-arm64.exe` |

```bash
# Linux amd64 example
curl -Lo zcp https://github.com/zsoftly/zcp-cli/releases/latest/download/zcp-linux-amd64
chmod +x zcp
sudo mv zcp /usr/local/bin/zcp
```

```powershell
# Windows amd64 example (PowerShell)
irm https://github.com/zsoftly/zcp-cli/releases/latest/download/zcp-windows-amd64.exe -OutFile zcp.exe
# then move zcp.exe to a directory on your PATH
```

### Verify

```bash
zcp version
zcp --help
```
