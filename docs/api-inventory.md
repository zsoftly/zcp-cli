# ZCP API Inventory

**Base URL**: `https://cloud.zcp.zsoftly.ca/api/v1`
**Auth**: Bearer token (`Authorization: Bearer <token>`)
**Style**: RESTful, resource-oriented endpoints with `{SLUG}` path parameters

---

## Endpoint Table

### Compute (Virtual Machines)

| #   | Path                                             | Method | Summary                  | CLI Group |
| --- | ------------------------------------------------ | ------ | ------------------------ | --------- |
| 1   | `/virtual-machines`                              | GET    | List all VMs             | `vm`      |
| 2   | `/virtual-machines`                              | POST   | Create a VM              | `vm`      |
| 3   | `/virtual-machines/{SLUG}`                       | GET    | Get VM details           | `vm`      |
| 4   | `/virtual-machines/{SLUG}/start`                 | PUT    | Start VM                 | `vm`      |
| 5   | `/virtual-machines/{SLUG}/stop`                  | PUT    | Stop VM                  | `vm`      |
| 6   | `/virtual-machines/{SLUG}/reboot`                | PUT    | Reboot VM                | `vm`      |
| 7   | `/virtual-machines/{SLUG}/reset`                 | PUT    | Reset VM                 | `vm`      |
| 8   | `/virtual-machines/{SLUG}/change-label`          | POST   | Change VM label          | `vm`      |
| 9   | `/virtual-machines/{SLUG}/change-password`       | POST   | Change VM password       | `vm`      |
| 10  | `/virtual-machines/{SLUG}/change-plan`           | POST   | Change VM plan/sizing    | `vm`      |
| 11  | `/virtual-machines/{SLUG}/change-template`       | POST   | Change VM template       | `vm`      |
| 12  | `/virtual-machines/{SLUG}/change-startup-script` | POST   | Change VM startup script | `vm`      |
| 13  | `/virtual-machines/{SLUG}/add-network`           | POST   | Add network to VM        | `vm`      |
| 14  | `/virtual-machines/{SLUG}/tags`                  | POST   | Add tags to VM           | `vm`      |
| 15  | `/virtual-machines/{SLUG}/tags`                  | DELETE | Remove tags from VM      | `vm`      |
| 16  | `/virtual-machines/{SLUG}/addons`                | GET    | List VM addons           | `vm`      |

### Storage (Block Storage)

| #   | Path                           | Method | Summary                     | CLI Group |
| --- | ------------------------------ | ------ | --------------------------- | --------- |
| 17  | `/blockstorages`               | GET    | List block storage volumes  | `storage` |
| 18  | `/blockstorages`               | POST   | Create block storage volume | `storage` |
| 19  | `/blockstorages/{SLUG}/attach` | POST   | Attach volume to VM         | `storage` |
| 20  | `/blockstorages/{SLUG}/detach` | POST   | Detach volume from VM       | `storage` |

### Snapshots

| #   | Path                                        | Method | Summary                   | CLI Group  |
| --- | ------------------------------------------- | ------ | ------------------------- | ---------- |
| 21  | `/virtual-machines/snapshots`               | GET    | List all VM snapshots     | `snapshot` |
| 22  | `/virtual-machines/{SLUG}/snapshots`        | POST   | Create VM snapshot        | `snapshot` |
| 23  | `/virtual-machines/{SLUG}/snapshots/revert` | POST   | Revert to VM snapshot     | `snapshot` |
| 24  | `/blockstorages/snapshots`                  | GET    | List all volume snapshots | `snapshot` |
| 25  | `/blockstorages/{SLUG}/snapshots`           | POST   | Create volume snapshot    | `snapshot` |
| 26  | `/blockstorages/{SLUG}/snapshots/revert`    | POST   | Revert to volume snapshot | `snapshot` |

### Backups

| #   | Path                               | Method | Summary                 | CLI Group |
| --- | ---------------------------------- | ------ | ----------------------- | --------- |
| 27  | `/virtual-machines/backups`        | GET    | List all VM backups     | `backup`  |
| 28  | `/virtual-machines/{SLUG}/backups` | POST   | Create VM backup        | `backup`  |
| 29  | `/blockstorages/backups`           | GET    | List all volume backups | `backup`  |
| 30  | `/blockstorages/{SLUG}/backups`    | POST   | Create volume backup    | `backup`  |

### Kubernetes

