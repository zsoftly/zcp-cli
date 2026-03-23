# ZCP CLI Command Taxonomy

**CLI name**: `zcp`
**Base URL**: `https://cloud.zcp.zsoftly.ca/`
**API path prefix**: `/restapi/`

---

## Global Flags

These flags apply to every command in the CLI:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--profile` | string | `default` | Named credential profile to use |
| `--output` | string | `table` | Output format: `table`, `json`, or `yaml` |
| `--api-url` | string | (from profile) | Override the API base URL |
| `--timeout` | duration | `30s` | HTTP request timeout |
| `--debug` | bool | `false` | Print HTTP request/response details to stderr |
| `--no-color` | bool | `false` | Disable terminal color output |

---

## Output Format Conventions

- **table** (default) — human-readable fixed-width columns; intended for interactive use
- **json** — raw JSON response object; suitable for `jq` pipelines and scripting
- **yaml** — YAML rendering of the same response object; suitable for config-driven workflows
- All three formats are fully machine-parseable: no extra prose is written to stdout
- Errors are always written to stderr regardless of `--output`
- Exit codes: `0` = success, `1` = API error or CLI error

---

## Full Command Tree

```
zcp
├── version                        Print CLI version and build info
├── completion <shell>             Generate shell completion script (bash/zsh/fish/powershell)
│
├── profile                        Manage credential profiles
│   ├── add                        Add a new named profile
│   ├── list                       List all saved profiles
│   ├── use                        Set the active default profile
│   ├── delete                     Remove a profile
│   └── show                       Show profile details (redacts secretkey)
│
├── auth                           Authentication utilities
│   └── validate                   Validate that the active profile credentials are accepted
│
├── zone                           Zone operations
│   └── list                       List available zones
│
├── offering                       Service offering catalogues
│   ├── compute                    Compute offering operations
│   │   └── list                   List compute offerings (optionally with pricing)
│   ├── storage                    Storage offering operations
│   │   └── list                   List storage offerings (optionally with pricing)
│   ├── network                    Network offering operations
│   │   └── list                   List network offerings (standard and VPC)
│   └── vpc                        VPC offering operations
│       └── list                   List VPC offerings
│
├── template                       VM template operations
│   └── list                       List available templates
│
├── resource                       Resource availability
│   └── available                  Show available resources by domain
│
├── instance                       VM instance operations
│   ├── list                       List instances
│   ├── status                     Get current power state of an instance
│   ├── create                     Create a new instance
│   ├── delete                     Destroy an instance (--expunge to permanently remove)
│   ├── start                      Start a stopped instance
│   ├── stop                       Stop a running instance (--force for forced shutdown)
│   ├── resize                     Resize an instance (offering, CPU, memory)
│   ├── recover                    Recover a destroyed (but not expunged) instance
│   ├── rename                     Update the display name of an instance
│   ├── reset-ssh-key              Reset the SSH key on an instance
│   ├── attach-network             Attach a network to an instance
│   ├── detach-network             Detach a network from an instance
│   ├── attach-iso                 Attach an ISO image to an instance
│   ├── detach-iso                 Detach an ISO image from an instance
│   ├── networks                   List networks attached to an instance
│   └── passwords                  List saved passwords for an instance
│
├── volume                         Block storage volume operations
│   ├── list                       List volumes
│   ├── create                     Create a new volume (async)
│   ├── delete                     Delete a volume
│   ├── attach                     Attach a volume to an instance (async)
│   ├── detach                     Detach a volume from an instance (async)
│   ├── resize                     Resize a volume (async)
│   └── upload                     Upload a volume from a URL (async)
│
├── snapshot                       Volume snapshot operations
│   ├── list                       List snapshots
│   ├── create                     Create a snapshot of a volume
│   └── delete                     Delete a snapshot
│
├── vm-snapshot                    VM (instance-level) snapshot operations
│   ├── list                       List VM snapshots
│   ├── create                     Create a VM snapshot (async)
│   ├── delete                     Delete a VM snapshot
│   └── revert                     Revert an instance to a VM snapshot (async)
│
├── snapshot-policy                Automated snapshot policy operations
│   ├── list                       List snapshot policies
│   ├── create                     Create a snapshot policy
│   └── delete                     Delete a snapshot policy
│
├── network                        Network operations
│   ├── list                       List networks
│   ├── get                        Get a network by ID
│   ├── create                     Create a network
│   ├── delete                     Delete a network
│   ├── update                     Update network name/description/CIDR
│   ├── restart                    Restart a network
│   ├── replace-acl                Replace the ACL on a network
│   └── change-security-group      Change the security group on a network
│
├── vpc                            VPC operations
│   ├── list                       List VPCs
│   ├── get                        Get a VPC by ID
│   ├── create                     Create a VPC
│   ├── delete                     Delete a VPC
│   ├── update                     Update VPC name/description
│   ├── restart                    Restart a VPC
│   ├── create-network             Create a network inside a VPC
│   └── update-network             Update a VPC network
│
├── acl                            Network ACL operations
│   ├── list                       List network ACLs
│   ├── create                     Create a network ACL
│   └── delete                     Delete a network ACL
│
├── ip                             Public IP address operations
│   ├── list                       List IP addresses
│   ├── acquire                    Acquire a new public IP address
│   ├── release                    Release a public IP address
│   ├── enable-static-nat          Enable static NAT for an IP
│   ├── disable-static-nat         Disable static NAT for an IP
│   ├── enable-vpn-access          Enable remote VPN access on an IP
│   └── disable-vpn-access         Disable remote VPN access on an IP
│
├── firewall                       Firewall rule operations
│   ├── list                       List firewall rules
│   ├── create                     Create a firewall rule
│   └── delete                     Delete a firewall rule
│
├── egress                         Egress rule operations
│   ├── list                       List egress rules
│   ├── create                     Create an egress rule
│   └── delete                     Delete an egress rule
│
├── portforward                    Port forwarding rule operations
│   ├── list                       List port forwarding rules
│   ├── create                     Create a port forwarding rule
│   └── delete                     Delete a port forwarding rule
│
├── loadbalancer                   Load balancer rule operations
│   ├── list                       List load balancer rules
│   ├── create                     Create a load balancer rule
│   ├── update                     Update a load balancer rule
│   └── delete                     Delete a load balancer rule
│
├── internal-lb                    Internal load balancer operations
│   ├── list                       List internal LB rules
│   ├── create                     Create an internal LB
│   ├── assign                     Assign an LB rule to an internal LB
│   └── delete                     Delete an internal LB
│
├── security-group                 Security group operations
│   ├── list                       List security groups
│   ├── create                     Create a security group
│   ├── delete                     Delete a security group
│   ├── add-firewall-rule          Add a firewall (ingress) rule to a security group
│   ├── add-egress-rule            Add an egress rule to a security group
│   ├── add-portforward-rule       Add a port forwarding rule to a security group
│   └── delete-rule                Delete a rule from a security group
│
├── ssh-key                        SSH key operations
│   ├── list                       List SSH keys
│   ├── create                     Register a new SSH key (provide public key)
│   └── delete                     Delete an SSH key
│
├── tag                            Resource tag operations
│   ├── list                       List resource tags
│   ├── create                     Add a tag to a resource
│   └── delete                     Remove a tag from a resource
│
├── vpn                            VPN operations
│   ├── gateway                    VPN gateway operations
│   │   ├── list                   List VPN gateways
│   │   ├── create                 Add a VPN gateway (attached to VPC)
│   │   └── delete                 Delete a VPN gateway
│   ├── connection                 VPN connection (site-to-site) operations
│   │   ├── list                   List VPN connections
│   │   ├── create                 Add a VPN connection
│   │   ├── reset                  Reset a VPN connection
│   │   └── delete                 Delete a VPN connection
│   ├── customer-gateway           VPN customer gateway operations
│   │   ├── list                   List VPN customer gateways
│   │   ├── create                 Add a VPN customer gateway
│   │   ├── update                 Update a VPN customer gateway
│   │   └── delete                 Delete a VPN customer gateway
│   └── user                       VPN remote-access user operations
│       ├── list                   List VPN users
│       ├── create                 Add a VPN user
│       └── delete                 Delete a VPN user
│
├── kubernetes                     Kubernetes cluster operations
│   ├── list                       List Kubernetes clusters
│   ├── nodes                      List nodes in a Kubernetes cluster
│   ├── create                     Create a Kubernetes cluster
│   ├── delete                     Destroy a Kubernetes cluster
│   ├── start                      Start a stopped Kubernetes cluster
│   ├── stop                       Stop a running Kubernetes cluster
│   └── scale                      Scale a Kubernetes cluster (worker count)
│
├── usage                          Usage and consumption reporting
│   ├── list                       List usage consumption records
│   ├── report                     List usage report summaries
│   └── progress                   Show current billing period progress status
│
├── cost                           Cost estimation (read-only, no auth on some)
│   └── estimate                   Subcommands for each resource type cost query
│       ├── compute                Compute offering cost plans
│       ├── storage                Storage offering cost plans
│       ├── network                Network offering costs
│       ├── vpc                    VPC offering costs
│       ├── ip                     IP address cost
│       ├── loadbalancer           Load balancer cost
│       ├── portforward            Port forwarding cost
│       ├── snapshot               Snapshot cost
│       ├── vm-snapshot            VM snapshot cost
│       ├── bandwidth              Bandwidth cost
│       ├── kubernetes             Kubernetes cost
│       ├── object-storage         Object storage cost
│       ├── vpn-user               VPN user cost
│       └── template               Template cost and category info
│
└── admin                          Administrator-only operations
    ├── host                       Physical host operations
    │   └── list                   List physical hosts
    ├── quota                      Resource quota operations
    │   └── show                   Show resource quota limits
    ├── invoice                    Invoice and billing operations
    │   ├── list                   List invoices by client
    │   ├── list-tax-pending       List invoices with pending tax
    │   ├── payment-status         Get payment status of an invoice
    │   ├── update-status          Update invoice payment status
    │   ├── update-tax             Update invoice tax details
    │   ├── generate               Generate an invoice PDF
    │   └── change-cost            Adjust invoice payment amount
    └── user                       User administration
        └── credit-balance         Show credit balance for a user account
