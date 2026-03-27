# Changelog

All notable changes to zcp will be documented in this file.

Format based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), using
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.0.4] - 2026-03-27

### Added

- **`vpc create-network`**: Create VPC tier networks via `/restapi/vpc/createVpcNetwork`
- **`vpc update-network`**: Update VPC tier networks via `/restapi/vpc/updateVpcNetwork`

**Full Changelog**: https://github.com/zsoftly/zcp-cli/compare/0.0.3...0.0.4

---

## [0.0.3] - 2026-03-27

### Added

- **`host list` command**: Lists hypervisor hosts with CPU cores, VM count, and status
- **`resource quota` subcommand**: Shows resource quota limits with unit/used/available/maximum
- **`vpc create-network`**: Create VPC tier networks via `/restapi/vpc/createVpcNetwork`
- **`vpc update-network`**: Update VPC tier networks via `/restapi/vpc/updateVpcNetwork`
- **Network VPC tier flags**: `--vpc`, `--gateway`, `--netmask`, `--acl` on `network create`
- **Integration test suite**: Full lifecycle tests (SSH key, security group, instance,
  volume, snapshot, stop/start/destroy) plus parallel smoke tests across 10 resource types

### Fixed

- **JSON unmarshal errors**: `volume.createdTimeStamp` (string to int64),
  `host.cpuCores`/`vmCount` (int to string), `kubernetes.minMemory`/`minCpuNumber`
  (string to int)
- **VPC/network CIDR field**: Corrected from `getcIDR` to `cIDR` (API spec was wrong)
- **VPC create**: `description` and `publicLoadBalancerProvider` now sent as required fields
- **`network.IsPublic`**: Moved from JSON body to query parameter per API spec
- **`snapshot-policy list`**: Now requires `--volume` flag (matches API requirement)

**Full Changelog**: https://github.com/zsoftly/zcp-cli/compare/0.0.2...0.0.3

---

## [0.0.2] - 2026-03-23

### Added

- **Default zone per profile**: Set once, never type `--zone` again
  - `zcp zone use <uuid>` — saves default zone to the active profile
  - `zcp zone use ""` — clears the default
  - All commands fall back to the profile default; explicit `--zone` still overrides

### Changed

- **`buildClientAndPrinter` returns `*config.Profile`**: Commands can now read any
  profile-level default, not just credentials
- **Actionable zone errors**: Missing zone now says `--zone is required (or set a default: zcp zone use <uuid>)`

### Fixed

- **CI: Windows config tests**: `TestSaveAndLoad` and `TestConfigFilePath` now use `APPDATA`
  on Windows instead of `XDG_CONFIG_HOME`
- **CI: Windows permission check**: `0600` assertion skipped on Windows (no Unix-style permissions)
- **CI: PSScriptAnalyzer**: Split shell and PowerShell lint into separate jobs; shell on
  ubuntu, PowerShell on windows-latest where PSGallery works reliably
- **CI: archive packaging**: Fixed `tar -C "$dir" *` glob expanding in wrong directory

### Removed

- **Dead `GlobalFlags` type**: `config.GlobalFlags` struct and `root.GlobalFlags()` function
  removed — neither was referenced after the profile-return refactor

**Full Changelog**: https://github.com/zsoftly/zcp-cli/compare/0.0.1...0.0.2

---

## [0.0.1] - 2026-03-15

### Added

- **Full ZCP Cloud Platform CLI** — 28 top-level commands covering compute, storage,
  networking, security, Kubernetes, and billing:
  - `instance` — list, get, create, start, stop, delete, reboot, resize, rename, recover, ssh
  - `volume` — list, create, attach, detach, delete, resize
  - `network` — list, get, create, update, delete, restart
  - `vpc` — list, get, create, update, delete, restart
  - `snapshot` / `vmsnapshot` / `snapshotpolicy` — full lifecycle
  - `firewall` / `egress` / `portforward` — rule management
  - `loadbalancer` / `internallb` — load balancer management
  - `acl` — network ACL management
  - `vpn` — gateway, customer gateway, connection, user management
  - `sshkey` — create, list, delete
  - `securitygroup` — full rule management
  - `kubernetes` — cluster lifecycle and node management
  - `offering` — compute, storage, network, VPC offering catalog
  - `template` — VM template listing
  - `tag` — resource tag management
  - `zone` — availability zone listing
  - `auth` — credential validation
  - `profile` — add, list, show, use, update, rename, delete
  - `usage` / `cost` — consumption and billing
  - `resource` — quota and availability
  - `admin` — host listing
- **Profile-based credentials**: `~/.config/zcp/config.yaml` (0600 permissions);
  multiple named profiles with `--profile` override
- **`--wait` flag**: Polls async operations to completion on instance start/stop/create
  and volume create
- **`instance ssh`**: Resolves instance IP and execs SSH with stdin/stdout passthrough
- **Shell completions**: bash, zsh, fish, PowerShell via `zcp completion <shell>`
- **`--pager` flag**: Pipes table output through `$PAGER` (default `less -FRX`) on TTY
- **Multi-platform binaries**: Linux, macOS, Windows — amd64 and arm64
- **One-liner installers**: `install.sh` (Unix) and `install.ps1` (Windows)

### Security

- **Config file permissions**: Credentials stored at 0600; config directory at 0700
- **No credentials in CLI flags**: API keys read from config file only
- **Retry safety**: Only GET requests are retried (idempotent); POST/PUT/DELETE are not

### Technical

- Go module `github.com/zsoftly/zcp-cli`, binary `zcp`
- Cobra CLI framework with persistent global flags
- Typed service layer per API domain (`internal/api/*`)
- Exponential backoff retry on GET requests (1s/2s/4s, max 3 retries, 5xx/429/network)
- Async job polling via `waiters.Waiter` (`GET /restapi/asyncjob/resourceStatus`)
- GitHub Actions CI: test, security scan, cross-platform build, GitHub Release

**Full Changelog**: https://github.com/zsoftly/zcp-cli/commits/0.0.1
