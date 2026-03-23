# ZCP API Inventory

**Base URL**: `https://cloud.zcp.zsoftly.ca/`
**Spec Version**: OpenAPI 3.0.1
**Total Endpoints**: 166
**Auth**: `apikey` and `secretkey` HTTP request headers on every call

---

## Endpoint Table

| # | Path | Method | Summary | CLI Group | Phase | Scope | Async? |
|---|------|--------|---------|-----------|-------|-------|--------|
| 1 | `/restapi/asyncjob/resourceStatus` | GET | Resource Status | (internal) | 1 | Customer | — |
| 2 | `/restapi/availableResource/getAvailableResourceByDomain` | GET | Resources Availability | `resource` | 1 | Customer | No |
| 3 | `/restapi/compute/computeOfferingList` | GET | Compute Offering List | `offering compute` | 1 | Customer | No |
| 4 | `/restapi/compute/computeOfferingListWithPrice` | GET | Compute Offering With Price | `offering compute` | 1 | Customer | No |
| 5 | `/restapi/costestimate/additional-template-resize-cost` | GET | Additional Template Resize Cost | `cost` | 3 | Customer | No |
| 6 | `/restapi/costestimate/bandwidth-cost` | GET | Bandwidth Cost | `cost` | 3 | Customer | No |
| 7 | `/restapi/costestimate/compute-category-list` | GET | Compute Category List | `cost` | 3 | Customer | No |
| 8 | `/restapi/costestimate/compute-plan-list` | GET | Compute Offering Cost | `cost` | 3 | Customer | No |
| 9 | `/restapi/costestimate/compute-plan-types` | GET | Compute Plan Types | `cost` | 3 | Customer | No |
| 10 | `/restapi/costestimate/getpublickey` | GET | Get Public Key | `cost` | 3 | Public | No |
| 11 | `/restapi/costestimate/ip-cost` | GET | IP Address Cost | `cost` | 3 | Customer | No |
| 12 | `/restapi/costestimate/k8s-cost` | GET | Kubernetes Cost | `cost` | 3 | Customer | No |
| 13 | `/restapi/costestimate/kubernetes-version-list` | GET | Kubernetes Version List | `cost` | 3 | Customer | No |
| 14 | `/restapi/costestimate/list-all-support-category` | GET | Support Categories | `cost` | 3 | Customer | No |
| 15 | `/restapi/costestimate/list-all-support-plans` | GET | Support Plans | `cost` | 3 | Customer | No |
| 16 | `/restapi/costestimate/list-sfs-storage-offerings` | GET | List SFS Storage Offerings | `cost` | 3 | Customer | No |
| 17 | `/restapi/costestimate/loadbalancer-cost` | GET | Load Balancer Cost | `cost` | 3 | Customer | No |
| 18 | `/restapi/costestimate/multicurrency` | GET | Multi-Currency List | `cost` | 3 | Customer | No |
| 19 | `/restapi/costestimate/networkoffering-list` | GET | Network Offering List (cost) | `cost` | 3 | Customer | No |
| 20 | `/restapi/costestimate/object-storage-cost` | GET | Object Storage Cost | `cost` | 3 | Customer | No |
| 21 | `/restapi/costestimate/portforwarding-cost` | GET | Port Forwarding Cost | `cost` | 3 | Customer | No |
| 22 | `/restapi/costestimate/service-list` | GET | Service List | `cost` | 3 | Customer | No |
| 23 | `/restapi/costestimate/snapshot-cost` | GET | Snapshot Cost | `cost` | 3 | Customer | No |
| 24 | `/restapi/costestimate/storage-category-list` | GET | Storage Category List | `cost` | 3 | Customer | No |
| 25 | `/restapi/costestimate/storage-plan-list` | GET | Storage Plan List | `cost` | 3 | Customer | No |
| 26 | `/restapi/costestimate/tax` | GET | Tax | `cost` | 3 | Customer | No |
| 27 | `/restapi/costestimate/template-category-list` | GET | Template Category List | `cost` | 3 | Customer | No |
| 28 | `/restapi/costestimate/template-distribution-list` | GET | Template Distribution List | `cost` | 3 | Customer | No |
| 29 | `/restapi/costestimate/template-list` | GET | Template List (cost) | `cost` | 3 | Customer | No |
| 30 | `/restapi/costestimate/template-platform-list` | GET | Template Platform List | `cost` | 3 | Customer | No |
| 31 | `/restapi/costestimate/vm-scheduler-cost` | GET | VM Scheduler Cost | `cost` | 3 | Customer | No |
| 32 | `/restapi/costestimate/vm-snapshot-cost` | GET | VM Snapshot Cost | `cost` | 3 | Customer | No |
| 33 | `/restapi/costestimate/vpcoffering-list` | GET | VPC Offering List (cost) | `cost` | 3 | Customer | No |
| 34 | `/restapi/costestimate/vpn-user-cost` | GET | VPN User Cost | `cost` | 3 | Customer | No |
| 35 | `/restapi/costestimate/zone-list` | GET | Zone List (cost) | `cost` | 3 | Customer | No |
| 36 | `/restapi/egressrule/createEgressRule` | POST | Create Egress Rule | `egress` | 2 | Customer | No |
| 37 | `/restapi/egressrule/deleteEgressRule/{uuid}` | DELETE | Delete Egress Rule | `egress` | 2 | Customer | No |
| 38 | `/restapi/egressrule/egressRuleList` | GET | Egress Rule List | `egress` | 1 | Customer | No |
| 39 | `/restapi/firewallrule/createFirewallRule` | POST | Create Firewall Rule | `firewall` | 2 | Customer | No |
| 40 | `/restapi/firewallrule/deleteFirewallRule/{uuid}` | DELETE | Delete Firewall Rule | `firewall` | 2 | Customer | No |
| 41 | `/restapi/firewallrule/firewallRuleList` | GET | Firewall Rule List | `firewall` | 1 | Customer | No |
| 42 | `/restapi/host/hostList` | GET | Host List | `admin host` | 1 | Admin | No |
| 43 | `/restapi/instance/attachIso` | GET | Attach ISO to Instance | `instance` | 3 | Customer | No |
| 44 | `/restapi/instance/attachNetwork` | POST | Attach Network to Instance | `instance` | 2 | Customer | No |
| 45 | `/restapi/instance/createInstance` | POST | Create Instance | `instance` | 2 | Customer | No |
| 46 | `/restapi/instance/destroyInstance` | GET | Destroy Instance | `instance` | 2 | Customer | No |
| 47 | `/restapi/instance/detachIso` | GET | Detach ISO | `instance` | 3 | Customer | No |
| 48 | `/restapi/instance/detachNetwork` | POST | Detach Network from Instance | `instance` | 2 | Customer | No |
| 49 | `/restapi/instance/instanceList` | GET | Instance List | `instance` | 1 | Customer | No |
| 50 | `/restapi/instance/instanceNetworkList` | GET | Instance Network List | `instance` | 1 | Customer | No |
| 51 | `/restapi/instance/instancePasswordList` | GET | Instance Password List | `instance` | 2 | Customer | No |
| 52 | `/restapi/instance/recoverVm` | GET | Recover VM Instance | `instance` | 2 | Customer | No |
| 53 | `/restapi/instance/resetSSHkey` | GET | Reset Instance SSH Key | `instance` | 2 | Customer | No |
| 54 | `/restapi/instance/resizeVm` | GET | Resize VM Instance | `instance` | 2 | Customer | No |
| 55 | `/restapi/instance/startInstance` | GET | Start Instance | `instance` | 2 | Customer | No |
| 56 | `/restapi/instance/stopInstance` | GET | Stop Instance | `instance` | 2 | Customer | No |
| 57 | `/restapi/instance/updateInstanceName` | PUT | Update Instance Name | `instance` | 2 | Customer | No |
| 58 | `/restapi/instance/vmStatus` | GET | VM Status | `instance` | 1 | Customer | No |
| 59 | `/restapi/internallb/assignLbRule` | GET | Assign Internal LB Rule | `internal-lb` | 2 | Customer | No |
| 60 | `/restapi/internallb/createInternalLb` | POST | Create Internal LB | `internal-lb` | 2 | Customer | No |
| 61 | `/restapi/internallb/deleteInternalLb/{uuid}` | DELETE | Delete Internal LB | `internal-lb` | 2 | Customer | No |
| 62 | `/restapi/internallb/internalLbList` | GET | Internal LB List | `internal-lb` | 1 | Customer | No |
| 63 | `/restapi/invoice/changeInvoiceCost` | GET | Change Invoice Payment Cost | `admin invoice` | 3 | Admin | No |
| 64 | `/restapi/invoice/generateInvoice` | GET | Generate Invoice | `admin invoice` | 3 | Admin | No |
| 65 | `/restapi/invoice/getInvoicePaymentStatus` | GET | Invoice Payment Status | `admin invoice` | 3 | Admin | No |
| 66 | `/restapi/invoice/listByClient` | GET | Invoice List by Client | `admin invoice` | 3 | Admin | No |
| 67 | `/restapi/invoice/listTaxPendingInvoice` | GET | Tax Pending Invoice List | `admin invoice` | 3 | Admin | No |
| 68 | `/restapi/invoice/updateInvoiceStatus` | POST | Update Invoice Status | `admin invoice` | 3 | Admin | No |
| 69 | `/restapi/invoice/updateInvoiceTax` | POST | Update Invoice Tax | `admin invoice` | 3 | Admin | No |
| 70 | `/restapi/ipaddress/acquireIpAddress` | GET | Acquire IP Address | `ip` | 2 | Customer | No |
| 71 | `/restapi/ipaddress/disableStaticNat` | DELETE | Disable Static NAT | `ip` | 2 | Customer | No |
| 72 | `/restapi/ipaddress/disableremotevpnaccess` | DELETE | Disable Remote VPN Access | `ip` | 2 | Customer | No |
| 73 | `/restapi/ipaddress/enableStaticNat` | POST | Enable Static NAT | `ip` | 2 | Customer | No |
| 74 | `/restapi/ipaddress/enableremotevpnaccess` | GET | Enable Remote VPN Access | `ip` | 2 | Customer | No |
| 75 | `/restapi/ipaddress/ipAddressList` | GET | IP Address List | `ip` | 1 | Customer | No |
| 76 | `/restapi/ipaddress/releaseIpAddress` | DELETE | Release IP Address | `ip` | 2 | Customer | No |
| 77 | `/restapi/kubernetes/createKubernetes` | POST | Create Kubernetes Cluster | `kubernetes` | 2 | Customer | No |
| 78 | `/restapi/kubernetes/destroyKubernetes` | DELETE | Destroy Kubernetes Cluster | `kubernetes` | 2 | Customer | No |
| 79 | `/restapi/kubernetes/listCluster` | GET | List Kubernetes Clusters | `kubernetes` | 1 | Customer | No |
| 80 | `/restapi/kubernetes/listNodes` | GET | List Kubernetes Nodes | `kubernetes` | 1 | Customer | No |
| 81 | `/restapi/kubernetes/scaleKubernetes` | PUT | Scale Kubernetes Cluster | `kubernetes` | 2 | Customer | No |
| 82 | `/restapi/kubernetes/startKubernetes` | PUT | Start Kubernetes Cluster | `kubernetes` | 2 | Customer | No |
| 83 | `/restapi/kubernetes/stopKubernetes` | PUT | Stop Kubernetes Cluster | `kubernetes` | 2 | Customer | No |
| 84 | `/restapi/loadbalancerrule/createLoadBalancerRule` | POST | Create Load Balancer Rule | `loadbalancer` | 2 | Customer | No |
| 85 | `/restapi/loadbalancerrule/deleteLoadBalancerRule/{uuid}` | DELETE | Delete Load Balancer Rule | `loadbalancer` | 2 | Customer | No |
| 86 | `/restapi/loadbalancerrule/loadBalancerRuleList` | GET | Load Balancer Rule List | `loadbalancer` | 1 | Customer | No |
| 87 | `/restapi/loadbalancerrule/updateLoadBalancerRule` | PUT | Update Load Balancer Rule | `loadbalancer` | 2 | Customer | No |
| 88 | `/restapi/network/changeSecurityGroup` | GET | Change Network Security Group | `network` | 2 | Customer | No |
| 89 | `/restapi/network/createNetwork` | POST | Create Network | `network` | 2 | Customer | No |
| 90 | `/restapi/network/deleteNetwork/{uuid}` | DELETE | Delete Network | `network` | 2 | Customer | No |
| 91 | `/restapi/network/networkId` | GET | Get Network by ID | `network` | 1 | Customer | No |
| 92 | `/restapi/network/networkList` | GET | Network List | `network` | 1 | Customer | No |
| 93 | `/restapi/network/replaceAcl` | GET | Replace Network ACL | `network` | 2 | Customer | No |
| 94 | `/restapi/network/restartNetwork` | GET | Restart Network | `network` | 2 | Customer | No |
| 95 | `/restapi/network/updateNetwork` | PUT | Update Network | `network` | 2 | Customer | No |
| 96 | `/restapi/networkacllist/createNetworkAcl` | POST | Create Network ACL | `acl` | 2 | Customer | No |
| 97 | `/restapi/networkacllist/deleteNetworkAcl/{uuid}` | DELETE | Delete Network ACL | `acl` | 2 | Customer | No |
| 98 | `/restapi/networkacllist/networkAclList` | GET | Network ACL List | `acl` | 1 | Customer | No |
| 99 | `/restapi/networkoffering/networkOfferingList` | GET | Network Offering List | `offering network` | 1 | Customer | No |
| 100 | `/restapi/networkoffering/vpcNetworkOfferingList` | GET | VPC Network Offering List | `offering network` | 1 | Customer | No |
| 101 | `/restapi/portforwardingrule/createPortForwardingRule` | POST | Create Port Forwarding Rule | `portforward` | 2 | Customer | No |
| 102 | `/restapi/portforwardingrule/deletePortForwardingRule/{uuid}` | DELETE | Delete Port Forwarding Rule | `portforward` | 2 | Customer | No |
| 103 | `/restapi/portforwardingrule/portForwardingRuleList` | GET | Port Forwarding Rule List | `portforward` | 1 | Customer | No |
| 104 | `/restapi/resource-quota/get-resource-limit` | GET | Get Resource Quota Limits | `admin quota` | 1 | Admin | No |
| 105 | `/restapi/resourcetags/createTags` | POST | Create Resource Tags | `tag` | 2 | Customer | No |
| 106 | `/restapi/resourcetags/deleteResourceTag/{uuid}` | DELETE | Delete Resource Tag | `tag` | 2 | Customer | No |
| 107 | `/restapi/resourcetags/resourceTagsList` | GET | Resource Tags List | `tag` | 1 | Customer | No |
| 108 | `/restapi/securitygroup/createSecurityGroup` | POST | Create Security Group | `security-group` | 2 | Customer | No |
| 109 | `/restapi/securitygroup/createSecurityGroupEgressRule` | POST | Create SG Egress Rule | `security-group` | 2 | Customer | No |
| 110 | `/restapi/securitygroup/createSecurityGroupFirewallRule` | POST | Create SG Firewall Rule | `security-group` | 2 | Customer | No |
| 111 | `/restapi/securitygroup/createSecurityGroupPortForwardingRule` | POST | Create SG Port Forwarding Rule | `security-group` | 2 | Customer | No |
| 112 | `/restapi/securitygroup/deleteSecurityGroup/{uuid}` | DELETE | Delete Security Group | `security-group` | 2 | Customer | No |
| 113 | `/restapi/securitygroup/deleteSecurityGroupRule` | DELETE | Delete Security Group Rule | `security-group` | 2 | Customer | No |
| 114 | `/restapi/securitygroup/securityList` | GET | Security Group List | `security-group` | 1 | Customer | No |
| 115 | `/restapi/snapshot/createSnapshot` | POST | Create Snapshot | `snapshot` | 2 | Customer | No |
| 116 | `/restapi/snapshot/deleteSnapshot/{uuid}` | DELETE | Delete Snapshot | `snapshot` | 2 | Customer | No |
| 117 | `/restapi/snapshot/snapshotList` | GET | Snapshot List | `snapshot` | 1 | Customer | No |
| 118 | `/restapi/snapshotPolicy/createSnapshotPolicy` | POST | Create Snapshot Policy | `snapshot-policy` | 2 | Customer | No |
| 119 | `/restapi/snapshotPolicy/deleteSnapshotPolicy/{uuid}` | DELETE | Delete Snapshot Policy | `snapshot-policy` | 2 | Customer | No |
| 120 | `/restapi/snapshotPolicy/snapshotPolicyList` | GET | Snapshot Policy List | `snapshot-policy` | 1 | Customer | No |
| 121 | `/restapi/sshkey/createSSHkey` | POST | Create SSH Key | `ssh-key` | 2 | Customer | No |
| 122 | `/restapi/sshkey/deleteSSHkey/{uuid}` | DELETE | Delete SSH Key | `ssh-key` | 2 | Customer | No |
| 123 | `/restapi/sshkey/sshkeyList` | GET | SSH Key List | `ssh-key` | 1 | Customer | No |
| 124 | `/restapi/storage/storageOfferingList` | GET | Storage Offering List | `offering storage` | 1 | Customer | No |
| 125 | `/restapi/storage/storageOfferingListWithPrice` | GET | Storage Offering With Price | `offering storage` | 1 | Customer | No |
| 126 | `/restapi/template/templateList` | GET | Template List | `template` | 1 | Customer | No |
| 127 | `/restapi/usage/usageConsumptionList` | GET | Usage Consumption List | `usage` | 3 | Customer | No |
| 128 | `/restapi/usage/usageConsumptionListWithSubDomain` | GET | Usage Consumption With Sub-Domain | `usage` | 3 | Customer | No |
| 129 | `/restapi/usage/usageProgressStatus` | GET | Usage Progress Status | `usage` | 3 | Customer | No |
| 130 | `/restapi/usage/usageReportList` | GET | Usage Report List | `usage` | 3 | Customer | No |
| 131 | `/restapi/user/creditBalance` | GET | User Credit Balance | `admin user` | 3 | Admin | No |
| 132 | `/restapi/vmsnapshot/createVmSnapshot` | POST | Create VM Snapshot | `vm-snapshot` | 2 | Customer | Yes |
| 133 | `/restapi/vmsnapshot/deleteVmSnapshot/{uuid}` | DELETE | Delete VM Snapshot | `vm-snapshot` | 2 | Customer | No |
| 134 | `/restapi/vmsnapshot/revertToVmSnapshot` | GET | Revert to VM Snapshot | `vm-snapshot` | 2 | Customer | Yes |
| 135 | `/restapi/vmsnapshot/vmsnapshotList` | GET | VM Snapshot List | `vm-snapshot` | 1 | Customer | No |
| 136 | `/restapi/volume/attachVolume` | GET | Attach Volume | `volume` | 2 | Customer | Yes |
| 137 | `/restapi/volume/createVolume` | POST | Create Volume | `volume` | 2 | Customer | Yes |
| 138 | `/restapi/volume/deleteVolume/{uuid}` | DELETE | Delete Volume | `volume` | 2 | Customer | No |
| 139 | `/restapi/volume/detachVolume` | GET | Detach Volume | `volume` | 2 | Customer | Yes |
| 140 | `/restapi/volume/resizeVolume` | GET | Resize Volume | `volume` | 2 | Customer | Yes |
| 141 | `/restapi/volume/uploadVolume` | POST | Upload Volume | `volume` | 3 | Customer | Yes |
| 142 | `/restapi/volume/volumeList` | GET | Volume List | `volume` | 1 | Customer | No |
| 143 | `/restapi/vpc/createVpc` | POST | Create VPC | `vpc` | 2 | Customer | No |
| 144 | `/restapi/vpc/createVpcNetwork` | POST | Create VPC Network | `vpc` | 2 | Customer | No |
| 145 | `/restapi/vpc/deleteVpc/{uuid}` | DELETE | Delete VPC | `vpc` | 2 | Customer | No |
| 146 | `/restapi/vpc/restartVpc` | GET | Restart VPC | `vpc` | 2 | Customer | No |
| 147 | `/restapi/vpc/updateVpc` | PUT | Update VPC | `vpc` | 2 | Customer | No |
| 148 | `/restapi/vpc/updateVpcNetwork` | PUT | Update VPC Network | `vpc` | 2 | Customer | No |
| 149 | `/restapi/vpc/vpcId` | GET | Get VPC by ID | `vpc` | 1 | Customer | No |
| 150 | `/restapi/vpc/vpcList` | GET | VPC List | `vpc` | 1 | Customer | No |
| 151 | `/restapi/vpcoffering/vpcOfferingList` | GET | VPC Offering List | `offering vpc` | 1 | Customer | No |
| 152 | `/restapi/vpnconnection/addVpnConnection` | POST | Add VPN Connection | `vpn connection` | 2 | Customer | No |
| 153 | `/restapi/vpnconnection/deleteVpnConnection/{uuid}` | DELETE | Delete VPN Connection | `vpn connection` | 2 | Customer | No |
| 154 | `/restapi/vpnconnection/resetVpnConnection/{uuid}` | PUT | Reset VPN Connection | `vpn connection` | 2 | Customer | No |
| 155 | `/restapi/vpnconnection/vpnConnectionList` | GET | VPN Connection List | `vpn connection` | 1 | Customer | No |
| 156 | `/restapi/vpncustomergateway/addVpnCustomerGateway` | POST | Add VPN Customer Gateway | `vpn customer-gateway` | 2 | Customer | No |
| 157 | `/restapi/vpncustomergateway/deleteVpnCustomerGateway/{uuid}` | DELETE | Delete VPN Customer Gateway | `vpn customer-gateway` | 2 | Customer | No |
| 158 | `/restapi/vpncustomergateway/updateVpnCustomerGateway` | PUT | Update VPN Customer Gateway | `vpn customer-gateway` | 2 | Customer | No |
| 159 | `/restapi/vpncustomergateway/vpnCustomerGatewayList` | GET | VPN Customer Gateway List | `vpn customer-gateway` | 1 | Customer | No |
| 160 | `/restapi/vpngateway/addVpnGateway` | POST | Add VPN Gateway | `vpn gateway` | 2 | Customer | No |
| 161 | `/restapi/vpngateway/deleteVpnGateway/{uuid}` | DELETE | Delete VPN Gateway | `vpn gateway` | 2 | Customer | No |
| 162 | `/restapi/vpngateway/vpnGatewayList` | GET | VPN Gateway List | `vpn gateway` | 1 | Customer | No |
| 163 | `/restapi/vpnuser/addVpnUser` | POST | Add VPN User | `vpn user` | 2 | Customer | No |
| 164 | `/restapi/vpnuser/deleteVpnUser` | DELETE | Delete VPN User | `vpn user` | 2 | Customer | No |
| 165 | `/restapi/vpnuser/vpnUserlist` | GET | VPN User List | `vpn user` | 1 | Customer | No |
| 166 | `/restapi/zone/zonelist` | GET | Zone List | `zone` | 1 | Customer | No |

