# Changelog

All notable changes to zcp will be documented in this file.

Format based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), using
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.0.19] - 2026-06-21

### Fixed

- **`zcp loadbalancer create` always failed because the API requires an initial rule.** The request sent an empty `rules` array, which the backend rejects. Create now builds a first rule from new flags — `--public-port`, `--private-port`, and `--algorithm` are **required** (the rule cannot be formed without them), with `--protocol` defaulting to `tcp` and `--rule-name` defaulting to `<lb-name>-rule`. Optional `--sticky-method`, `--enable-tls`, `--enable-proxy-protocol`, and repeatable `--vm <slug>` (attach back-ends) round out the rule. Add further rules afterward with `zcp loadbalancer create-rule`.
- **`zcp instance list` (and every instance lookup) only saw the first page of VMs.** The `/virtual-machines` endpoint is paginated, but the client fetched a single page, so accounts with more VMs than one page silently lost the rest — and, with reference resolution (below), a valid instance could resolve to "not found". `List` now walks every page.

### Added

- **Reference an instance by its ID (`vm_id`), name, or slug — not just its slug.** Every instance subcommand (`get`, `start`, `stop`, `reboot`, `reset`, `delete`, `logs`, `ssh`, `tag-*`, `change-*`, `add-network`, `addons`, `purchase-addon`) now resolves the argument against your VMs: exact ID/`vm_id` first, then name (reported as ambiguous, with the matching IDs, if two VMs share a name), then slug. Resolution searches your active region/project first and **falls back to an unscoped lookup** when the reference isn't found there, so operating on a globally-unique ID/slug keeps working across regions without passing `--region`.
- **`ID` column in instance output.** `zcp instance list` and `zcp instance get` now show the instance ID (the value to copy for the reference above); `get` also shows the record ID when it differs. `-o json`/`-o yaml` and `--debug` expand to the full column set (slug, template, created).
- **Account access control from the CLI: sub-users, roles, and permissions.** Three new account-level command groups (region-free, like `dns`): `zcp sub-user` (`list`/`create`/`update`/`block`/`unblock`/`delete`, alias `subuser`), `zcp role` (`list`/`get`/`create`/`update`/`delete`), and `zcp permission list`. Sub-users are addressable by **ID or email**; `create` requires `--name`, a company `--email`, a strong `--password` (8+ chars, mixed case + number + symbol), a `--role` slug, and one or more `--project` slugs — newly created sub-users start blocked until `unblock`. Roles group permission slugs from `zcp permission list`: `create`/`update` take repeatable `--permission`, which **replaces** the role's set (not additive); `update` preserves the flags you don't pass; and the predefined `owner`/`service-administrator`/`service-viewer` roles are protected from edit/delete with a clear message. Deletes are idempotent (including the API's 500 `No query results` for an already-deleted role). Verified end-to-end against the live API (list → create → update → block/unblock → delete) and covered by unit and smoke tests.
- **Affinity-group `--type` help and docs corrected** to the four values the API actually accepts — `host affinity`, `host anti-affinity`, `non-strict host affinity`, `non-strict host anti-affinity` — replacing the previous `host-affinity` example, which the API rejects as invalid.

### Changed

- **`zcp instance reboot` refuses a VM that isn't `Running`** with a clear message (`instance "…" is Stopped; it must be Running before it can be rebooted`) instead of issuing a reboot the platform silently ignores.
- **`zcp loadbalancer list` and `zcp instance list` emit full objects for `-o json`/`-o yaml`** instead of a flattened, all-string projection of the table columns — automation now gets every field.
- **`zcp auth validate` honors `ZCP_DEBUG`** (it previously only checked the `--debug` flag), matching every other command.
- **Documentation URL updated** to `https://docs.zcp.zsoftly.ca`.

## [v0.0.18] - 2026-06-19

### Fixed

- **`zcp ssh-key import` always failed with `API error 500 … Attempt to read property "id" on null`** — the API requires **both** `project` and `region` (it derives the cloud provider from them); the CLI marked both optional, so a call without them sent neither and the backend dereferenced a null. `--project` and `--region` are now **required** (honoring `ZCP_PROJECT`/`ZCP_REGION`), and both are always sent. Verified end-to-end against the live API: import → list → reference at VM create (VM came back with the key attached) → delete.
- **API validation errors (HTTP 422) showed only the generic `Validation errors` with no detail** — this API returns field-level messages under `data` (not `errors`) and omits `status`, so they were dropped. `apierrors.ParseResponse` now surfaces them from either location, e.g. `Validation errors — public_key: The public key has already been taken.` / `name: The name field must not be greater than 20 characters.`
- **Region-specific catalog listings returned every region's entries, not just the target region.** The commands sent no region and the API returns all regions unless filtered, so e.g. `zcp plan vm` listed both YUL (`ca*`) and YOW (`ci*`) offerings. Picking a wrong-region plan (Intel `ci*` in YUL) then failed to **schedule** ("no destination found") — the VM sat in `Starting`, flipped to `Error`, and was cleaned up with no IP, which looked like a boot failure. All region-specific catalog commands now **require a region** (`--region` or `ZCP_REGION`) and send `filter[region]=<slug>`: `zcp plan …` (all service types; use `os-yow`/`os-yul` for `plan object-storage`), `zcp template list`, `zcp iso list` (which previously ignored `--region` entirely), `zcp marketplace list`, and `zcp storage-category list`. Genuinely global catalogs are unchanged (`region`, `cloud-provider`, `currency`, `billing-cycle`, `server`). This does **not** fix the underlying CMP catalog, which still presents cross-region offerings as selectable for a target region — that needs region-scoped offering filtering in the plan catalog (a cmp2.0-ansible-zsoftly change).

### Added

- **`zcp ssh-key import` validates `--name` length client-side** (≤ 20 chars) before calling the API, with a clear message instead of a server round-trip.
- **`--region` and `--project` are now mandatory for every region/project-scoped command**, enforced centrally for all action commands. Satisfy them with `--region`/`--project`, the `ZCP_REGION`/`ZCP_PROJECT` env vars, or per-profile defaults saved by `zcp profile add`. The only exemptions are account-level/meta/discovery commands that have no region/project dimension: `dns`, `auth`, `profile`, `region`, `project`, `cloud-provider`, `currency`, `billing-cycle`, `server`, `support`, `dashboard`, `billing`, `product`, `store`, `version`, `completion`, plus two mixed groups whose region/project-scoped subcommands validate scope themselves while their other subcommands are account-wide: `ssh-key` (`list`/`delete` are account-wide; `import` requires `project`+`region`) and `object-storage` (acts on `os-yul`/`os-yow` regions distinct from the compute default, so `create`/`list` resolve their own region). `monitoring` is gated like the rest. Every resource and catalog **list also filters** its output by the region and project (`filter[region]`/`filter[project]`): instances, networks, IPs, volumes, VPCs, Kubernetes clusters, load balancers, virtual routers, autoscale groups, affinity groups, block-storage/VM snapshots, block-storage/VM backups, object storage, and all `plan`/`template`/`iso`/`marketplace`/`storage-category` catalogs.
- **`zcp profile add` now captures a default region (required) and project**, like `aws configure`. They are stored in the profile and used to satisfy the mandatory region/project requirement when no flag/env is set, so `zcp profile add` once and subsequent commands need no `--region`/`--project`.

### Changed

- **`zcp instance create --ssh-key <name>`** now sends `authMethod: "ssh-key"` (and an empty `password`) alongside the key name, matching the Web UI's VM-create payload — previously the key name was sent without the auth-method flag, so SSH-key auth would not engage. The key is referenced here by name (import it first with `zcp ssh-key import`).
- **Docs/help clarified** that (a) SSH key names and **public-key material** must be unique — re-importing the same key (even under a new name) is rejected with "The public key has already been taken"; and (b) a **VPC alone cannot host a VM** — you must create a network (tier) inside it (`zcp network create --vpc …`) and attach a VM to that tier (`zcp instance add-network`), since a bare VPC has no usable subnet.

## [v0.0.17] - 2026-06-17

### Added

- **`zcp object-storage object download <storage> <bucket> <key>`** — download an object's contents to a local file over the S3 protocol (minio-go `FGetObject`). `--dest` sets the destination (a file path, or a directory to write the object's base name into); it defaults to the base name in the current directory. Previously the CLI could upload, list, delete, and show object metadata, but had no way to fetch an object's bytes (`object get` returns metadata only).
- **Object-storage S3 feature build-out (Tier 1 + Tier 2 lifecycle), all via the existing minio-go S3 client.** These operations talk directly to the Ceph RADOS Gateway and are **CLI-only** — the CMP has not yet exposed them on the ZCP REST API or Web UI:
  - **Versioning workflows:** `object versions` (list versions + delete markers), `object download`/`object delete --version-id`, and `object restore` (undelete by removing the latest delete marker) — versioning is now usable, not just toggleable.
  - **`object copy` / `object move`** — server-side copy (and copy-then-delete), no download/upload round-trip.
  - **`object stat`** — full S3 metadata via HEAD (size, content-type, ETag, storage class, user metadata); `object put` gains `--content-type` and `--metadata key=value` (sets `x-amz-meta-*`).
  - **`object put-url`** — pre-signed URL for client uploads (HTTP PUT), the symmetric half of `object url`.
  - **`bucket uploads list|abort`** — see and reclaim storage held by incomplete multipart uploads.
  - **Richer lifecycle:** `bucket lifecycle expire` gains `--noncurrent-days` (expire old versions) and `--abort-multipart-days` (clean up stalled uploads), pairing with versioning.
  - `bucket policy get` / `lifecycle get` / `cors get` now honor `-o yaml` (JSON document is converted to YAML; table falls back to JSON).
