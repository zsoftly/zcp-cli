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

| #   | Path                                      | Method | Summary                            | CLI Group    |
| --- | ----------------------------------------- | ------ | ---------------------------------- | ------------ |
| 31  | `/kubernetes-clusters`                    | GET    | List Kubernetes clusters           | `kubernetes` |
| 32  | `/kubernetes-clusters`                    | POST   | Create Kubernetes cluster          | `kubernetes` |
| 33  | `/kubernetes-clusters/versions`           | GET    | List available Kubernetes versions | `kubernetes` |
| 34  | `/kubernetes-clusters/{SLUG}`             | GET    | Get cluster details                | `kubernetes` |
| 35  | `/kubernetes-clusters/{SLUG}/start`       | PUT    | Start Kubernetes cluster           | `kubernetes` |
| 36  | `/kubernetes-clusters/{SLUG}/stop`        | PUT    | Stop Kubernetes cluster            | `kubernetes` |
| 37  | `/kubernetes-clusters/{SLUG}/scale`       | PUT    | Scale worker node count            | `kubernetes` |
| 38  | `/kubernetes-clusters/{SLUG}/change-plan` | PUT    | Change cluster compute plan        | `kubernetes` |
| 39  | `/kubernetes-clusters/{SLUG}/version`     | POST   | Upgrade Kubernetes version         | `kubernetes` |
| 40  | `/kubernetes-clusters/{SLUG}`             | DELETE | Delete Kubernetes cluster          | `kubernetes` |

### Load Balancers

| #   | Path                            | Method | Summary                    | CLI Group |
| --- | ------------------------------- | ------ | -------------------------- | --------- |
| 41  | `/load-balancers`               | GET    | List load balancers        | `lb`      |
| 42  | `/load-balancers`               | POST   | Create load balancer       | `lb`      |
| 43  | `/load-balancers/{SLUG}/rules`  | POST   | Add load balancer rule     | `lb`      |
| 44  | `/load-balancers/{SLUG}/attach` | POST   | Attach VM to load balancer | `lb`      |

### Autoscale

| #   | Path                                | Method | Summary                    | CLI Group   |
| --- | ----------------------------------- | ------ | -------------------------- | ----------- |
| 45  | `/autoscale`                        | GET    | List autoscale groups      | `autoscale` |
| 46  | `/autoscale`                        | POST   | Create autoscale group     | `autoscale` |
| 47  | `/autoscale/{SLUG}/change-plan`     | POST   | Change autoscale plan      | `autoscale` |
| 48  | `/autoscale/{SLUG}/change-template` | POST   | Change autoscale template  | `autoscale` |
| 49  | `/autoscale/{SLUG}/enable`          | PUT    | Enable autoscale group     | `autoscale` |
| 50  | `/autoscale/{SLUG}/disable`         | PUT    | Disable autoscale group    | `autoscale` |
| 51  | `/autoscale/{SLUG}/policies`        | GET    | List autoscale policies    | `autoscale` |
| 52  | `/autoscale/{SLUG}/policies`        | POST   | Create autoscale policy    | `autoscale` |
| 53  | `/autoscale/{SLUG}/policies/{ID}`   | PUT    | Update autoscale policy    | `autoscale` |
| 54  | `/autoscale/{SLUG}/policies/{ID}`   | DELETE | Delete autoscale policy    | `autoscale` |
| 55  | `/autoscale/{SLUG}/conditions`      | GET    | List autoscale conditions  | `autoscale` |
| 56  | `/autoscale/{SLUG}/conditions`      | POST   | Create autoscale condition | `autoscale` |
| 57  | `/autoscale/{SLUG}/conditions/{ID}` | PUT    | Update autoscale condition | `autoscale` |
| 58  | `/autoscale/{SLUG}/conditions/{ID}` | DELETE | Delete autoscale condition | `autoscale` |

### Networks