| #   | Path                                      | Method | Summary                   | CLI Group    |
| --- | ----------------------------------------- | ------ | ------------------------- | ------------ |
| 31  | `/kubernetes-clusters`                    | GET    | List Kubernetes clusters  | `kubernetes` |
| 32  | `/kubernetes-clusters`                    | POST   | Create Kubernetes cluster | `kubernetes` |
| 33  | `/kubernetes-clusters/{SLUG}/start`       | PUT    | Start Kubernetes cluster  | `kubernetes` |
| 34  | `/kubernetes-clusters/{SLUG}/stop`        | PUT    | Stop Kubernetes cluster   | `kubernetes` |
| 35  | `/kubernetes-clusters/{SLUG}/change-plan` | PUT    | Change cluster plan       | `kubernetes` |

### Load Balancers

| #   | Path                            | Method | Summary                    | CLI Group |
| --- | ------------------------------- | ------ | -------------------------- | --------- |
| 36  | `/load-balancers`               | GET    | List load balancers        | `lb`      |
| 37  | `/load-balancers`               | POST   | Create load balancer       | `lb`      |
| 38  | `/load-balancers/{SLUG}/rules`  | POST   | Add load balancer rule     | `lb`      |
| 39  | `/load-balancers/{SLUG}/attach` | POST   | Attach VM to load balancer | `lb`      |

### Autoscale

| #   | Path                                | Method | Summary                    | CLI Group   |
| --- | ----------------------------------- | ------ | -------------------------- | ----------- |
| 40  | `/autoscale`                        | GET    | List autoscale groups      | `autoscale` |
| 41  | `/autoscale`                        | POST   | Create autoscale group     | `autoscale` |
| 42  | `/autoscale/{SLUG}/change-plan`     | POST   | Change autoscale plan      | `autoscale` |
| 43  | `/autoscale/{SLUG}/change-template` | POST   | Change autoscale template  | `autoscale` |
| 44  | `/autoscale/{SLUG}/enable`          | PUT    | Enable autoscale group     | `autoscale` |
| 45  | `/autoscale/{SLUG}/disable`         | PUT    | Disable autoscale group    | `autoscale` |
| 46  | `/autoscale/{SLUG}/policies`        | GET    | List autoscale policies    | `autoscale` |
| 47  | `/autoscale/{SLUG}/policies`        | POST   | Create autoscale policy    | `autoscale` |
| 48  | `/autoscale/{SLUG}/policies/{ID}`   | PUT    | Update autoscale policy    | `autoscale` |
| 49  | `/autoscale/{SLUG}/policies/{ID}`   | DELETE | Delete autoscale policy    | `autoscale` |
| 50  | `/autoscale/{SLUG}/conditions`      | GET    | List autoscale conditions  | `autoscale` |
| 51  | `/autoscale/{SLUG}/conditions`      | POST   | Create autoscale condition | `autoscale` |
| 52  | `/autoscale/{SLUG}/conditions/{ID}` | PUT    | Update autoscale condition | `autoscale` |
| 53  | `/autoscale/{SLUG}/conditions/{ID}` | DELETE | Delete autoscale condition | `autoscale` |

### Networks

| #   | Path                                          | Method | Summary                     | CLI Group |
| --- | --------------------------------------------- | ------ | --------------------------- | --------- |
| 54  | `/networks`                                   | GET    | List networks               | `network` |
| 55  | `/networks`                                   | POST   | Create network              | `network` |
| 56  | `/networks/{SLUG}`                            | PUT    | Update network              | `network` |
| 57  | `/networks/categories`                        | GET    | List network categories     | `network` |
| 58  | `/networks/{SLUG}/egress-firewall-rules`      | GET    | List egress firewall rules  | `network` |
| 59  | `/networks/{SLUG}/egress-firewall-rules`      | POST   | Create egress firewall rule | `network` |
| 60  | `/networks/{SLUG}/egress-firewall-rules/{ID}` | PUT    | Update egress firewall rule | `network` |
| 61  | `/networks/{SLUG}/egress-firewall-rules/{ID}` | DELETE | Delete egress firewall rule | `network` |

### Virtual Routers

| #   | Path                             | Method | Summary               | CLI Group |
| --- | -------------------------------- | ------ | --------------------- | --------- |
| 62  | `/virtual-routers`               | GET    | List virtual routers  | `router`  |
| 63  | `/virtual-routers`               | POST   | Create virtual router | `router`  |
| 64  | `/virtual-routers/{SLUG}/reboot` | GET    | Reboot virtual router | `router`  |