- **Object-storage bucket management built out on the S3 protocol (minio-go), all confirmed against live Ceph RGW:**
  - `bucket tag get|set|delete` and `object tag get|set|delete` — bucket and object tags (`--tag key=value`, repeatable).
  - `bucket encryption status|enable|disable` — default SSE-S3 encryption.
  - `bucket lifecycle expire --days N [--prefix P] | get | delete` — object-expiration rules.
  - `bucket cors set --origin --method [--header --max-age] | get | delete` — cross-origin rules for browser apps.
  - `bucket empty` and `bucket delete --purge` — remove all objects **and object versions**, fixing the case where a bucket that ever had versioning enabled could not be deleted (its versions/delete-markers blocked the REST delete with a vague 403).
- **`zcp object-storage bucket versioning enable|suspend|status`** — manage S3 object versioning on a bucket (minio-go). Verified supported on Ceph RGW.
- **`zcp object-storage bucket policy get|set|delete`** — read, set (from a JSON file or `--file -` stdin), or remove a bucket's raw S3 policy, for fine-grained access beyond the canned `set-acl` public/private. Verified against live Ceph RGW.
- **`zcp object-storage object url <storage> <bucket> <key>`** — generate a pre-signed, time-limited HTTPS URL (minio-go) that a client can use to download an object with no ZCP credentials, even when the bucket is private. `--expires` sets the lifetime (default `1h`, max `168h`/7 days).
- **`zcp object-storage bucket set-acl --acl public-read|public-read-write|private`** now makes a bucket genuinely public/private by applying an S3 **bucket policy** via minio-go (the mechanism Ceph honors for anonymous object access). Previously it called a REST endpoint that 500'd, and a bucket canned ACL would not have granted anonymous `s3:GetObject` anyway.
- **`zcp plan object-storage`** — list Object Storage plans (the slugs for `object-storage create --plan`), with storage size and pricing.