| #   | Path                                          | Method | Summary                     | CLI Group |
| --- | --------------------------------------------- | ------ | --------------------------- | --------- |
| 59  | `/networks`                                   | GET    | List networks               | `network` |
| 60  | `/networks`                                   | POST   | Create network              | `network` |
| 61  | `/networks/{SLUG}`                            | PUT    | Update network              | `network` |
| 62  | `/networks/categories`                        | GET    | List network categories     | `network` |
| 63  | `/networks/{SLUG}/egress-firewall-rules`      | GET    | List egress firewall rules  | `network` |
| 64  | `/networks/{SLUG}/egress-firewall-rules`      | POST   | Create egress firewall rule | `network` |
| 65  | `/networks/{SLUG}/egress-firewall-rules/{ID}` | PUT    | Update egress firewall rule | `network` |
| 66  | `/networks/{SLUG}/egress-firewall-rules/{ID}` | DELETE | Delete egress firewall rule | `network` |

### Virtual Routers

| #   | Path                             | Method | Summary               | CLI Group |
| --- | -------------------------------- | ------ | --------------------- | --------- |
| 67  | `/virtual-routers`               | GET    | List virtual routers  | `router`  |
| 68  | `/virtual-routers`               | POST   | Create virtual router | `router`  |
| 69  | `/virtual-routers/{SLUG}/reboot` | GET    | Reboot virtual router | `router`  |

### VPC

| #   | Path                                 | Method | Summary            | CLI Group |
| --- | ------------------------------------ | ------ | ------------------ | --------- |
| 70  | `/vpcs`                              | GET    | List VPCs          | `vpc`     |
| 71  | `/vpcs`                              | POST   | Create VPC         | `vpc`     |
| 72  | `/vpcs/{SLUG}`                       | PUT    | Update VPC         | `vpc`     |
| 73  | `/vpcs/{SLUG}/restart`               | GET    | Restart VPC        | `vpc`     |
| 74  | `/vpcs/{SLUG}/network-acl-list`      | GET    | List network ACLs  | `vpc`     |
| 75  | `/vpcs/{SLUG}/network-acl-list`      | POST   | Create network ACL | `vpc`     |
| 76  | `/vpcs/{SLUG}/network-acl-list/{ID}` | PUT    | Update network ACL | `vpc`     |
| 77  | `/vpcs/{SLUG}/network-acl-list/{ID}` | DELETE | Delete network ACL | `vpc`     |
| 78  | `/vpcs/{SLUG}/vpn-gateways`          | GET    | List VPN gateways  | `vpc`     |
| 79  | `/vpcs/{SLUG}/vpn-gateways`          | POST   | Create VPN gateway | `vpc`     |
| 80  | `/vpcs/{SLUG}/vpn-gateways/{ID}`     | PUT    | Update VPN gateway | `vpc`     |
| 81  | `/vpcs/{SLUG}/vpn-gateways/{ID}`     | DELETE | Delete VPN gateway | `vpc`     |

### IP Addresses

| #   | Path                                             | Method | Summary                     | CLI Group |
| --- | ------------------------------------------------ | ------ | --------------------------- | --------- |
| 82  | `/ipaddresses`                                   | GET    | List IP addresses           | `ip`      |
| 83  | `/ipaddresses`                                   | POST   | Acquire IP address          | `ip`      |
| 84  | `/ipaddresses/{SLUG}/static-nat`                 | POST   | Enable/disable static NAT   | `ip`      |
| 85  | `/ipaddresses/{SLUG}/firewall-rules`             | GET    | List firewall rules         | `ip`      |
| 86  | `/ipaddresses/{SLUG}/firewall-rules`             | POST   | Create firewall rule        | `ip`      |
| 87  | `/ipaddresses/{SLUG}/firewall-rules/{ID}`        | PUT    | Update firewall rule        | `ip`      |
| 88  | `/ipaddresses/{SLUG}/firewall-rules/{ID}`        | DELETE | Delete firewall rule        | `ip`      |
| 89  | `/ipaddresses/{SLUG}/port-forwarding-rules`      | GET    | List port forwarding rules  | `ip`      |
| 90  | `/ipaddresses/{SLUG}/port-forwarding-rules`      | POST   | Create port forwarding rule | `ip`      |
| 91  | `/ipaddresses/{SLUG}/port-forwarding-rules/{ID}` | PUT    | Update port forwarding rule | `ip`      |
| 92  | `/ipaddresses/{SLUG}/port-forwarding-rules/{ID}` | DELETE | Delete port forwarding rule | `ip`      |
| 93  | `/ipaddresses/{SLUG}/remote-access-vpns`         | GET    | List remote access VPNs     | `ip`      |
| 94  | `/ipaddresses/{SLUG}/remote-access-vpns`         | POST   | Create remote access VPN    | `ip`      |
| 95  | `/ipaddresses/{SLUG}/remote-access-vpns/{ID}`    | PUT    | Update remote access VPN    | `ip`      |
| 96  | `/ipaddresses/{SLUG}/remote-access-vpns/{ID}`    | DELETE | Delete remote access VPN    | `ip`      |