### VPC

| #   | Path                                 | Method | Summary            | CLI Group |
| --- | ------------------------------------ | ------ | ------------------ | --------- |
| 65  | `/vpcs`                              | GET    | List VPCs          | `vpc`     |
| 66  | `/vpcs`                              | POST   | Create VPC         | `vpc`     |
| 67  | `/vpcs/{SLUG}`                       | PUT    | Update VPC         | `vpc`     |
| 68  | `/vpcs/{SLUG}/restart`               | GET    | Restart VPC        | `vpc`     |
| 69  | `/vpcs/{SLUG}/network-acl-list`      | GET    | List network ACLs  | `vpc`     |
| 70  | `/vpcs/{SLUG}/network-acl-list`      | POST   | Create network ACL | `vpc`     |
| 71  | `/vpcs/{SLUG}/network-acl-list/{ID}` | PUT    | Update network ACL | `vpc`     |
| 72  | `/vpcs/{SLUG}/network-acl-list/{ID}` | DELETE | Delete network ACL | `vpc`     |
| 73  | `/vpcs/{SLUG}/vpn-gateways`          | GET    | List VPN gateways  | `vpc`     |
| 74  | `/vpcs/{SLUG}/vpn-gateways`          | POST   | Create VPN gateway | `vpc`     |
| 75  | `/vpcs/{SLUG}/vpn-gateways/{ID}`     | PUT    | Update VPN gateway | `vpc`     |
| 76  | `/vpcs/{SLUG}/vpn-gateways/{ID}`     | DELETE | Delete VPN gateway | `vpc`     |

### IP Addresses

| #   | Path                                             | Method | Summary                     | CLI Group |
| --- | ------------------------------------------------ | ------ | --------------------------- | --------- |
| 77  | `/ipaddresses`                                   | GET    | List IP addresses           | `ip`      |
| 78  | `/ipaddresses`                                   | POST   | Acquire IP address          | `ip`      |
| 79  | `/ipaddresses/{SLUG}/static-nat`                 | POST   | Enable/disable static NAT   | `ip`      |
| 80  | `/ipaddresses/{SLUG}/firewall-rules`             | GET    | List firewall rules         | `ip`      |
| 81  | `/ipaddresses/{SLUG}/firewall-rules`             | POST   | Create firewall rule        | `ip`      |
| 82  | `/ipaddresses/{SLUG}/firewall-rules/{ID}`        | PUT    | Update firewall rule        | `ip`      |
| 83  | `/ipaddresses/{SLUG}/firewall-rules/{ID}`        | DELETE | Delete firewall rule        | `ip`      |
| 84  | `/ipaddresses/{SLUG}/port-forwarding-rules`      | GET    | List port forwarding rules  | `ip`      |
| 85  | `/ipaddresses/{SLUG}/port-forwarding-rules`      | POST   | Create port forwarding rule | `ip`      |
| 86  | `/ipaddresses/{SLUG}/port-forwarding-rules/{ID}` | PUT    | Update port forwarding rule | `ip`      |
| 87  | `/ipaddresses/{SLUG}/port-forwarding-rules/{ID}` | DELETE | Delete port forwarding rule | `ip`      |
| 88  | `/ipaddresses/{SLUG}/remote-access-vpns`         | GET    | List remote access VPNs     | `ip`      |
| 89  | `/ipaddresses/{SLUG}/remote-access-vpns`         | POST   | Create remote access VPN    | `ip`      |
| 90  | `/ipaddresses/{SLUG}/remote-access-vpns/{ID}`    | PUT    | Update remote access VPN    | `ip`      |
| 91  | `/ipaddresses/{SLUG}/remote-access-vpns/{ID}`    | DELETE | Delete remote access VPN    | `ip`      |

### VPN

| #   | Path                            | Method | Summary                     | CLI Group |
| --- | ------------------------------- | ------ | --------------------------- | --------- |
| 92  | `/vpn-users`                    | GET    | List VPN users              | `vpn`     |
| 93  | `/vpn-users`                    | POST   | Create VPN user             | `vpn`     |
| 94  | `/vpn-users/{SLUG}`             | PUT    | Update VPN user             | `vpn`     |
| 95  | `/vpn-users/{SLUG}`             | DELETE | Delete VPN user             | `vpn`     |
| 96  | `/vpn-customer-gateways`        | GET    | List VPN customer gateways  | `vpn`     |
| 97  | `/vpn-customer-gateways`        | POST   | Create VPN customer gateway | `vpn`     |
| 98  | `/vpn-customer-gateways/{SLUG}` | PUT    | Update VPN customer gateway | `vpn`     |
| 99  | `/vpn-customer-gateways/{SLUG}` | DELETE | Delete VPN customer gateway | `vpn`     |