### Fixed

- **`zcp object-storage bucket create` always failed with a 500** — the request sent only `{name}`, but the API requires an initial ACL grant (`The acl grantee field is required`). It now sends `acl_grantee: "Owner"` + `acl_permission: "FULL_CONTROL"` (a private bucket owned by the creator; make it public later with `bucket set-acl`).
- **`zcp object-storage create --plan` required a matching `--storage-category` or failed with `Invalid Storage Category`** — the category is now derived automatically from the plan (the API rejects a mismatch and requires a non-empty category even with a plan), so `--storage-category` is optional. Object-storage regions are `os-yul`/`os-yow`; custom `--storage-gb` is not configured there, so use a `--plan`.

  All of the above were verified end-to-end against the live Ceph object storage in YOW: create instance (plan-only) → create bucket → upload → download (content verified) → make public (anonymous GET 200) → pre-signed URL (200) → make private (anonymous GET 403) → delete.

### Changed

- **Actionable errors for missing/extra arguments** — every command that takes positional arguments (128 subcommands across 30 command groups) now prints what's wrong, the correct usage line, and the command's own examples instead of cobra's terse default. Previously `zcp profile add` printed only `Error: accepts 1 arg(s), received 0`; it now prints:

  ```text
  Error: missing required argument: <name>

  Usage:
    zcp profile add <name> [flags]

  Examples:
    zcp profile add default
    zcp profile add prod --bearer-token <token>
  ```

  Multi-argument commands name each missing placeholder (e.g. `missing required arguments: <vpc-slug>, <acl-name-or-id>`), and supplying too many arguments reports `too many arguments: expected N, got M` with the same usage/examples block. This is purely a messaging change — which commands accept how many arguments is unchanged.

- **Unknown subcommands now error instead of silently printing help** — running a command group with an invalid subcommand (e.g. `zcp region lists`) used to print the group's help text and exit `0`, which hides the typo and lets scripts treat a mistake as success. It now prints an error to stderr and exits non-zero. The message adapts to the group: a group with a single subcommand points straight at it (with its example), while a group with several lists the valid subcommands and suggests the closest match:

  ```text
  Error: unknown subcommand "lists" for "zcp region"

  Run this instead:
    zcp region list    List available regions

  Example:
    zcp region list
  ```

  Running a group with no arguments still prints its help and exits `0`.

- **`--cloud-provider` is now auto-detected** — the cloud provider slug is no longer something customers pass. `zcp auth validate` and `zcp profile add` detect the account's compute provider (the one whose service catalog includes "Virtual Machine" — `nimbo` in production) and persist it to the profile (`cloud_provider` field); create commands read it automatically. Verified against the production `/cloud-providers` catalog, which has three providers: `nimbo` (Cloud Stack, all compute/storage/networking), `ceph` (Object Storage), and `dns` (Dns Domain). Object storage and DNS default to their own providers (`ceph` and `dns`) automatically. The flag is hidden from help but still works as an override, and `ZCP_CLOUD_PROVIDER` is still honored. When the provider can't be determined, create commands now print `could not determine cloud provider — run 'zcp auth validate' to detect it, or pass --cloud-provider …` instead of the old terse `--cloud-provider is required`. All three provider values were verified against the live API: every region maps 1:1 to a provider (`yow-1`/`yul-1`→`nimbo`, `default`→`dns`, `os-yow`/`os-yul`→`ceph`), and a real instance and volume both store `cloud_provider: nimbo`.

