# ZCP CLI Roadmap

Features planned, in progress, or blocked on platform support.

---

## Completed (v0.0.7+)

- Object storage — full instance + bucket + object lifecycle; S3 protocol for object put/delete (minio-go)
- 42 commands covering VM, storage, networking, billing, monitoring, DNS, projects, support, and more
- Full VM lifecycle: create, start, stop, reboot, reset, tags, change-plan, change-OS, cancel
- VPC lifecycle: create, list, update, restart, ACL list create, VPN gateway create
- VPC tier/subnet creation via `POST /networks` with `type=Vpc`
- Bearer token authentication
- Global `--auto-approve` / `-y` flag for CI/CD automation
- All old STKBILL code removed, zero `/restapi/` references

---

## Planned for next patch

### CLI improvements (no platform dependency)

- [x] `network create` — `--vpc`, `--type`, `--gateway`, `--netmask` flags for VPC tier creation (v0.0.16)
- [x] `network create` — `--acl` flag that resolves ACL name to ID and attaches right after creation (v0.0.16)
- [ ] `portforward create` — add `--public-end-port` and `--private-end-port` flags (API requires them)
- [ ] `instance change-hostname` — fix request body field name (`vm_label` instead of `label`)
- [ ] `region` command — add `use` subcommand to set default region in profile
- [x] Default `--cloud-provider`, `--region`, `--project` via `ZCP_CLOUD_PROVIDER`, `ZCP_REGION`, `ZCP_PROJECT` env vars (v0.0.9)

### Blocked on STKCNSL platform

These features require API endpoints or fixes from the STKCNSL team.

#### Missing DELETE endpoints

The API has no DELETE for these resource types. Resources can only be removed via `billing cancel-service` for VMs/volumes, but not for networking resources.

- [x] `DELETE /vpcs/{slug}` — VPC deletion (live as of 2026-06-11)
- [x] `DELETE /networks/{slug}` — network deletion (live as of 2026-06-11)
- [ ] `DELETE /virtual-routers/{slug}` — virtual router deletion
- [ ] `DELETE /ipaddresses/{slug}` — IP address release
- [x] `DELETE /vpcs/{slug}/network-acl-list/{id}` — ACL list deletion (live; `zcp acl delete` added in v0.0.16)
- [ ] `billing cancel-service` for VPC/Virtual Router service type — currently returns "service not found"

#### ACL rule CRUD — exists, segment is `network-acl` (resolved 2026-06-11)

The rule routes live under `/network-acl` (singular), not `/rules` — confirmed from the
backend route table (`Modules\Vpc NetworkAclController`). A `rules` array embedded in the
ACL-list create POST is still silently ignored; rules must be added one per request after
the list is created.

- [x] `GET /vpcs/{slug}/network-acl-list/{acl_list_id}/network-acl` — list ACL rules (`zcp acl rules`, v0.0.16)
- [x] `POST /vpcs/{slug}/network-acl-list/{acl_list_id}/network-acl` — create ACL rule (`zcp acl create-rule`, v0.0.16)
- [x] `DELETE /vpcs/{slug}/network-acl-list/{acl_list_id}/network-acl/{rule_id}` — delete ACL rule (`zcp acl delete-rule`, v0.0.16)
- [x] `PUT /vpcs/{slug}/network-acl-list/{acl_list_id}/network-acl/{rule_id}` — update ACL rule (`zcp acl update-rule`, v0.0.16; in-place, rule ID preserved)

#### Network create (isolated) — target region

`POST /networks` returns `missing parameter networkofferingid` for the the target region region. The API doesn't expose the network offering field. Likely a region configuration issue.

- [ ] Network offering mapping for the target region

#### DNS provisioning

`POST /dns/domains` returns `cloud_provider_setup: DNS configuration required`. DNS needs admin-side provisioning.

- [ ] DNS enabled for our account/region

#### Network quota

Only 3 VPC tier networks allowed before quota exceeded (CloudStack `vpc.max.networks`, verified 2026-06-11: 4th tier create returns 403).

- [ ] Quota increase for testing

---

## Future (v0.0.9+)

- [ ] Pagination support — `--page`, `--per-page` flags for list commands
- [ ] `--wait` flag on VPC create, volume create (poll until ready)
- [ ] JSON output improvements — consistent envelope stripping
- [ ] Shell completion for dynamic values (region slugs, plan slugs, etc.)
- [ ] `zcp config set` for default cloud-provider, region, project
- [ ] Kubernetes cluster full lifecycle (create works, delete via billing cancel-service)