**Phase key:**
- Phase 1 — Read-only discovery: list, get, status operations (building now)
- Phase 2 — Instance lifecycle, volume, network, and standard CRUD operations
- Phase 3 — Advanced/ancillary: cost estimates, usage reporting, ISO, upload, admin billing

**Async key:**
- `Yes` — Response object contains a `jobId` field; poll `/restapi/asyncjob/resourceStatus?jobId=<id>` for completion
- `No` — Operation returns the final result synchronously

---

## API Response Envelope Patterns

### List Response

All list endpoints return a consistent two-field envelope:

```json
{
  "count": 3,
  "<listFieldName>": [ ... ]
}
```

The list field name matches the schema and operation, for example:

| Endpoint family | List field name |
|----------------|-----------------|
| instance | `listInstanceResponse` |
| volume | `listVolumeResponse` |
| network | `listNetworkResponse` |
| vpc | `listVpcResponse` |
| snapshot | `listSnapShotResponse` |
| vmsnapshot | `listVmSnapshotResponse` |
| securitygroup | `listSecurityGroupResponse` |
| sshkey | `listSSHKeyResponse` |
| ipaddress | `listIpAddressResponse` |
| loadbalancerrule | `listLoadBalancerRuleResponse` |
| portforwardingrule | `listPortForwardingResponse` |
| firewallrule | `listFirewallRuleResponse` |
| egressrule | `listEgressRuleResponse` |
| internallb | `listInternalLbResponse` |
| networkacllist | `listNetworkAclListResponse` |
| vpnconnection | `listVpnConnectionResponse` |
| vpncustomergateway | `listVpnCustomerGatewayResponse` |
| vpngateway | `listVpnGatewayResponse` |
| vpnuser | `listVpnUserResponse` |
| resourcetags | `kongCreateTagsResponse` |
| host | `listHostResponse` |
| invoice | `listInvoiceResponse` |