### DNS

| #   | Path                               | Method | Summary           | CLI Group |
| --- | ---------------------------------- | ------ | ----------------- | --------- |
| 100 | `/dns/domains`                     | GET    | List DNS domains  | `dns`     |
| 101 | `/dns/domains`                     | POST   | Create DNS domain | `dns`     |
| 102 | `/dns/domains/{SLUG}`              | PUT    | Update DNS domain | `dns`     |
| 103 | `/dns/domains/{SLUG}`              | DELETE | Delete DNS domain | `dns`     |
| 104 | `/dns/domains/{SLUG}/records`      | POST   | Create DNS record | `dns`     |
| 105 | `/dns/domains/{SLUG}/records/{ID}` | DELETE | Delete DNS record | `dns`     |

### Projects

| #   | Path                         | Method | Summary               | CLI Group |
| --- | ---------------------------- | ------ | --------------------- | --------- |
| 106 | `/projects`                  | GET    | List projects         | `project` |
| 107 | `/projects`                  | POST   | Create project        | `project` |
| 108 | `/projects/{SLUG}`           | PUT    | Update project        | `project` |
| 109 | `/projects/{SLUG}/dashboard` | GET    | Get project dashboard | `project` |
| 110 | `/projects/{SLUG}/icons`     | GET    | Get project icons     | `project` |
| 111 | `/projects/{SLUG}/users`     | GET    | List project users    | `project` |
| 112 | `/projects/{SLUG}/users`     | POST   | Add user to project   | `project` |

### ISOs

| #   | Path           | Method | Summary             | CLI Group |
| --- | -------------- | ------ | ------------------- | --------- |
| 113 | `/isos`        | GET    | List ISOs           | `iso`     |
| 114 | `/isos`        | POST   | Upload/register ISO | `iso`     |
| 115 | `/isos/{SLUG}` | PUT    | Update ISO          | `iso`     |
| 116 | `/isos/{SLUG}` | DELETE | Delete ISO          | `iso`     |

### Affinity Groups

| #   | Path                      | Method | Summary               | CLI Group        |
| --- | ------------------------- | ------ | --------------------- | ---------------- |
| 117 | `/affinity-groups`        | GET    | List affinity groups  | `affinity-group` |
| 118 | `/affinity-groups`        | POST   | Create affinity group | `affinity-group` |
| 119 | `/affinity-groups/{SLUG}` | DELETE | Delete affinity group | `affinity-group` |

### Templates

| #   | Path                        | Method | Summary                 | CLI Group  |
| --- | --------------------------- | ------ | ----------------------- | ---------- |
| 120 | `/templates`                | GET    | List public templates   | `template` |
| 121 | `/account/templates`        | GET    | List account templates  | `template` |
| 122 | `/account/templates`        | POST   | Create account template | `template` |
| 123 | `/account/templates/{SLUG}` | PUT    | Update account template | `template` |
| 124 | `/account/templates/{SLUG}` | DELETE | Delete account template | `template` |

### Monitoring

| #   | Path                                    | Method | Summary                    | CLI Group    |
| --- | --------------------------------------- | ------ | -------------------------- | ------------ |
| 125 | `/monitoring/global`                    | GET    | Global monitoring overview | `monitoring` |
| 126 | `/monitoring/charts`                    | GET    | Monitoring chart data      | `monitoring` |
| 127 | `/monitoring/{SLUG}/cpu-usage`          | GET    | VM CPU usage metrics       | `monitoring` |
| 128 | `/monitoring/{SLUG}/disk-read-write`    | GET    | VM disk read/write metrics | `monitoring` |
| 129 | `/monitoring/{SLUG}/memory-usage`       | GET    | VM memory usage metrics    | `monitoring` |
| 130 | `/monitoring/{SLUG}/network-traffic`    | GET    | VM network traffic metrics | `monitoring` |
| 131 | `/monitoring/{SLUG}/disk-io-read-write` | GET    | VM disk I/O metrics        | `monitoring` |

### Billing

