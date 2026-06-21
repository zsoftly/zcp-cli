# zcp v0.0.19 Release Notes

## Load balancers create cleanly, and instances are addressable by ID or name

This release fixes `loadbalancer create` (it never worked — the API requires an initial
rule), and makes every instance command accept the VM's **ID, name, or slug**, not just its
slug. It also paginates the instance list so large accounts see all of their VMs.

Highlights:

- **`zcp loadbalancer create` works** — it now sends a required first rule
  (`--public-port`/`--private-port`/`--algorithm`) and can attach back-ends with `--vm`.
- **Refer to an instance by ID, name, or slug** in every `instance` subcommand, with a clear
  error when a name is ambiguous.
- **`instance list` shows the `ID` column** and returns **all** VMs (the list is now paginated).
- **`reboot` refuses a non-`Running` VM** instead of silently no-op'ing.
- **Manage account access control** — new `zcp sub-user`, `zcp role`, and `zcp permission`
  commands for creating sub-users, defining roles from permissions, and blocking/unblocking access.

---

## Fixed

### `zcp loadbalancer create` always failed

The request sent an empty `rules` array, which the API rejects — a load balancer must be created
with at least one rule. Create now builds that first rule from new flags:

```bash
zcp loadbalancer create --name my-lb --network <network-slug> --ip <ip-slug> \
  --billing-cycle hourly \
  --public-port 80 --private-port 8080 --algorithm roundrobin \
  --vm web-1 --vm web-2          # optional: attach back-ends

# add more rules later
zcp loadbalancer create-rule <lb-slug> --name api-rule \
  --public-port 8443 --private-port 443 --protocol tcp --algorithm leastconn
```

`--public-port`, `--private-port`, and `--algorithm` are **required** (the rule can't be formed
without them). `--protocol` defaults to `tcp`, `--rule-name` defaults to `<lb-name>-rule`, and
`--sticky-method`, `--enable-tls`, and `--enable-proxy-protocol` are optional.

### `instance list` only returned the first page of VMs

The `/virtual-machines` endpoint is paginated, but the CLI fetched a single page — accounts with
more VMs than fit on one page silently lost the rest. The list now walks every page, which also
makes instance reference resolution (below) reliable.

## Added

### Address an instance by ID, name, or slug

Every `instance` subcommand — `get`, `start`, `stop`, `reboot`, `reset`, `delete`, `logs`, `ssh`,
`tag-*`, `change-*`, `add-network`, `addons`, `purchase-addon` — now accepts any unique reference
to the VM:

```bash
zcp instance reboot vm-1a2b3c        # by ID (vm_id)
zcp instance reboot my-web-server    # by name
zcp instance reboot my-web-server-1  # by slug
```

Match order is ID/`vm_id`, then name, then slug. If a name matches two VMs, the command lists the
matching IDs and asks you to pick one. Resolution checks your active region/project first and
**falls back to an unscoped lookup** when the reference isn't found there, so a globally-unique ID
or slug still works without `--region`.

### `ID` column in instance output

`zcp instance list` and `zcp instance get` now show the instance `ID` (the value to copy for the
references above). `-o json`/`-o yaml` and `--debug` expand to the full set of columns.

### Manage sub-users, roles, and permissions

Account access control is now scriptable. These are account-level commands — no `--region`/`--project`
needed.

```bash
# Permissions: the read-only catalog you build roles from
zcp permission list
zcp permission list --category "Virtual Machine"

# Roles: group permissions, then assign to sub-users
zcp role list
zcp role get service-administrator                 # shows its permissions + assigned users
zcp role create --name "VM Operator" \
  --permission virtual-machine-read --permission virtual-machine-manage
zcp role update vm-operator --permission virtual-machine-read --permission dns-read
zcp role delete vm-operator

# Sub-users: additional users under your account (addressable by id OR email)
zcp sub-user create --name "Jane Doe" --email jane@yourco.com \
  --password 'S3cret!pass' --role service-viewer --project default-9
zcp sub-user update jane@yourco.com --role service-administrator
zcp sub-user block jane@yourco.com                 # revoke access without deleting
zcp sub-user unblock jane@yourco.com
zcp sub-user delete jane@yourco.com
```

Notes: `--permission` on a role **replaces** the role's full set (it isn't additive), and `role update`
preserves any flags you don't pass. The predefined `owner`, `service-administrator`, and
`service-viewer` roles can't be edited or deleted. Sub-user `--email` must be a company address,
`--password` needs 8+ chars with mixed case, a number, and a symbol, and newly created sub-users start
**blocked** until you `unblock` them.

## Changed

- **`zcp instance reboot` refuses a VM that isn't `Running`**, e.g.
  `instance "my-vm" is Stopped; it must be Running before it can be rebooted`, instead of issuing a
  reboot the platform silently ignores.
- **`zcp loadbalancer list` and `zcp instance list` emit full objects for `-o json`/`-o yaml`**
  rather than a flattened, all-string copy of the table — automation gets every field.
- **`zcp auth validate` honors `ZCP_DEBUG`** like every other command.
- **Documentation URL** is now `https://docs.zcp.zsoftly.ca`.

## Upgrade notes

`zcp loadbalancer create` now **requires** `--public-port`, `--private-port`, and `--algorithm`.
Scripts that called `create` without them will need to add these flags (the command previously
failed at the API anyway). Everything else is additive — existing slug-based instance commands keep
working unchanged.
