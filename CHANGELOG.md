# Changelog

All notable changes to zcp will be documented in this file.

Format based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), using
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.0.8] - 2026-04-09

### Fixed

- **VPC create**: Correct payload structure — `cidr` is the network address (e.g. `10.1.0.1`), `size` is the mask (e.g. `16`), requires `type=Vpc`, `billing_cycle`, `plan` (from router plans), `storage_category`
- **ACL create**: Fixed to create ACL lists (name, description, vpc) instead of incorrectly sending protocol/port rule fields
- **Volume Size type**: Fixed `string` to `interface{}` — API returns number, not string
- **JSON tags**: Fixed camelCase to snake_case for `cloud_provider` across vpc, vpn, autoscale request structs
- **VPN user create**: Updated to accept `UserCreateRequest` struct with cloud_provider, region, project

### Added

- **VPC tier/subnet creation**: Confirmed working via `POST /networks` with `type=Vpc`, `gateway`, `netmask`, `acl_id`
- **`--cloud-provider`, `--region`, `--project` flags**: Added to network, vpc, virtualrouter, dns, vpn, autoscale create commands
- **`docs/roadmap.md`**: Feature roadmap documenting what works, what's coming, and what's blocked on platform

### Changed

- **VPC create flags**: Replaced old `--zone`, `--offering`, `--network-domain`, `--lb-provider` with `--cidr`, `--size`, `--plan`, `--billing-cycle`, `--storage-category`, `--cloud-provider`, `--region`, `--project`
- **ACL commands**: `zcp acl create` and `zcp vpc acl-create-rule` now take `--name` and `--description` (matching the actual API)

**Full Changelog**: https://github.com/zsoftly/zcp-cli/compare/0.0.7...0.0.8

---

## [0.0.7] - 2026-04-08

### Added

- **`region list`**: List available regions (replaces dead `zone list`)
- **`profile-info`**: User profile management (get, update, company details, time settings, API enable/disable, login activity, activity logs — 2FA status is shown via `get` but not managed)
- **`vm-backup`**: VM backup operations (list, create)
- **`cloud-provider list`**: List available cloud providers
- **`server list`**: List available servers
- **`currency list`**: List available currencies
- **`billing-cycle list`**: List available billing cycles
- **`storage-category list`**: List available storage categories

### Removed

- **Dead STKBILL commands**: `zone`, `offering`, `resource`, `host`, `cost`, `usage`, `internal-lb`, `snapshot-policy`, `security-group`, `tag`, `admin` — all pointed at old `/restapi/` endpoints that no longer exist
- **Dead API packages**: `zone`, `offering`, `resource`, `host`, `cost`, `usage`, `internallb`, `snapshotpolicy`, `securitygroup`, `tags`, `quota`, `waiters`, `invoice` — 13 packages removed
- **Zero `/restapi/` references** remaining in the codebase

### Fixed

- **`auth validate`**: Now uses `region` API instead of dead `zone` API
- **Integration tests**: Migrated from deleted packages (`offering`, `securitygroup`, `tags`) to STKCNSL equivalents (`plan`, `instance`)

**Full Changelog**: https://github.com/zsoftly/zcp-cli/compare/0.0.6...0.0.7

---

## [0.0.6] - 2026-04-08

### Changed

- **API backend migration**: Migrated entire CLI from STKBILL to STKCNSL API backend
- **Authentication**: Switched from `apikey`/`secretkey` headers to Bearer token authentication
- **Config format**: Profile config now uses `bearer_token` instead of `apikey`/`secretkey`
- **API style**: All endpoints now use RESTful paths with slug identifiers instead of RPC-style paths with UUID query parameters
- **Response handling**: All API responses now parsed from `{status, message, data}` envelope format with built-in pagination support

### Added

- **`dns`**: Domain and record management (list, show, create, delete, record-create, record-delete)
- **`project`**: Project management (list, create, update, dashboard, icons, users)
- **`monitoring`**: Resource monitoring (global, per-VM CPU/memory/disk/network metrics)
- **`billing`**: Billing operations (costs, balance, invoices, subscriptions, usage, coupons, budget alerts)
- **`support`**: Support ticket management (CRUD, replies, feedback, FAQs)
- **`autoscale`**: VM autoscale groups with policies and conditions
- **`dashboard`**: Account service counts overview
- **`plan`**: Service plan listing for all resource types (VM, storage, K8s, LB, etc.)
- **`store`**: Store items and checkout
- **`marketplace`**: Marketplace app listing
- **`product`**: Product categories and listing
- **`iso`**: ISO image management (list, create, update, delete)
- **`affinity-group`**: Affinity group management (list, create, delete)
- **`backup`**: VM and block storage backup operations
- **`region`**: Region listing (replaces zone-based discovery)
- **`project delete`**: Delete projects with confirmation prompt
- **`--auto-approve` / `-y`**: Global flag to skip all confirmation prompts (useful for CI/CD automation)
- **`--blockstorage-plan`**: Required flag on `instance create` for selecting block storage plan size
- **`billing cancel-service`**: Now accepts `--service`, `--reason`, `--type` flags matching the API requirements
- **VM operations**: reboot, reset, tags, change-hostname, change-password, change-plan, change-OS, add-network, addons
- **Network egress rules**: list, create, delete egress firewall rules
- **VPC ACL management**: list, create, replace ACL rules
- **VPC VPN gateways**: list, create, delete
- **Load balancer rules**: create rules, attach VMs to rules
- **Discovery endpoints**: cloud-providers, currencies, storage-categories, billing-cycles, unit-pricings
- **Envelope helpers**: `GetEnvelope`/`PostEnvelope`/`PutEnvelope` on httpclient for clean response unwrapping
- **Generic response types**: `response.Envelope[T]` and `response.Single[T]` in new `api/response` package

### Removed

- **STKBILL API support**: All old `/restapi/` RPC-style endpoints removed
- **`apikey`/`secretkey` config fields**: Replaced by `bearer_token`
- **Zone-based filtering**: Replaced by region/slug-based resource identification

**Full Changelog**: https://github.com/zsoftly/zcp-cli/compare/0.0.5...0.0.6

---

## [0.0.5] - 2026-03-31

### Fixed

- **Delete false positives**: `vpc`, `network`, `volume`, and `security-group` delete
  commands now verify the resource is actually gone after delete; warn if it still exists
- **Volume list duplicates**: Deduplicate by UUID (Kong API returns duplicate entries)
- **Snapshot error message**: `snapshot create` on a detached volume now gives a clear
  message instead of raw CloudStack error
- **Firewall list on empty accounts**: Returns empty table instead of API error when
  account has no IP addresses

**Full Changelog**: https://github.com/zsoftly/zcp-cli/compare/0.0.4...0.0.5

---

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
