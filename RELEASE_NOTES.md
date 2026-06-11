# zcp v0.0.15 Release Notes

## Network / VPC / VPN Bug Fixes

This release fixes a set of related bugs across the VPC, VPN, and network commands where the
CLI was sending or receiving incorrect JSON field names — causing commands to return blank
output, crash with a decode error, or silently ignore updates. All issues were confirmed against
the live API before release.

---

## Fixed

### VPC VPN gateway — fields always blank / crash on create

`zcp vpc vpn-gateway create` and `list` both returned empty rows.

**Root cause:** `VPNGateway` struct used wrong JSON tags (`slug`, `publicIpAddress`, `vpcUuid`,
`vpcSlug`, `zoneName`, `status`). The live API returns `id`, `public_ip`, `vpc_id`, `vpc_name`.
Additionally, the create endpoint returns `{"data":null}` instead of the created object.

**Fix:** Corrected all JSON tags; create now falls back to listing and returning the first
matching gateway when the API returns null data.

### `zcp vpc update` — returns empty fields

PUT `/vpcs/{slug}` always responds with `data:null`. The command was unmarshaling null into
an empty struct and displaying blank values.

**Fix:** After any null PUT response, the command fetches the VPC via GET and returns the
real updated state.

### `zcp network update` — crashes with JSON decode error

PUT `/networks/{slug}` returns `data:[null]` (an array with a null element). Attempting to
unmarshal that into a `Network` struct caused a hard error.

**Fix:** Same fallback-to-GET pattern; now handles both `null` and `[null]` responses without
crashing.

### VPN customer gateway — all VPN config fields blank

`zcp vpn customer-gateway create`, `update`, and `list` all showed empty Gateway, IKE Policy,
CIDR, and other fields.

**Root causes:**

- Three JSON tags were wrong: `ipsec_preshared_key` → `ipsecpsk`, `force_encapsulation` →
  `forceencap`, `dead_peer_detection` → `dpd`
- `SplitConnections` was typed as `bool` but the API returns it as a string
- Create API returns a metadata-only response (no VPN config); a subsequent GET is now made
  to the detail endpoint `/vpn-customer-gateways/{slug}`

**Fix:** All tags corrected; `CustomerGatewayService.Get()` added; create falls back gracefully
when CloudStack provisioning is still in progress.

### VPN customer gateway — missing required API fields

The `--ike-dh`, `--esp-dh`, and `--esp-pfs` CLI flags were missing. The API rejects creates
without `ike_dh` and `esp_pfs`, so `zcp vpn customer-gateway create` could never succeed.

**Fix:** Three new flags added to both `create` and `update`.

### VPN user list — Username column always blank

`User.UserName` had JSON tag `userName`; the API returns `username`.

**Fix:** Tag corrected; username now appears in `zcp vpn user list`.

### `zcp network update --description ""` / `zcp vpc update --description ""` ignored

`description` in both `network.UpdateRequest` and `vpc.UpdateRequest` had `omitempty`.
Sending an empty string was silently dropped, making it impossible to clear a description.

**Fix:** `omitempty` removed from both structs.

### `zcp ip allocate` — cannot assign to a project

No `--project` flag existed on `zcp ip allocate`, and the `CreateRequest` struct lacked the
`project` field entirely.

**Fix:** Field added to the struct; `--project` flag exposed on the command.

### VPN customer gateway update — all fields blank + unusable without cloud context

`zcp vpn customer-gateway update` returned blank VPN config fields (gateway, IKE policy, CIDR,
etc.) because the PUT response is a metadata-only envelope — the VPN config fields are only
returned by the GET detail endpoint. Additionally, the update command was missing
`--cloud-provider`, `--region`, and `--project` flags, which the API requires on every PUT,
making the command return a server-side validation error in all cases.

**Fix:** `Update()` now always falls back to `GET /vpn-customer-gateways/{slug}` to retrieve
the full VPN configuration. The update command now accepts and validates `--cloud-provider`,
`--region`, and `--project` (resolving from env vars as with create).

---

## New service methods (for Go library consumers)

| Method                                      | Description                                         |
| ------------------------------------------- | --------------------------------------------------- |
| `vpn.CustomerGatewayService.Get(ctx, slug)` | Fetch full VPN config for a single customer gateway |
| `network.Service.Get(ctx, slug)`            | Fetch a single network by slug                      |

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
# zcp version v0.0.15
```

---

## Full Changelog

https://github.com/zsoftly/zcp-cli/compare/v0.0.14...v0.0.15