### VPN

| #   | Path                            | Method | Summary                     | CLI Group |
| --- | ------------------------------- | ------ | --------------------------- | --------- |
| 97  | `/vpn-users`                    | GET    | List VPN users              | `vpn`     |
| 98  | `/vpn-users`                    | POST   | Create VPN user             | `vpn`     |
| 99  | `/vpn-users/{SLUG}`             | PUT    | Update VPN user             | `vpn`     |
| 100 | `/vpn-users/{SLUG}`             | DELETE | Delete VPN user             | `vpn`     |
| 101 | `/vpn-customer-gateways`        | GET    | List VPN customer gateways  | `vpn`     |
| 102 | `/vpn-customer-gateways`        | POST   | Create VPN customer gateway | `vpn`     |
| 103 | `/vpn-customer-gateways/{SLUG}` | PUT    | Update VPN customer gateway | `vpn`     |
| 104 | `/vpn-customer-gateways/{SLUG}` | DELETE | Delete VPN customer gateway | `vpn`     |

### DNS

| #   | Path                               | Method | Summary           | CLI Group |
| --- | ---------------------------------- | ------ | ----------------- | --------- |
| 105 | `/dns/domains`                     | GET    | List DNS domains  | `dns`     |
| 106 | `/dns/domains`                     | POST   | Create DNS domain | `dns`     |
| 107 | `/dns/domains/{SLUG}`              | PUT    | Update DNS domain | `dns`     |
| 108 | `/dns/domains/{SLUG}`              | DELETE | Delete DNS domain | `dns`     |
| 109 | `/dns/domains/{SLUG}/records`      | POST   | Create DNS record | `dns`     |
| 110 | `/dns/domains/{SLUG}/records/{ID}` | DELETE | Delete DNS record | `dns`     |

### Projects

| #   | Path                         | Method | Summary               | CLI Group |
| --- | ---------------------------- | ------ | --------------------- | --------- |
| 111 | `/projects`                  | GET    | List projects         | `project` |
| 112 | `/projects`                  | POST   | Create project        | `project` |
| 113 | `/projects/{SLUG}`           | PUT    | Update project        | `project` |
| 114 | `/projects/{SLUG}/dashboard` | GET    | Get project dashboard | `project` |
| 115 | `/projects/{SLUG}/icons`     | GET    | Get project icons     | `project` |
| 116 | `/projects/{SLUG}/users`     | GET    | List project users    | `project` |
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

### Object Storage

Object storage instances, buckets, and object metadata are managed via the ZCP REST API. Object uploads and deletes use the **S3 protocol** (AWS Signature V4) directly against the region's `s3_endpoint` — not the ZCP REST API.