- **`dns create` is now hands-off and uses the correct region** — it defaults `--region` to `default` (the only region the `dns` provider serves) and ignores the compute-oriented `ZCP_REGION`, so `zcp dns create --name example.com --project default` works on its own. Previously the docs/examples paired DNS with `--region yow-1` and `--cloud-provider nimbo`, which belong to the compute provider and would mismatch. Object-storage examples likewise now use object-storage regions (`os-yul`/`os-yow`) instead of `yul-1`.

### Removed

- **Backend technology is no longer shown in command output** — display-only columns that surfaced the underlying platform (e.g. "Cloud Stack", "Ceph", "Dns", "PowerDNS") have been removed: the `PROVIDER`/`COMING SOON` columns from `region list`, `SERVICE` from `backup list`, `snapshot list`, and `vm-backup list`, the `Service` row from `instance get`, `DISPLAY NAME` from `cloud-provider list`, and the `DNS PROVIDER` column / `DNS Provider` row from `dns list`, `dns show`, and `dns create`. The `--dns-provider` flag (which named the backend, e.g. `powerdns`) is likewise hidden from help, keeping its working default. These fields were informational only — resource creation uses the provider/region **slug**, which is retained. Billing/dashboard/project "SERVICE" columns are unaffected (they name billing categories like "Virtual Machine", not backend tech).

### Internal

- Added `internal/commands/args.go` with drop-in `exactArgs`/`minArgs`/`maxArgs`/`rangeArgs` validators replacing `cobra.ExactArgs`/`MaximumNArgs` throughout the `commands` package. Missing-argument names are derived from each command's `Use` line and examples from its `Example` field, so the helpers need no per-command wiring.
- `EnforceSubcommandErrors` in the same file walks the command tree once (called from `root.go` after all subcommands are registered) and installs the unknown-subcommand handler on every group, so new command groups get the behavior automatically.

---

## [v0.0.16] - 2026-06-11

All fixes below were confirmed against the live API (YUL region) before release.

### Added

- **`zcp network create --vpc <vpc-slug>`** — create a network as a VPC subnet (tier). Sends `type=Vpc` (the exact value the API requires); `--gateway`, `--netmask`, and `--billing-cycle` are validated as required. Previously VPC tiers could not be created from the CLI at all.
- **`zcp network create --acl <name-or-id>`** — attach a custom network ACL immediately after creating a VPC subnet (the API has no attach-at-create parameter, so the CLI creates the network and then calls the replace-ACL endpoint). ACL names are resolved to IDs automatically.
- **`zcp network create --network-plan <slug>`** — network plan for isolated/L2 networks (required by the live API; the old `--category` flow never worked because the categories endpoint returns an empty list). `--type Isolated|L2` selects the network type.
- **`zcp network get <slug>`** — new command showing provider-side state from `GET /networks/{slug}`: CIDR, gateway, netmask, state, VPC ID, and attached ACL. Previously there was no way to see a network's CIDR/state/VPC membership from the CLI.
- **`zcp plan network`** — new command listing Network plans (pNet/iNet/l2Net) with their slugs and network types; these slugs are the values for `--network-plan`.
- **`zcp acl delete <vpc-slug> <acl-name-or-id>`** — new command; the platform now supports `DELETE /vpcs/{slug}/network-acl-list/{id}`.
- **`zcp acl rules` / `zcp acl create-rule` / `zcp acl update-rule` / `zcp acl delete-rule`** — full ACL rule management. `update-rule` replaces a rule in place (the rule ID is preserved) and `--cidr` accepts comma-separated lists (e.g. `10.30.1.0/24,10.30.2.0/24`). The rule endpoints live under `/vpcs/{vpc}/network-acl-list/{acl_list_id}/network-acl` (note the singular segment — `/rules`-style paths do not exist). Rules are added one per request after the ACL list exists; `create-rule` validates the live API contract (ports required for tcp/udp, ICMP type/code for icmp, allow/deny, ingress/egress) and resolves ACL names to IDs.
- **`zcp acl replace --vpc`** / **`zcp vpc acl-replace --vpc`** — optional flag to resolve an ACL _name_ to its ID; without it, `--acl` must be the ACL ID.
- **`pkg/api/network.Service.GetDetail`**, **`pkg/api/acl.Service.Resolve`**, **`pkg/api/acl.Service.Delete`** — new service methods.

### Fixed