| #   | Path                      | Method | Summary                   | CLI Group |
| --- | ------------------------- | ------ | ------------------------- | --------- |
| 132 | `/billing/costs`          | GET    | Get current costs         | `billing` |
| 133 | `/billing/balance`        | GET    | Get account balance       | `billing` |
| 134 | `/billing/monthly-usage`  | GET    | Get monthly usage summary | `billing` |
| 135 | `/billing/credit-limit`   | GET    | Get credit limit          | `billing` |
| 136 | `/billing/service-counts` | GET    | Get service counts        | `billing` |
| 137 | `/billing/subscriptions`  | GET    | List subscriptions        | `billing` |
| 138 | `/billing/invoices`       | GET    | List invoices             | `billing` |
| 139 | `/billing/usage`          | GET    | Get detailed usage        | `billing` |
| 140 | `/billing/free-credits`   | GET    | Get free credits          | `billing` |
| 141 | `/billing/contracts`      | GET    | List contracts            | `billing` |
| 142 | `/billing/trials`         | GET    | List trials               | `billing` |
| 143 | `/billing/payments`       | GET    | List payments             | `billing` |
| 144 | `/billing/coupons`        | GET    | List coupons              | `billing` |
| 145 | `/billing/coupons`        | POST   | Apply coupon              | `billing` |
| 146 | `/billing/budget-alerts`  | GET    | List budget alerts        | `billing` |
| 147 | `/billing/budget-alerts`  | POST   | Create budget alert       | `billing` |
| 148 | `/billing/cancel-service` | POST   | Cancel a service          | `billing` |

### Profile

| #   | Path                       | Method | Summary                | CLI Group |
| --- | -------------------------- | ------ | ---------------------- | --------- |
| 149 | `/profile`                 | GET    | Get user profile       | `profile` |
| 150 | `/profile`                 | PUT    | Update user profile    | `profile` |
| 151 | `/profile/company-details` | PUT    | Update company details | `profile` |
| 152 | `/profile/time-settings`   | POST   | Update time settings   | `profile` |
| 153 | `/profile/api-enable`      | POST   | Enable API access      | `profile` |
| 154 | `/profile/api-disable`     | DELETE | Disable API access     | `profile` |
| 155 | `/profile/activity-logs`   | GET    | Get activity logs      | `profile` |

### SSH Keys

| #   | Path                     | Method | Summary        | CLI Group |
| --- | ------------------------ | ------ | -------------- | --------- |
| 156 | `/users/ssh-keys`        | GET    | List SSH keys  | `ssh-key` |
| 157 | `/users/ssh-keys`        | POST   | Create SSH key | `ssh-key` |
| 158 | `/users/ssh-keys/{SLUG}` | DELETE | Delete SSH key | `ssh-key` |

### Support

| #   | Path                              | Method | Summary               | CLI Group |
| --- | --------------------------------- | ------ | --------------------- | --------- |
| 159 | `/support/tickets`                | GET    | List support tickets  | `support` |
| 160 | `/support/tickets`                | POST   | Create support ticket | `support` |
| 161 | `/support/tickets/{SLUG}`         | PUT    | Update support ticket | `support` |
| 162 | `/support/tickets/{SLUG}`         | DELETE | Delete support ticket | `support` |
| 163 | `/support/tickets/{SLUG}/replies` | GET    | List ticket replies   | `support` |
| 164 | `/support/tickets/{SLUG}/replies` | POST   | Reply to ticket       | `support` |
| 165 | `/support/feedback`               | GET    | List feedback         | `support` |
| 166 | `/support/feedback`               | POST   | Submit feedback       | `support` |
| 167 | `/support/faqs`                   | GET    | List FAQs             | `support` |

### Plans

| #   | Path                      | Method | Summary                  | CLI Group |
| --- | ------------------------- | ------ | ------------------------ | --------- |
| 168 | `/plans/service/VM`       | GET    | List VM plans            | `plan`    |
| 169 | `/plans/service/Router`   | GET    | List router plans        | `plan`    |
| 170 | `/plans/service/Storage`  | GET    | List storage plans       | `plan`    |
| 171 | `/plans/service/LB`       | GET    | List load balancer plans | `plan`    |
| 172 | `/plans/service/K8s`      | GET    | List Kubernetes plans    | `plan`    |
| 173 | `/plans/service/IP`       | GET    | List IP address plans    | `plan`    |
| 174 | `/plans/service/Snapshot` | GET    | List snapshot plans      | `plan`    |
| 175 | `/plans/service/Template` | GET    | List template plans      | `plan`    |
| 176 | `/plans/service/ISO`      | GET    | List ISO plans           | `plan`    |
| 177 | `/plans/service/Backups`  | GET    | List backup plans        | `plan`    |