| #   | Path                                              | Method | Summary                               | CLI Group        |
| --- | ------------------------------------------------- | ------ | ------------------------------------- | ---------------- |
| 132 | `/object-storages`                                | GET    | List object storage instances         | `object-storage` |
| 133 | `/object-storages`                                | POST   | Create object storage instance        | `object-storage` |
| 134 | `/object-storages/{SLUG}`                         | GET    | Get object storage instance           | `object-storage` |
| 135 | `/object-storages/{SLUG}/resize`                  | POST   | Resize object storage instance        | `object-storage` |
| 136 | `/object-storages/{SLUG}`                         | DELETE | Delete object storage instance        | `object-storage` |
| 137 | `/object-storages/{SLUG}/buckets`                 | GET    | List buckets                          | `object-storage` |
| 138 | `/object-storages/{SLUG}/buckets`                 | POST   | Create bucket                         | `object-storage` |
| 139 | `/object-storages/{SLUG}/buckets/{BSLUG}`         | GET    | Get bucket                            | `object-storage` |
| 140 | `/object-storages/{SLUG}/buckets/{BSLUG}`         | PUT    | Update bucket settings                | `object-storage` |
| 141 | `/object-storages/{SLUG}/buckets/{BSLUG}`         | DELETE | Delete bucket                         | `object-storage` |
| 142 | `/object-storages/{SLUG}/buckets/{BSLUG}/acl`     | PUT    | Set bucket ACL                        | `object-storage` |
| 143 | `/object-storages/{SLUG}/buckets/{BSLUG}/objects` | GET    | List objects in bucket (cursor-paged) | `object-storage` |

> **S3 endpoints** (used by `object put` and `object delete`): the CLI reads `region.cloud_provider_setup.config.s3_endpoint` from the `GET /object-storages/{SLUG}?include=region` response and opens a direct S3 connection using the instance's `api_key` / `api_secret` as AWS access credentials.

### Billing

| #   | Path                      | Method | Summary                   | CLI Group |
| --- | ------------------------- | ------ | ------------------------- | --------- |
| 144 | `/billing/costs`          | GET    | Get current costs         | `billing` |
| 145 | `/billing/balance`        | GET    | Get account balance       | `billing` |
| 146 | `/billing/monthly-usage`  | GET    | Get monthly usage summary | `billing` |
| 147 | `/billing/credit-limit`   | GET    | Get credit limit          | `billing` |
| 148 | `/billing/service-counts` | GET    | Get service counts        | `billing` |
| 149 | `/billing/subscriptions`  | GET    | List subscriptions        | `billing` |
| 150 | `/billing/invoices`       | GET    | List invoices             | `billing` |
| 151 | `/billing/usage`          | GET    | Get detailed usage        | `billing` |
| 152 | `/billing/free-credits`   | GET    | Get free credits          | `billing` |
| 153 | `/billing/contracts`      | GET    | List contracts            | `billing` |
| 154 | `/billing/trials`         | GET    | List trials               | `billing` |
| 155 | `/billing/payments`       | GET    | List payments             | `billing` |
| 156 | `/billing/coupons`        | GET    | List coupons              | `billing` |
| 157 | `/billing/coupons`        | POST   | Apply coupon              | `billing` |
| 158 | `/billing/budget-alerts`  | GET    | List budget alerts        | `billing` |
| 159 | `/billing/budget-alerts`  | POST   | Create budget alert       | `billing` |
| 160 | `/billing/cancel-service` | POST   | Cancel a service          | `billing` |

### Profile

| #   | Path                       | Method | Summary                | CLI Group |
| --- | -------------------------- | ------ | ---------------------- | --------- |
| 161 | `/profile`                 | GET    | Get user profile       | `profile` |
| 162 | `/profile`                 | PUT    | Update user profile    | `profile` |
| 163 | `/profile/company-details` | PUT    | Update company details | `profile` |
| 164 | `/profile/time-settings`   | POST   | Update time settings   | `profile` |
| 165 | `/profile/api-enable`      | POST   | Enable API access      | `profile` |
| 166 | `/profile/api-disable`     | DELETE | Disable API access     | `profile` |
| 167 | `/profile/activity-logs`   | GET    | Get activity logs      | `profile` |

### SSH Keys

| #   | Path                     | Method | Summary        | CLI Group |
| --- | ------------------------ | ------ | -------------- | --------- |
| 168 | `/users/ssh-keys`        | GET    | List SSH keys  | `ssh-key` |
| 169 | `/users/ssh-keys`        | POST   | Create SSH key | `ssh-key` |
| 170 | `/users/ssh-keys/{SLUG}` | DELETE | Delete SSH key | `ssh-key` |

