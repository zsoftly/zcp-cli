# zcp v0.0.21 Release Notes

## Profile defaults now work everywhere — including create commands

`zcp profile add default --region yul-1 --project default-9` stores your default scope
so you never repeat `--region`/`--project`. Until now that promise only held for
list/get commands: create and mutate commands (e.g. `network create`, `instance create`)
resolved their own scope from flags and environment variables only, so a fully
configured user still hit `--region is required`. That gap is closed — configure once,
and every command picks the defaults up.

Highlights:

- **Profile default region/project are honored by create/mutate commands.** The root
  scope gate now injects the resolved scope (flag > env > profile default, respecting
  `--profile`) onto the command's flags. Verified end-to-end against the production API.
- **First-run setup points at the production defaults.** The installers print
  copy-paste setup for `zcp profile add default --region yul-1 --project default-9` —
  every account's initial project is `default-9` (like `us-east-1` on AWS).
- **All command examples use slugs verified against the live production catalog** —
  broken template, backup, and virtual-router plan slugs are fixed.

---

## Fixed

### Profile default region/project honored by create/mutate commands

```bash
zcp profile add default --region yul-1 --project default-9
zcp auth validate

# Previously: Error: --region is required
# Now: creates the network in yul-1 / default-9 from your profile defaults
zcp network create --name my-net --network-plan inet-yul --billing-cycle hourly
```

Explicit `--region`/`--project` flags and `ZCP_REGION`/`ZCP_PROJECT` still take
precedence over the profile default, and `--profile <name>` selects which profile's
defaults apply. Two command groups manage their own scope by design and are
unaffected: `dns create` (fixed `default` region; still needs an explicit
`--project`) and `object-storage create/list` (object-storage `os-*` regions).

### First-run examples point at the production defaults

The Unix and Windows installers now end with copy-paste setup commands
(`zcp profile add default --region yul-1 --project default-9`, `zcp auth validate`)
plus matching `ZCP_REGION`/`ZCP_PROJECT` examples for scripts. README,
configuration docs, and command examples consistently use `yul-1` as the primary
compute region and YUL-compatible plan slugs.

### Command examples verified against the live production catalog

Examples that referenced nonexistent slugs are fixed:

| Was                         | Now                           | Where                                                                                      |
| --------------------------- | ----------------------------- | ------------------------------------------------------------------------------------------ |
| `ubuntu-2604-lts`           | `ubuntu-2604-lts-1`           | instance/autoscale `--template` (template slugs are region-specific; this is yul-1's)      |
| `backup-1`, `backup-basic`  | `backup-yul`                  | `backup create`, `vm-backup create` `--plan` (backup plans are now enabled in the catalog) |
| `virtual-private-cloud-vpc` | `virtual-private-cloud-vpc-1` | `virtual-router create --plan`                                                             |

The docs Backup section was rewritten to show the real `backup create` flags
(`--volume`/`--interval`/`--plan` …) and to drop nonexistent `backup get`/`backup
restore` subcommands. Verified live in yul-1: `ca2sl`/`ca2sm`/`ca2sxs`, `b2g1`,
`pro-nvme`, `inet-yul`, `l2net-yul`, `k8s-la-yul-1`, `k8s-xla-yul-1`,
`vm-snapshot-yul`, `ipv4-yul`, `lb-yul`, `backup-yul`, `virtual-private-cloud-vpc-1`,
`ubuntu-2604-lts-1`.

## Upgrade notes

No breaking changes. If you have profile defaults configured (`zcp profile add`
with `--region`/`--project`), create/mutate commands that previously errored
without explicit flags now use those defaults automatically — pass `--region`/
`--project` (or set `ZCP_REGION`/`ZCP_PROJECT`) to override per invocation.

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