### Async Job Response

Operations that return a `jobId` (volume mutations, VM snapshot create/revert) embed the job fields inside the resource object itself. Poll for completion using:

```
GET /restapi/asyncjob/resourceStatus?jobId=<id>
```

Async job response schema (`KongResourceApiResponse`):

```json
{
  "jobId": "string",
  "resourceId": "string",
  "resourceType": "string",
  "status": "string",
  "errorCode": "string",
  "errorMessage": "string"
}
```

`status` transitions: `IN_PROGRESS` → `SUCCEEDED` / `FAILED`

Volume operations that embed `jobId` in their response items:
- `POST /restapi/volume/createVolume`
- `GET /restapi/volume/attachVolume`
- `GET /restapi/volume/detachVolume`
- `GET /restapi/volume/resizeVolume`
- `POST /restapi/volume/uploadVolume`

VM snapshot operations that embed `jobId`:
- `POST /restapi/vmsnapshot/createVmSnapshot`
- `GET /restapi/vmsnapshot/revertToVmSnapshot`

### Delete Response

Most delete operations return HTTP 200 with an empty body (`{}`). Exceptions:

| Path | Returns |
|------|---------|
| `DELETE /restapi/volume/deleteVolume/{uuid}` | `{uuid, status}` |
| `DELETE /restapi/vmsnapshot/deleteVmSnapshot/{uuid}` | `{uuid, status}` |