- **`zcp acl replace` / `zcp vpc acl-replace` — always failed with 403** — the request body used `aclSlug`; the live API requires `acl_id` with the ACL's ID. Both commands now send the correct field and accept names via `--vpc` resolution.
- **Silent detached-network trap** — sending `type=Isolated` together with `vpc` passes API validation but silently ignores the VPC and creates a standalone isolated network. The CLI now always sends `type=Vpc` when `--vpc` is set and rejects a conflicting `--type`.
- **`zcp vpc get` — CIDR/Status/Zone always blank** — the command filtered the list endpoint, which omits provider state. It now calls `GET /vpcs/{slug}` and maps the CloudStack `meta` block (state, cidr, zone_name, network_domain), falling back to the list for older deployments.
- **`zcp vpc create` — blank CIDR/Status in output** — the create response omits provider state; the command now fetches the detail view after creation.
- **`zcp vpc create` — network-address quirk** — the API records the network address verbatim (e.g. `10.30.0.1/16` instead of `10.30.0.0/16`); the CLI now prints a warning when the given address is not the canonical network base.
- **`zcp plan router` (and lb/k8s/ip/vm-snapshot/template/iso/backup/storage) — missing SLUG column** — `vpc create --plan` requires a plan slug but no plan table showed one. All plan tables now include SLUG.
- **`zcp network create` — crash decoding create response** — the create endpoint returns `is_default` as `0/1` while the list endpoint returns `true/false`; the decoder now accepts both.
- **`zcp acl list` / `zcp vpc acl-list` — SLUG/STATUS columns always blank** — the live API returns `id`, `name`, `description`; tables now show ID (needed for `acl replace`/`acl delete`).
- **`zcp vpc delete` — false "may not have been deleted" warning** — deletion is an async CloudStack job, but the command checked existence once after 2s and reported failure (with a misleading "delete all network tiers first" hint) while the job was still completing. It now polls for up to 30s, reports success when the VPC is gone, and otherwise says the deletion may still be in progress or blocked, with the exact command to check.
- **`zcp network update --description` — failed with a 500** — the API requires `name` on every PUT; a description-only update now re-sends the current name automatically.
- **Cryptic errors when deleting already-deleted resources** — the API reports missing resources as `403 "The provided <resource> is invalid."`; this is now recognized as not-found, so `vpc delete`, `acl delete`, and `acl delete-rule` print "already deleted" and exit 0 instead of surfacing a raw 403 (validation errors, which use "selected", are unaffected).
- **`zcp profile delete` — ignored `-y`/`--auto-approve`** — the global auto-approve flag now skips the confirmation prompt (it previously only honored its own `--yes`).
- **`zcp network create` — required `--category`** — the flag is now optional (legacy); the live API ignores it and requires `network_plan` + `type` instead.

### Known platform limitations (not CLI bugs)

- An embedded `rules` array on ACL-list create is silently ignored — create the list first, then add rules one per request (`zcp acl create-rule`).
- VPCs are limited to 3 subnets (CloudStack `vpc.max.networks`); the 4th create returns a generic 403.
- VPC `description` is not persisted by the API.

---

## [v0.0.15] - 2026-06-10

### Fixed

- **`zcp vpc vpn-gateway create` — crash on `data:null` API response** — the create endpoint returns `{"data":null}` instead of the created gateway object; the command now falls back to listing gateways and returning the first one found, exactly matching how the portal behaves
- **`zcp vpc vpn-gateway list/create` — all fields blank** — `VPNGateway` struct had wrong JSON tags (`slug`, `publicIpAddress`, `vpcUuid`, `vpcSlug`, `zoneName`, `status`); corrected to match actual API response (`id`, `public_ip`, `vpc_id`, `vpc_name`); `zcp vpc vpn-gateway list` now displays `ID`, `PUBLIC IP`, and `VPC ID` correctly
- **`zcp vpc update` — returns empty fields** — PUT `/vpcs/{slug}` returns `data:null`; the command now falls back to a GET to return the updated VPC state
- **`zcp network update` — crashes with JSON decode error** — PUT `/networks/{slug}` returns `data:[null]` (an array); the command was attempting to unmarshal an array into a struct, causing a fatal error; now falls back to a GET after any non-usable PUT response
- **`zcp vpn customer-gateway create/update` — all VPN config fields blank** — `CustomerGateway` struct had wrong JSON tags for three fields: `ipsec_preshared_key` → `ipsecpsk`, `force_encapsulation` → `forceencap`, `dead_peer_detection` → `dpd`; and `SplitConnections` was typed as `bool` but the API returns it as a string; all corrected
- **`zcp vpn customer-gateway create` — shows empty result** — create API returns a metadata-only response (no VPN config); the command now calls GET `/vpn-customer-gateways/{slug}` after creation and falls back gracefully to the partial slug/name when CloudStack provisioning is still in progress
- **`zcp vpn customer-gateway update` — all VPN fields blank** — PUT response is a metadata-only envelope (no VPN config fields); the command now always falls back to GET `/vpn-customer-gateways/{slug}` to return the full VPN configuration
- **`zcp vpn customer-gateway update` — `--cloud-provider`/`--region`/`--project` missing** — the API requires these three fields on every PUT; the update command now accepts and validates `--cloud-provider`, `--region`, and `--project` flags (resolving from env vars as with create)
- **`zcp vpn user list` — Username column always blank** — `User.UserName` had JSON tag `userName` (camelCase); corrected to `username` to match the API response
- **`zcp network update --description ""` / `zcp vpc update --description ""` ignored** — `description` field in `UpdateRequest` had `omitempty`; clearing a description sent no field and the API ignored it; `omitempty` removed from both structs