### Support

| #   | Path                              | Method | Summary               | CLI Group |
| --- | --------------------------------- | ------ | --------------------- | --------- |
| 171 | `/support/tickets`                | GET    | List support tickets  | `support` |
| 172 | `/support/tickets`                | POST   | Create support ticket | `support` |
| 173 | `/support/tickets/{SLUG}`         | PUT    | Update support ticket | `support` |
| 174 | `/support/tickets/{SLUG}`         | DELETE | Delete support ticket | `support` |
| 175 | `/support/tickets/{SLUG}/replies` | GET    | List ticket replies   | `support` |
| 176 | `/support/tickets/{SLUG}/replies` | POST   | Reply to ticket       | `support` |
| 177 | `/support/feedback`               | GET    | List feedback         | `support` |
| 178 | `/support/feedback`               | POST   | Submit feedback       | `support` |
| 179 | `/support/faqs`                   | GET    | List FAQs             | `support` |

### Plans

| #   | Path                      | Method | Summary                  | CLI Group |
| --- | ------------------------- | ------ | ------------------------ | --------- |
| 180 | `/plans/service/VM`       | GET    | List VM plans            | `plan`    |
| 181 | `/plans/service/Router`   | GET    | List router plans        | `plan`    |
| 182 | `/plans/service/Storage`  | GET    | List storage plans       | `plan`    |
| 183 | `/plans/service/LB`       | GET    | List load balancer plans | `plan`    |
| 184 | `/plans/service/K8s`      | GET    | List Kubernetes plans    | `plan`    |
| 185 | `/plans/service/IP`       | GET    | List IP address plans    | `plan`    |
| 186 | `/plans/service/Snapshot` | GET    | List snapshot plans      | `plan`    |
| 187 | `/plans/service/Template` | GET    | List template plans      | `plan`    |
| 188 | `/plans/service/ISO`      | GET    | List ISO plans           | `plan`    |
| 189 | `/plans/service/Backups`  | GET    | List backup plans        | `plan`    |

### Discovery

| #   | Path                  | Method | Summary                 | CLI Group   |
| --- | --------------------- | ------ | ----------------------- | ----------- |
| 190 | `/regions`            | GET    | List regions            | `discovery` |
| 191 | `/servers`            | GET    | List servers            | `discovery` |
| 192 | `/cloud-providers`    | GET    | List cloud providers    | `discovery` |
| 193 | `/currencies`         | GET    | List currencies         | `discovery` |
| 194 | `/storage-categories` | GET    | List storage categories | `discovery` |
| 195 | `/billing-cycles`     | GET    | List billing cycles     | `discovery` |
| 196 | `/unit-pricings`      | GET    | List unit pricings      | `discovery` |

### Store

| #   | Path                         | Method | Summary                 | CLI Group |
| --- | ---------------------------- | ------ | ----------------------- | --------- |
| 197 | `/store/items`               | GET    | List store items        | `store`   |
| 198 | `/store/checkout`            | POST   | Checkout store cart     | `store`   |
| 199 | `/store/marketplace-apps`    | GET    | List marketplace apps   | `store`   |
| 200 | `/store/products/categories` | GET    | List product categories | `store`   |

### Auth

| #   | Path              | Method | Summary                      | CLI Group |
| --- | ----------------- | ------ | ---------------------------- | --------- |
| 201 | `/login`          | POST   | Log in (obtain Bearer token) | `auth`    |
| 202 | `/register`       | POST   | Register new account         | `auth`    |
| 203 | `/reset-password` | POST   | Reset password               | `auth`    |
| 204 | `/mfa/enable`     | POST   | Enable MFA                   | `auth`    |
| 205 | `/mfa/disable`    | POST   | Disable MFA                  | `auth`    |
| 206 | `/mfa/verify`     | POST   | Verify MFA code              | `auth`    |

**Total endpoints**: 206

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
