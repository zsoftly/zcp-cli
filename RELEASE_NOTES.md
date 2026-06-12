# zcp v0.0.16 Release Notes

## VPC Subnets, Network ACLs, and Plan Discovery

This release makes the full three-tier VPC workflow possible from the CLI: create a VPC,
create subnets (tiers) inside it, and attach custom network ACLs — none of which worked in
v0.0.15. All fixes were confirmed against the live API (a three-tier VPC was built end-to-end
in the YUL region with per-tier ACLs as the release verification).

---

## Added

### VPC subnet (tier) creation — `zcp network create --vpc`

```
zcp network create --name web-tier --vpc my-vpc --acl web-acl \
  --gateway 10.30.1.1 --netmask 255.255.255.0 --billing-cycle hourly \
  --cloud-provider nimbo --region yul-1 --project default
```

The API requires `type=Vpc` with that exact casing — any other value passes validation but
silently creates a _detached_ isolated network. The CLI now always sends the correct type and
rejects conflicting flags, so this trap is no longer reachable.

`--acl` attaches a custom ACL right after creation (the API has no attach-at-create
parameter); names are resolved to ACL IDs automatically.

### `zcp network get <slug>`

Shows provider-side state that no command exposed before: CIDR, gateway, netmask, state,
VPC membership, and the attached ACL.

### `zcp plan network`

Lists Network plans (pNet/iNet/l2Net) with slugs — the values for the new
`--network-plan` flag, which isolated/L2 network creation requires. The old `--category`
flow never worked against the live API (the categories endpoint returns an empty list);
`--category` is kept as an optional legacy flag.

### `zcp acl delete <vpc> <acl>`

The platform now supports ACL list deletion; the CLI exposes it with name resolution and a
confirmation prompt.

### ACL rule management — `zcp acl rules` / `create-rule` / `update-rule` / `delete-rule`

```
zcp acl create-rule my-vpc web-acl --number 1 --protocol tcp \
  --start-port 443 --end-port 443 --cidr 0.0.0.0/0 --action allow --traffic-type ingress
```

The rule endpoints live under `.../network-acl-list/{id}/network-acl` (singular). The order
of operations matters: create the ACL list first, then add rules one per request — an
embedded `rules` array on list creation is silently ignored. `create-rule` mirrors the live
validation (ports for tcp/udp, ICMP type/code for icmp) and resolves ACL names to IDs.
`update-rule` modifies a rule in place (rule ID preserved); `--cidr` accepts comma-separated
multi-CIDR lists on both create and update.

## Fixed

- **`acl replace` / `vpc acl-replace` always failed with 403** — body field was `aclSlug`;
  the API requires `acl_id` (the ACL's ID). Both now send the right field, and `--vpc`
  resolves ACL names to IDs.
- **`vpc get` / `vpc create` showed blank CIDR/Status/Zone** — now read from
  `GET /vpcs/{slug}` (CloudStack `meta` block) with a list fallback for older deployments.
- **All plan tables missing the SLUG column** — `vpc create --plan` needs a slug, but
  `plan router` only showed UUIDs. Every plan table now includes SLUG.
- **Create-response decode crash** — `is_default` arrives as `0/1` on create but
  `true/false` on list; the decoder accepts both.
- **`acl list` blank columns** — now shows ID / NAME / DESCRIPTION (matching the live API).
- **VPC network-address quirk** — the API stores the address verbatim (`10.30.0.1/16`);
  the CLI warns when it isn't the canonical network base.

## Known platform limitations (tracked in docs/roadmap.md)

- **3 subnets per VPC** (CloudStack `vpc.max.networks`) — the 4th create returns a generic 403.
- **VPC `description` is not persisted** by the API.