### Added

- **`zcp vpn customer-gateway create/update --ike-dh`** — new flag for IKE Diffie-Hellman group (e.g. `modp2048`); required by the API
- **`zcp vpn customer-gateway create/update --esp-dh`** — new flag for ESP Diffie-Hellman group
- **`zcp vpn customer-gateway create/update --esp-pfs`** — new flag for ESP Perfect Forward Secrecy group (e.g. `modp2048`); required by the API
- **`zcp ip allocate --project`** — new optional flag to assign the IP to a specific project at allocation time
- **`vpn.CustomerGatewayService.Get(ctx, slug)`** — new service method; fetches a single customer gateway's full VPN config from `GET /vpn-customer-gateways/{slug}`
- **`network.Service.Get(ctx, slug)`** — new service method; fetches a single network by slug using list-and-filter

### Internal

- `pkg/api/vpc.VPNGateway` — removed `Slug`, `VPCUUID`, `ZoneName`, `Status` fields; renamed/added `ID`, `PublicIP`, `VPCID`, `VPCName` with corrected snake_case JSON tags
- `pkg/api/vpn.CustomerGatewayRequest` — all JSON tags updated to snake_case; `ForceEncap` → `ForceEncapsulation`, `SplitConnection` → `SplitConnections`, `DPD` → `DeadPeerDetection`; new fields `IKEDH`, `ESPDH`, `ESPPFS` added; `Update()` now always calls `Get()` fallback to return full VPN config
- `pkg/api/ipaddress.CreateRequest` — added `Project string \`json:"project,omitempty"\``

---

## [v0.0.14] - 2026-06-10

### ⚠ Breaking Changes

**API packages moved from `internal/` to `pkg/`.**

All service packages and the HTTP client are now importable by external Go modules
(e.g. the ZCP Terraform provider). If you import any of these paths in your own code,
update them as follows:

| Old path                                            | New path                                       |
| --------------------------------------------------- | ---------------------------------------------- |
| `github.com/zsoftly/zcp-cli/internal/httpclient`    | `github.com/zsoftly/zcp-cli/pkg/httpclient`    |
| `github.com/zsoftly/zcp-cli/internal/api/<service>` | `github.com/zsoftly/zcp-cli/pkg/api/<service>` |

CLI end users are **not affected** — the binary behaviour is unchanged.

### Changed

- All 28 API service packages moved: `internal/api/*` → `pkg/api/*`
- HTTP client moved: `internal/httpclient` → `pkg/httpclient`
- CLI-internal packages (`internal/commands`, `internal/config`, `internal/output`, `internal/version`) remain under `internal/` and are not part of the public API

### Meta

- Release tags now use the `v` prefix (`v0.0.14`, `v0.1.0`, …) to align with Go module
  and Terraform Registry conventions. The previous tag format (`0.0.12` etc.) is preserved
  for backwards compatibility but will not be used for future releases.
- CI pipeline updated to trigger on `v[0-9]*` tags

**Full Changelog**: https://github.com/zsoftly/zcp-cli/compare/0.0.12...v0.0.14

---

## [0.0.12] - 2026-06-09

### Added

- **`zcp kubernetes upgrade-version`** — upgrade the Kubernetes version of a running cluster; accepts `--version v1.x.y` (e.g. `v1.35.1`, `v1.36.1`); resolves the correct version slug for the cluster's region automatically from the CMP catalog; returns a clear error if the requested version is unavailable in that region

### Fixed

- **`zcp kubernetes scale` — misleading state-guard error** — error message said "scale requires Running state" even though the guard accepts both Running and Scaling; corrected to "requires Running or Scaling state"

### Internal

- `ClusterMeta` gains `KubernetesVersionID` field; used by `upgrade-version` to match the cluster's current version against the catalog and derive its region
- `kubernetes.Service` gains `ListVersions` and `UpgradeVersion` methods
- `docs/api-inventory.md` — duplicate endpoint number 112 in the Projects table corrected to 117; cascading +5 renumber across ISOs (118–121), Affinity Groups (122–124), Templates (125–129), Monitoring (130–136), Object Storage (137–148), Billing (149–165), Profile (166–172), SSH Keys (173–175), Support (176–184), Plans (185–194), Discovery (195–201), Store (202–205), Auth (206–211); total updated 206 → 211
- `docs/command-taxonomy.md` — kubernetes section corrected: `scale`, `upgrade-version`, `get-config` added; `upgrade` relabelled as compute-plan change (not version upgrade)