### Error Response

HTTP 550 (API-level error) and HTTP 401 (auth error) are returned as:

```json
{
  "listErrorResponse": {
    "errorCode": "string",
    "errorMsg": "string"
  }
}
```

### Single Object Response

Some operations return a single flat object rather than a list:

| Path | Schema | Fields |
|------|--------|--------|
| `GET /restapi/instance/vmStatus` | `KongInstanceStatusResponse` | `uuid`, `status` |
| `GET /restapi/user/creditBalance` | `KongUserBalanceResponse` | `userEmail`, `userType`, `balanceAmount`, `type` |
| `GET /restapi/costestimate/getpublickey` | `KongKeyResponse` | `apiKey`, `secretKey` |
| `POST /restapi/kubernetes/createKubernetes` | `KongKubernetesResponse` | `uuid`, `name`, `state`, `size`, `controlNodes`, ... |

---

## Auth Model

- **Mechanism**: Two HTTP request headers on every call: `apikey` and `secretkey`
- **No login endpoint**: There is no session token or OAuth flow in the spec
- **Profile-based management**: The CLI manages named profiles storing `apikey`/`secretkey` pairs locally
- **Credential source**: Credentials are obtained out-of-band from the ZCP portal
- **No credential rotation endpoint**: The spec does not expose a key rotation API
- The `GET /restapi/costestimate/getpublickey` endpoint returns an `apiKey`/`secretKey` pair but requires no auth headers — this appears to be for the cost estimator widget, not the CLI

