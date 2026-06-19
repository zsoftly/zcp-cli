# zcp v0.0.18 Release Notes

## Region- and project-correctness: nothing runs unscoped

This release makes the CLI **region- and project-aware everywhere**, so it can no longer show
catalog entries that don't exist in your region or deploy a resource into the wrong zone. It also
fixes the SSH-key import flow and surfaces real API validation messages.

Highlights:

- **`--region` and `--project` are now mandatory** for every region/project-scoped command, with
  per-profile defaults captured at `zcp profile add` (like `aws configure`).
- **Every list filters by region and project** — no more cross-region clutter or un-deployable
  plans in the output.
- **`zcp ssh-key import` works** (it required `project` + `region` all along).
- **422 validation errors now show the field-level reason** instead of a generic message.

---

## Fixed

### `zcp ssh-key import` always returned `500 … Attempt to read property "id" on null`

The API derives the cloud provider from **both** `project` and `region`, and the CLI marked them
optional — so a call without them sent neither and the backend dereferenced a null. `--project` and
`--region` are now required (honoring `ZCP_PROJECT`/`ZCP_REGION`) and always sent. Verified
end-to-end: import → list → reference at VM create (the VM came back with the key attached) → delete.
`--name` is also validated client-side (≤ 20 chars) before the call.

### API validation errors were swallowed

HTTP 422 responses returned only a generic `Validation errors`. This API puts the field-level
messages under `data` (not `errors`) and omits `status`, so they were dropped. They're now surfaced:

```
Error: ... API error 422: Validation errors — public_key: The public key has already been taken.
Error: ... API error 422: Validation errors — name: The name field must not be greater than 20 characters.
```

### Catalog and resource listings returned every region's entries

`zcp plan vm` listed both YUL (`ca*`) and YOW (`ci*`) offerings; picking a wrong-region plan (an
Intel `ci*` plan in YUL) then failed to **schedule** ("no destination found") — the VM sat in
`Starting`, flipped to `Error`, and was cleaned up with no IP, which looked like a boot failure. The
CLI now sends `filter[region]`/`filter[project]` on every list, so you only ever see entries valid
for your region and project.

> This does not fix the underlying CMP catalog, which still presents cross-region offerings as
> selectable for a target region — that needs region-scoped offering filtering in the plan catalog.

## Added

### Region + project are required everywhere (with profile defaults)

Every region/project-scoped command now requires a region and a project. Satisfy them three ways:

```bash
# 1. Per-profile defaults (recommended) — captured at configure time, like `aws configure`
zcp profile add default        # prompts for token, default region, and default project

# 2. Environment variables
export ZCP_REGION=yow-1 ZCP_PROJECT=default-9

# 3. Per command
zcp instance list --region yow-1 --project default-9
```

Account-level commands have no region/project dimension and are exempt: `dns`, `auth`, `profile`,
`region`, `project`, `cloud-provider`, `currency`, `billing-cycle`, `server`, `support`, `dashboard`,
`billing`, `product`, `store`. (`object-storage` is scoped too; it uses the `os-yul`/`os-yow`
regions.)

### Every list is now scoped to your region and project

Lists send `filter[region]`/`filter[project]` and return only what belongs to that region/project —
instances, networks, IPs, volumes, VPCs, Kubernetes clusters, load balancers, virtual routers,
autoscale groups, affinity groups, block-storage and VM snapshots, block-storage and VM backups,
object storage, and the full catalog (`plan`, `template`, `iso`, `marketplace`, `storage-category`).
For example, `zcp plan vm --region yul-1` now lists only the `ca*` family YUL actually runs (the
Intel `ci*` plans appear only under `yow-1`).

### `zcp profile add` captures a default region and project

Like `aws configure`, `profile add` now prompts for (and requires) a default **region** and a
default **project**, stored in the profile and used whenever `--region`/`--project` and
`ZCP_REGION`/`ZCP_PROJECT` are not set:

```bash
zcp profile add default \
  --bearer-token <token> --region yow-1 --project default-9
```

## Changed

- **`zcp instance create --ssh-key <name>`** now sends `authMethod: "ssh-key"` (and an empty
  `password`) alongside the key name, matching the Web UI's VM-create payload — previously the
  key name was sent without the auth-method flag, so SSH-key auth would not engage.
- **Docs/help clarified**: SSH key names and the public-key material must be unique (re-importing
  the same key, even under a new name, is rejected); and a **VPC alone cannot host a VM** — create
  a network (tier) inside it (`zcp network create --vpc …`) and attach a VM to that tier
  (`zcp instance add-network`), since a bare VPC has no usable subnet.

## Upgrade notes

This release **requires a region and project** for scoped commands. If you have scripts that ran
`zcp instance list`, `zcp plan vm`, etc. without one, either re-run `zcp profile add` to store
defaults, export `ZCP_REGION`/`ZCP_PROJECT`, or add `--region`/`--project`. Account-level commands
(listed above) are unaffected.