### Discovery

| #   | Path                  | Method | Summary                 | CLI Group   |
| --- | --------------------- | ------ | ----------------------- | ----------- |
| 178 | `/regions`            | GET    | List regions            | `discovery` |
| 179 | `/servers`            | GET    | List servers            | `discovery` |
| 180 | `/cloud-providers`    | GET    | List cloud providers    | `discovery` |
| 181 | `/currencies`         | GET    | List currencies         | `discovery` |
| 182 | `/storage-categories` | GET    | List storage categories | `discovery` |
| 183 | `/billing-cycles`     | GET    | List billing cycles     | `discovery` |
| 184 | `/unit-pricings`      | GET    | List unit pricings      | `discovery` |

### Store

| #   | Path                         | Method | Summary                 | CLI Group |
| --- | ---------------------------- | ------ | ----------------------- | --------- |
| 185 | `/store/items`               | GET    | List store items        | `store`   |
| 186 | `/store/checkout`            | POST   | Checkout store cart     | `store`   |
| 187 | `/store/marketplace-apps`    | GET    | List marketplace apps   | `store`   |
| 188 | `/store/products/categories` | GET    | List product categories | `store`   |

### Auth

| #   | Path              | Method | Summary                      | CLI Group |
| --- | ----------------- | ------ | ---------------------------- | --------- |
| 189 | `/login`          | POST   | Log in (obtain Bearer token) | `auth`    |
| 190 | `/register`       | POST   | Register new account         | `auth`    |
| 191 | `/reset-password` | POST   | Reset password               | `auth`    |
| 192 | `/mfa/enable`     | POST   | Enable MFA                   | `auth`    |
| 193 | `/mfa/disable`    | POST   | Disable MFA                  | `auth`    |
| 194 | `/mfa/verify`     | POST   | Verify MFA code              | `auth`    |

**Total endpoints**: 194

---

## Auth Model

- **Mechanism**: Bearer token via `Authorization: Bearer <token>` header on every authenticated request
- **Login endpoint**: `POST /login` returns a Bearer token given valid credentials
- **Token management**: The CLI stores tokens per-profile in the local config directory
- **MFA support**: Optional MFA flow via `/mfa/enable`, `/mfa/verify`, `/mfa/disable`
- **No API key headers**: The old `apikey`/`secretkey` header pattern is replaced by Bearer tokens

---

## API Response Patterns

### List Response

List endpoints return a JSON array of resource objects, or a wrapper with a `data` array and pagination metadata:

```json
{
  "data": [ ... ],
  "meta": {
    "total": 42,
    "page": 1,
    "per_page": 25
  }
}
```

### Single Resource Response

GET/PUT/POST on a single resource returns the resource object directly:

```json
{
  "slug": "abc-123",
  "name": "my-resource",
  "status": "running",
  ...
}
```

### Error Response

Errors return standard HTTP status codes with a JSON body:

```json
{
  "error": {
    "code": "not_found",
    "message": "Resource not found"
  }
}
```

Common status codes:

- `401` — Invalid or expired Bearer token
- `403` — Insufficient permissions
- `404` — Resource not found
- `422` — Validation error
- `429` — Rate limit exceeded

### Delete Response

Delete operations return HTTP 204 (No Content) with an empty body on success.

---

## Resource Identifiers

- Resources are identified by **SLUG** (a URL-friendly unique identifier) rather than UUIDs
- SLUGs appear in URL paths: `/virtual-machines/{SLUG}`
- Sub-resources use numeric or secondary IDs: `/ipaddresses/{SLUG}/firewall-rules/{ID}`

---

## Pagination

- List endpoints support `page` and `per_page` query parameters
- Response `meta` object includes `total`, `page`, and `per_page` fields
- Default page size varies by endpoint (typically 25)

---

## Notes

- All write operations (POST/PUT/DELETE) require a valid Bearer token
- Discovery endpoints (`/regions`, `/currencies`, etc.) may be publicly accessible
- The `/plans/service/{ServiceType}` pattern uses fixed service type values (VM, Router, Storage, LB, K8s, IP, Snapshot, Template, ISO, Backups)
- Monitoring endpoints require the target VM SLUG and return time-series data