```

---

## CLI Group to API Path Mapping

| CLI Group | API Path Prefix | Notes |
|-----------|----------------|-------|
| `zone` | `/restapi/zone/` | |
| `offering compute` | `/restapi/compute/` | |
| `offering storage` | `/restapi/storage/` | |
| `offering network` | `/restapi/networkoffering/` | |
| `offering vpc` | `/restapi/vpcoffering/` | |
| `template` | `/restapi/template/` | |
| `resource` | `/restapi/availableResource/` | |
| `instance` | `/restapi/instance/` | |
| `volume` | `/restapi/volume/` | |
| `snapshot` | `/restapi/snapshot/` | |
| `vm-snapshot` | `/restapi/vmsnapshot/` | |
| `snapshot-policy` | `/restapi/snapshotPolicy/` | |
| `network` | `/restapi/network/` | |
| `vpc` | `/restapi/vpc/` | |
| `acl` | `/restapi/networkacllist/` | |
| `ip` | `/restapi/ipaddress/` | |
| `firewall` | `/restapi/firewallrule/` | |
| `egress` | `/restapi/egressrule/` | |
| `portforward` | `/restapi/portforwardingrule/` | |
| `loadbalancer` | `/restapi/loadbalancerrule/` | |
| `internal-lb` | `/restapi/internallb/` | |
| `security-group` | `/restapi/securitygroup/` | |
| `ssh-key` | `/restapi/sshkey/` | |
| `tag` | `/restapi/resourcetags/` | |
| `vpn gateway` | `/restapi/vpngateway/` | |
| `vpn connection` | `/restapi/vpnconnection/` | |
| `vpn customer-gateway` | `/restapi/vpncustomergateway/` | |
| `vpn user` | `/restapi/vpnuser/` | |
| `kubernetes` | `/restapi/kubernetes/` | |
| `usage` | `/restapi/usage/` | |
| `cost` | `/restapi/costestimate/` | |
| `admin host` | `/restapi/host/` | Admin only |
| `admin quota` | `/restapi/resource-quota/` | Admin only |
| `admin invoice` | `/restapi/invoice/` | Admin only |
| `admin user` | `/restapi/user/` | Admin only |
| (internal) | `/restapi/asyncjob/` | Used internally for job polling |

---

## Phase Assignment by CLI Group

### Phase 1 — Read-Only Discovery (Current Build)

These commands are the initial scope. All are read-only `list`/`get`/`status` operations.

| CLI Group | Commands | API Endpoints |
|-----------|----------|---------------|
| `zone` | `list` | `GET /restapi/zone/zonelist` |
| `offering compute` | `list` | `GET /restapi/compute/computeOfferingList`, `computeOfferingListWithPrice` |
| `offering storage` | `list` | `GET /restapi/storage/storageOfferingList`, `storageOfferingListWithPrice` |
| `offering network` | `list` | `GET /restapi/networkoffering/networkOfferingList`, `vpcNetworkOfferingList` |
| `offering vpc` | `list` | `GET /restapi/vpcoffering/vpcOfferingList` |
| `template` | `list` | `GET /restapi/template/templateList` |
| `resource` | `available` | `GET /restapi/availableResource/getAvailableResourceByDomain` |
| `instance` | `list`, `status` | `GET /restapi/instance/instanceList`, `vmStatus` |
| `instance` | `networks` | `GET /restapi/instance/instanceNetworkList` |
| `volume` | `list` | `GET /restapi/volume/volumeList` |
| `snapshot` | `list` | `GET /restapi/snapshot/snapshotList` |
| `vm-snapshot` | `list` | `GET /restapi/vmsnapshot/vmsnapshotList` |
| `snapshot-policy` | `list` | `GET /restapi/snapshotPolicy/snapshotPolicyList` |
| `network` | `list`, `get` | `GET /restapi/network/networkList`, `networkId` |
| `vpc` | `list`, `get` | `GET /restapi/vpc/vpcList`, `vpcId` |
| `acl` | `list` | `GET /restapi/networkacllist/networkAclList` |
| `ip` | `list` | `GET /restapi/ipaddress/ipAddressList` |
| `firewall` | `list` | `GET /restapi/firewallrule/firewallRuleList` |
| `egress` | `list` | `GET /restapi/egressrule/egressRuleList` |
| `portforward` | `list` | `GET /restapi/portforwardingrule/portForwardingRuleList` |
| `loadbalancer` | `list` | `GET /restapi/loadbalancerrule/loadBalancerRuleList` |
| `internal-lb` | `list` | `GET /restapi/internallb/internalLbList` |
| `security-group` | `list` | `GET /restapi/securitygroup/securityList` |
| `ssh-key` | `list` | `GET /restapi/sshkey/sshkeyList` |
| `tag` | `list` | `GET /restapi/resourcetags/resourceTagsList` |
| `vpn gateway` | `list` | `GET /restapi/vpngateway/vpnGatewayList` |
| `vpn connection` | `list` | `GET /restapi/vpnconnection/vpnConnectionList` |
| `vpn customer-gateway` | `list` | `GET /restapi/vpncustomergateway/vpnCustomerGatewayList` |
| `vpn user` | `list` | `GET /restapi/vpnuser/vpnUserlist` |
| `kubernetes` | `list`, `nodes` | `GET /restapi/kubernetes/listCluster`, `listNodes` |
| `admin host` | `list` | `GET /restapi/host/hostList` |
| `admin quota` | `show` | `GET /restapi/resource-quota/get-resource-limit` |

### Phase 2 — Instance Lifecycle, Volume, Network CRUD

Mutating operations on core compute and networking resources. Includes async operations with job polling.

| CLI Group | Commands | Notes |
|-----------|----------|-------|
| `instance` | `create`, `delete`, `start`, `stop`, `resize`, `recover`, `rename`, `reset-ssh-key`, `attach-network`, `detach-network`, `attach-iso`, `detach-iso`, `passwords` | |
| `volume` | `create`, `delete`, `attach`, `detach`, `resize` | `create`, `attach`, `detach`, `resize` are async |
| `snapshot` | `create`, `delete` | |
| `vm-snapshot` | `create`, `delete`, `revert` | `create`, `revert` are async |
| `snapshot-policy` | `create`, `delete` | |
| `network` | `create`, `delete`, `update`, `restart`, `replace-acl`, `change-security-group` | |
| `vpc` | `create`, `delete`, `update`, `restart`, `create-network`, `update-network` | |
| `acl` | `create`, `delete` | |
| `ip` | `acquire`, `release`, `enable-static-nat`, `disable-static-nat`, `enable-vpn-access`, `disable-vpn-access` | |
| `firewall` | `create`, `delete` | |
| `egress` | `create`, `delete` | |
| `portforward` | `create`, `delete` | |
| `loadbalancer` | `create`, `update`, `delete` | |
| `internal-lb` | `create`, `assign`, `delete` | |
| `security-group` | `create`, `delete`, `add-firewall-rule`, `add-egress-rule`, `add-portforward-rule`, `delete-rule` | |
| `ssh-key` | `create`, `delete` | |
| `tag` | `create`, `delete` | |
| `vpn gateway` | `create`, `delete` | |
| `vpn connection` | `create`, `reset`, `delete` | |
| `vpn customer-gateway` | `create`, `update`, `delete` | |
| `vpn user` | `create`, `delete` | |
| `kubernetes` | `create`, `delete`, `start`, `stop`, `scale` | |

### Phase 3 — Advanced, Ancillary, and Admin

Cost estimation, usage reporting, ISO/upload operations, and all admin billing/invoice operations.

| CLI Group | Commands | Notes |
|-----------|----------|-------|
| `instance` | `attach-iso`, `detach-iso` | Moved from P2 for prioritization |
| `volume` | `upload` | Async, requires URL and checksum |
| `usage` | `list`, `report`, `progress` | May require admin for sub-domain variant |
| `cost` | All `estimate` subcommands | Read-only pricing queries |
| `admin invoice` | `list`, `list-tax-pending`, `payment-status`, `update-status`, `update-tax`, `generate`, `change-cost` | Admin only |
| `admin user` | `credit-balance` | Admin only |

---

## Async Operation Handling

Commands that trigger async API operations must poll for completion:

1. Issue the mutating request; the response item contains a `jobId` field
2. Poll `GET /restapi/asyncjob/resourceStatus?jobId=<id>` until `status` is `SUCCEEDED` or `FAILED`
3. On `FAILED`, surface `errorCode` and `errorMessage` to the user

**CLI behavior flags for async commands:**
- Default: poll with a spinner until completion, then print result
- `--no-wait`: return immediately after the initial request and print the `jobId`
- `--timeout`: maximum time to wait before giving up (inherits global `--timeout` default)

**Async commands (Phase 2):**
- `volume create`, `volume attach`, `volume detach`, `volume resize`, `volume upload`
- `vm-snapshot create`, `vm-snapshot revert`

---

## Notes on API Quirks

- Several mutating operations use `GET` instead of `POST`/`DELETE` (e.g., `startInstance`, `stopInstance`, `destroyInstance`, `attachVolume`). The CLI must preserve these as-is; they cannot be changed to the semantically correct HTTP methods.
- `GET /restapi/instance/destroyInstance` accepts an `expunge` query param (`true`/`false`) to control whether the instance is permanently deleted or placed in a recoverable destroyed state.
- `DELETE /restapi/vpnuser/deleteVpnUser` does not take a `{uuid}` path param — the user identifier is passed as a query parameter instead.
- `GET /restapi/kubernetes/listCluster` returns an inline schema with no named `$ref` in the spec; the actual cluster list structure must be confirmed from a live response.
- `GET /restapi/costestimate/getpublickey` requires no auth headers; it is a public endpoint.