**Full Changelog**: https://github.com/zsoftly/zcp-cli/compare/0.0.11...0.0.12

---

## [0.0.11] - 2026-06-08

### Added

- **`zcp instance create --user-data`** — pass a cloud-init / user-data script inline at VM creation time, matching portal capability
- **`zcp instance create --user-data-file`** — read user-data from a local file; mutually exclusive with `--user-data` (returns a clear error if both are set)

### Fixed

- **`zcp ssh-key import`** — CLI was sending `cloud_provider` to the API; the portal (and API) expect `region`. The `--cloud-provider` flag has been replaced with `--region` on this command
- **`zcp instance get` / `zcp instance list` — Private IP always blank** — the top-level `private_ip` API field is `null`; the real value is in `networks[].pivot.ipaddress`. Both commands now read from the network pivot, preferring the default network (`is_default` / `pivot.is_default`) before falling back to the first attached network
- **`zcp instance get` — transient 403 after VM creation** — CMP returns a 403 "The route virtual-machines/\<slug\> could not be found" briefly after creation while its routing layer indexes the new slug. `instance get` now retries up to 5 times with 2 / 4 / 8 / 16-second exponential backoff before surfacing the error
- **`zcp dns record-delete`** — "not found" log message used `%q` (string-quote verb) for the integer record ID; corrected to `%d`

### Changed

- **`zcp instance create --blockstorage-plan`** — no longer required; backend auto-assigns when omitted (consistent with portal behaviour)
- **`zcp instance create --network-plan`** — help text example values corrected from `inet-yow` / `inet-yul` to `pnet-yow` / `pnet-yul`
- **`zcp template account-delete --yes`** — added `-f` shorthand; `-y` is reserved globally by `--auto-approve`
- **`zcp volume create --size`** — validation now uses `cmd.Flags().Changed("size")`; explicitly passing `--size 0` returns `--size must be > 0` instead of the misleading `--plan or --size is required`; `--plan` and `--size` are now mutually exclusive even when `--size 0` is passed

### Internal

- **`IsTransientRoutingError`** — detection tightened from broad substring match to anchored regexp `(?i)\bthe route\b.*could not be found`, reducing false-positive risk as CMP error messages evolve
- **`NetworkPrivateIP()`** — respects `is_default` / `pivot.is_default` network ordering instead of returning the first slice element; stable when multiple networks are attached
- **Retry backoff timer** — `instance get` retry loop replaced `time.After` with `time.NewTimer` + `Stop()` to release the timer immediately on context cancellation rather than leaving it to fire unobserved

### Tests

- **`TestIsTransientRoutingError`** (10 subtests) — covers exact live CMP message, case-insensitivity, wrong status codes, partial-phrase matches, no-prefix match, nil, and non-API errors
- **`TestInstanceGetRetrySucceeds`** — httptest server that returns two 403 routing errors then a 200; verifies call count and retry message in stderr
- **`TestInstanceGetRetryExhausted`** — always-403 server; verifies all 5 attempts are exhausted and the error is surfaced
- **`TestInstanceGetNonRoutingErrorNoRetry`** — non-routing 403; verifies a single attempt with no retry

**Full Changelog**: https://github.com/zsoftly/zcp-cli/compare/0.0.10...0.0.11

---

## [0.0.10] - 2026-06-07

### Added

- **`object-storage`** — full Ceph/S3 object storage management: list, get, create, delete, resize storage instances; bucket management (list, get, create, delete); object management (list, get, upload, delete); `zcp object-storage credentials` for S3 access key display. Cloud provider defaults to `ceph` — do not pass `--cloud-provider nimbo` (that is the CloudStack compute provider, not Ceph)
- **`zcp instance delete`** — permanently delete virtual machines; `--force` flag for immediate expunge (skips the deferred expunge window)
- **`zcp network delete`** — delete isolated networks with confirmation; also releases the associated SOURCE-NAT IP
- **`zcp snapshot delete`** — permanently delete block storage snapshots
- **`zcp vm-backup delete`** — permanently delete VM backup schedules
- **`zcp volume delete`** — permanently delete block storage volumes (detach first)
- **`zcp vpc delete`** — permanently delete VPCs
- **`zcp backup delete`** — permanently delete block storage backup schedules
- **`zcp kubernetes scale`** — scale worker node count up or down; idempotent (no-ops if already at target); `--wait` flag to block until the cluster returns to Running (10-min timeout, context-aware)
- **`zcp kubernetes get-config`** — download kubeconfig to stdout (default) or a file with `--output <path>`; prints the export command when writing to a file
- **`zcp lb delete-rule`** — delete a rule from a load balancer
- **`zcp lb detach-vm`** — detach a VM from a load balancer rule
- **Smoke testing framework** — `tests/smoke/` directory with live end-to-end test scripts (`smoke.sh`, `cases.sh`, `affected.sh`, `lib.sh`) for running full lifecycle tests against the ZCP API