---

## Pagination

- **No cursor/token pagination**: The API does not support page tokens, `nextPageToken`, `offset`, or `limit` query parameters
- **Count field**: Every list response includes a `count` integer reflecting the total number of items returned
- **Full result sets**: All matching records are returned in a single response
- **UUID filter**: Most list endpoints accept a `uuid` query parameter to retrieve a single specific resource by UUID
- **Zone filter**: Most list endpoints accept a `zoneUuid` query parameter to scope results to a zone
- **No server-side sorting**: No `sort` or `orderBy` parameters are present in the spec

---

## Key Request Body Schemas (Write Operations)

| Operation | Schema | Required Fields |
|-----------|--------|----------------|
| Create Instance | `KongInstanceRequest` | `name`, `zoneUuid`, `templateUuid`, `computeOfferingUuid`, `networkUuid` |
| Create Volume | `KongVolumeRequest` | `name`, `zoneUuid`, `storageOfferingUuid`, `diskSize` |
| Create Network | `KongNetworkRequest` | `name`, `zoneUuid`, `networkOfferingUuid` |
| Create VPC | `KongVpcRequest` | `name`, `zoneUuid`, `vpcOfferingUuid`, `getcIDR` |
| Create Kubernetes | `KongKubernetesRequest` | `name`, `zoneUuid`, `kubernetesSupportedVersionUuid`, `computeOfferingUuid`, `transNetworkUuid`, `size` |
| Create Snapshot | `KongSnapShotRequest` | `name`, `zoneUuid`, `volumeUuid` |
| Create VM Snapshot | `KongVmSnapshotRequest` | `name`, `zoneUuid`, `virtualmachineUuid` |
| Create SSH Key | `KongSSHKeyRequest` | `name`, `publicKey` |
| Create Security Group | `KongSecurityGroupRequest` | `name` |
| Create Firewall Rule | `KongFirewallRuleRequest` | `ipAddressUuid`, `protocol` |
| Create Egress Rule | `KongEgressRuleRequest` | `networkUuid`, `protocol` |
| Create Port Forwarding Rule | `KongPortForwardingRequest` | `ipAddressUuid`, `protocol`, `privatePort`, `publicPort`, `virtualmachineUuid`, `networkUuid` |
| Create Load Balancer Rule | `KongLoadBalancerRuleRequest` | `name`, `publicIpUuid`, `publicport`, `privateport`, `networkUuid`, `algorithm` |
| Create VPN Connection | `KongVpnConnectionRequest` | `vpcUuid`, `customerGatewayUuid` |
| Create VPN Customer Gateway | `KongVpnCustomerGatewayRequest` | `name`, `gateway`, `cidrlist`, `ipsecpsk`, `ikepolicy`, `esppolicy` |
| Add VPN Gateway | `KongVpnGatewayRequest` | `vpcUuid` |
| Add VPN User | `KongVpnUserRequest` | `username`, `password` |