### Fixed

- **`zcp vm-backup create --pseudo-service`** — flag was misspelled `--psudo-service`; corrected to `--pseudo-service` (API JSON tag `psudo_service` preserved for wire compatibility)
- **`zcp egress create`** — protocol values were sent as-typed; API requires uppercase; `tcp`/`udp` are now normalised to `TCP`/`UDP` before the request
- **`zcp kubernetes get`** — was showing `Workers=0`, `Version=""` for Running clusters; now reads real values from CloudStack meta fields (`meta.size`, `meta.kubernetes_version_name`, `meta.ipaddress`, `meta.end_point`)
- **`zcp kubernetes scale --wait`** — polling loop used `context.Background()` with no upper bound; now inherits `cmd.Context()` with a 10-minute deadline; non-`Scaling` terminal states (e.g. `Error`) return an error instead of looping forever
- **`zcp kubernetes get-config --output`** — directory creation used `strings.LastIndex` which panicked when the path had no `/`; replaced with `filepath.Dir`
- **`zcp kubernetes scale` idempotency** — `strconv.Atoi` parse error on `meta.size` silently zeroed `currentWorkers`, causing false "already at N" matches; now falls back to the top-level `node_size` field on parse failure
- **Confirmation prompts** — all 15 destructive-action prompts across 9 command files (`instance`, `network`, `ip`, `backup`, `snapshot`, `vmsnapshot`, `volume`, `vmbackup`, `objectstorage`) now use `bufio.Scanner`; prompts go to `stderr`; both `y` and `yes` are accepted; previously `fmt.Scanln` / stdout-only / `y`-only

### Changed

- **`zcp kubernetes get`** — prefers CloudStack meta fields over top-level API fields for all display values (version, workers, control nodes, IP, endpoint)
- **`zcp kubernetes create`** — `--storage-category` is now validated as required (API returns "quota not found" without it); enhanced with `--ssh-key`, `--auth-method`, `--username`, `--password` flags
- **`zcp plan list`** — now shows storage category in the output table
- **Project dashboard** — updated to consume structured service data returned by the API
- **Go toolchain** — upgraded to `go1.26.4`
- **Help text** — all `--cloud-provider` examples corrected from `zcp` to `nimbo` (12+ files); all angle-bracket placeholders replaced with realistic values (`bs-001001-0042`, `ss-001001-0001`, `en-001001-0018`, etc.); `mtl-1` corrected to `yul-1` in load balancer examples

### Tests

- **25 new API-layer unit tests** — Delete path/method/error coverage added to `backup`, `snapshot`, `volume`, `loadbalancer` (Delete, DeleteRule, DetachVM), `kubernetes` (Delete, Scale, GetKubeconfig — including the not-ready case)
- **New `vmbackup` test file** — List, Create, Delete (the package had no tests before)
- **Instance delete tests** — success, `--force`, and 404 not-found scenarios
- **IP address release tests** added to `ipaddress` package

**Full Changelog**: https://github.com/zsoftly/zcp-cli/compare/0.0.9...0.0.10

---

## [0.0.9] - 2026-04-14

### Added

- **Environment variable overrides**: `ZCP_PROJECT`, `ZCP_REGION`, `ZCP_CLOUD_PROVIDER`, `ZCP_OUTPUT`, `ZCP_DEBUG` — reduces repetitive flags in CI/CD and scripting
- **Zero-config mode**: CLI can now operate with only `ZCP_BEARER_TOKEN` and `ZCP_API_URL` env vars — no config file or profile required
- **`ZCP_PROFILE` env var**: Selects the active profile without `--profile` flag
- **`ZCP_BEARER_TOKEN` env var**: Overrides profile credentials at runtime
- **`ZCP_API_URL` env var**: Overrides the API base URL at runtime
- **Env var tests**: 14 new tests covering all resolution helpers and config env overrides

### Fixed

- **Kubernetes create**: Restored missing `--billing-cycle` validation (was accidentally removed)
- **Kubernetes create**: Fixed resolve order — `resolveRegion/resolveProject/resolveCloudProvider` now called before validation checks so env vars are applied correctly
- **All create commands**: Consistent resolve order — env var resolution always runs before required-field validation across all 18 command files

### Changed

- **`config.ResolveProfile`**: Now checks `ZCP_PROFILE` env var before falling back to `active_profile` in config file
- **`config.ActiveAPIURL`**: Now checks `ZCP_API_URL` env var before falling back to profile URL
- **Documentation**: Updated `docs/configuration.md` and `README.md` with all new environment variables and CI/CD usage examples

**Full Changelog**: https://github.com/zsoftly/zcp-cli/compare/0.0.8...0.0.9

---

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
- **ACL commands**: `zcp acl create` and `zcp vpc acl-create` now take `--name` and `--description` (matching the actual API)

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
