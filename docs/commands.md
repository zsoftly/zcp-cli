# ZCP CLI — Command Reference

Copy-paste examples for every resource the CLI manages. Use `zcp <command> --help`
for the full flag list of any command.

> All examples use working defaults: region `yow-1`, project `default`, and billing
> cycle `hourly`. Substitute your own values as needed — run `zcp region list` and
> `zcp project list` to see what is available to your account.

## The cloud provider is automatic

You do not pass a cloud provider. Your account's compute provider is detected and
saved to your profile the first time you run `zcp profile add` or `zcp auth validate`,
then applied to every create command automatically. Object storage and DNS use their
own providers automatically, so those commands need nothing extra either.

An override is still available via the hidden `--cloud-provider <slug>` flag or the
`ZCP_CLOUD_PROVIDER` environment variable; run `zcp cloud-provider list` to see the
slugs.

## Set your other defaults once

Region and project are still needed by most create commands. Export them once and
omit the flags everywhere:

```bash
export ZCP_REGION=yow-1
export ZCP_PROJECT=default
```

With those set, `zcp instance create --name my-vm --template ubuntu-22f ...` no longer
needs `--region` or `--project`. The examples below pass them explicitly so each one
works on its own; drop them if you have exported the variables. See
[configuration.md](configuration.md) for the full list of environment variables.

---

## Discovery

Start here — these read-only commands show what your account can provision.

```bash
# Regions, cloud providers, and other catalog data
zcp region list
zcp cloud-provider list    # only needed if your account has multiple providers
zcp server list            # available servers
zcp currency list          # available currencies
zcp billing-cycle list     # billing-cycle slugs for --billing-cycle
zcp storage-category list  # storage-category slugs for --storage-category

# Plans by service type (preferred over legacy 'offering' commands)
zcp plan vm                # Virtual Machine plans
zcp plan storage           # Block Storage plans — shows storage category slug and pool per plan
zcp plan kubernetes        # Kubernetes plans
zcp plan lb                # Load Balancer plans
zcp plan router            # Virtual Router plans
zcp plan ip                # IP Address plans
zcp plan vm-snapshot       # VM Snapshot plans
zcp plan template          # My Template plans
zcp plan iso               # ISO plans
zcp plan backup            # Backup plans
zcp plan network           # network plan slugs for --network-plan
zcp plan object-storage    # Object Storage plans — slugs for object-storage create --plan

# Images and catalogs
zcp template list          # VM templates
zcp marketplace list       # Marketplace
zcp iso list               # ISO images
zcp store list             # Store
```

---

## Compute

```bash
# List and inspect
zcp instance list
zcp instance get <slug>

# Create — use --wait to block until the instance is Running
zcp instance create \
  --name my-vm \
  --project default \
  --region yow-1 \
  --template ubuntu-22f \
  --plan bp-4vc-8gb \
  --billing-cycle hourly \
  --storage-category nvme \
  --blockstorage-plan 50-gb-2 \
  --ssh-key mykey

zcp instance create ... --wait

# Lifecycle
zcp instance start <slug>
zcp instance stop <slug>
zcp instance reboot <slug>
zcp instance reset <slug>            # Hard reset (prompts for confirmation)

# Change plan (instance must be stopped)
zcp instance change-plan <slug> --plan <new-plan> --billing-cycle hourly

# Change hostname
zcp instance change-hostname <slug> --hostname new-hostname

# Change OS (DESTRUCTIVE — reinstalls the VM)
zcp instance change-os <slug> --template ubuntu-22f

# Change startup script
zcp instance change-script <slug> --user-data "#!/bin/bash\napt update"

# Change password
zcp instance change-password <slug> --password "newSecureP@ss"

# Add a network to a running instance
zcp instance add-network <slug> --network <network-slug>

# Activity logs
zcp instance logs <slug>

# Tags
zcp instance tag-create <slug> --key env --value prod
zcp instance tag-delete <slug> --key env

# Addons
zcp instance addons <slug>

# Open an SSH session directly from the CLI
zcp instance ssh <slug>
zcp instance ssh <slug> --user ubuntu
zcp instance ssh <slug> --user root --identity-file ~/.ssh/my-key.pem --port 2222

# Delete an instance permanently
zcp instance delete <slug>
zcp instance delete <slug> --yes                # skip confirmation
zcp instance delete <slug> --force --yes        # force-expunge from hypervisor immediately
```

The `--wait` flag on `create`, `start`, and `stop` polls the API until the instance
reaches the target state, printing progress to stderr.

---

## Storage

```bash
# Volumes
zcp volume list
zcp volume create \
  --name my-disk \
  --project default \
  --region yow-1 \
  --billing-cycle hourly \
  --storage-category nvme \
  --plan 50-gb-2
zcp volume create ... --vm <vm-slug>   # Attach on creation
zcp volume attach <volume-slug> --vm <vm-slug>
zcp volume detach <volume-slug>

# Snapshots
zcp snapshot list
zcp snapshot create \
  --volume <slug> \
  --name my-snapshot \
  --plan snapshot-per-gb \
  --region yow-1 \
  --billing-cycle hourly \
  --project default
zcp snapshot revert <snapshot-slug> --volume <volume-slug>

# VM snapshots (whole-instance checkpoint)
zcp vm-snapshot list
zcp vm-snapshot create \
  --vm <slug> \
  --name my-checkpoint \
  --plan basic \
  --billing-cycle hourly \
  --project default \
  --region yow-1
zcp vm-snapshot revert <slug>
```

---

## Networking

```bash
# Networks
zcp network list
zcp network get <slug>                           # provider state: CIDR, state, VPC, attached ACL
zcp plan network                                 # network plan slugs for --network-plan
zcp network create --name my-net --network-plan inet-yow --billing-cycle hourly \
  --region yow-1 --project default
zcp network update <slug> --name "New Name"
zcp network delete <slug>                        # also releases the SOURCE-NAT IP; use after VMs are removed

# VPC subnets (tiers) — optionally attach a custom ACL at creation
zcp network create --name web-tier --vpc <vpc-slug> --acl <acl-name> \
  --gateway 10.1.1.1 --netmask 255.255.255.0 --billing-cycle hourly \
  --region yow-1 --project default

# Public IP addresses
zcp ip list
zcp ip allocate --network <slug>
zcp ip release <slug>
zcp ip static-nat enable <slug> --instance <slug> --network <slug>

# Firewall rules (ingress)
zcp firewall list
zcp firewall create --ip <slug> --protocol tcp --start-port 80 --end-port 80

# Egress rules
zcp egress list
zcp egress create --network <slug> --protocol tcp

# Port forwarding
zcp portforward list
zcp portforward create \
  --ip <slug> \
  --protocol tcp \
  --public-port 2222 \
  --private-port 22 \
  --instance <slug>
```

---

## Advanced Networking

```bash
# VPCs
zcp vpc list
zcp vpc create \
  --name my-vpc \
  --region yow-1 \
  --project default \
  --plan vpc-1 \
  --network-address 10.1.0.1 \
  --size 16 \
  --billing-cycle hourly \
  --storage-category nvme

# Network ACL lists and rules
zcp acl list <vpc-slug>
zcp acl create <vpc-slug> --name web-acl --description "Web tier ACL"
zcp acl rules <vpc-slug> web-acl
zcp acl create-rule <vpc-slug> web-acl --number 1 --protocol tcp \
  --start-port 443 --end-port 443 --cidr 0.0.0.0/0          # --cidr takes comma-separated lists
zcp acl update-rule <vpc-slug> web-acl <rule-id> --number 1 --protocol tcp \
  --start-port 443 --end-port 443 --cidr 10.0.1.0/24,10.0.2.0/24
zcp acl delete-rule <vpc-slug> web-acl <rule-id>
zcp acl replace --network <network-slug> --acl web-acl --vpc <vpc-slug>
zcp acl delete <vpc-slug> web-acl

# Public load balancers
zcp loadbalancer list
zcp loadbalancer create --ip <slug> --name my-lb --algorithm roundrobin
zcp loadbalancer delete <slug>

# VPN gateways and connections
zcp vpn list
zcp vpn create --vpc <slug> --name my-vpn
zcp vpn delete <slug>
```

---

## Security and Access

```bash
# SSH keys
zcp ssh-key list
zcp ssh-key create --name mykey --public-key "$(cat ~/.ssh/id_rsa.pub)"
zcp ssh-key delete <slug>

# Affinity groups
zcp affinity-group list
zcp affinity-group create --name my-ag --type host-affinity
zcp affinity-group delete <slug>
```

---

## DNS

```bash
# Domains
zcp dns list
zcp dns show <slug>

# Create a domain (cloud provider + region are selected automatically)
zcp dns create --name example.com --project default

# Create a record
zcp dns record-create --domain <domain-slug> --name www --type A --content 192.0.2.1
zcp dns record-create --domain <domain-slug> --name mail --type MX --content mail.example.com --ttl 3600

# Delete a record or domain
zcp dns record-delete --domain <domain-slug> --record-id 42
zcp dns delete <domain-slug>
```

---

## Backup

```bash
zcp backup list
zcp backup get <slug>
zcp backup create --instance <slug> --name my-backup
zcp backup restore <slug>
zcp backup delete <slug>
```

---

## Autoscale

```bash
zcp autoscale list
zcp autoscale get <slug>
zcp autoscale create --name my-policy --min 1 --max 5 --region yow-1 --project default
zcp autoscale delete <slug>
```

---

## Monitoring

```bash
zcp monitoring list
zcp monitoring get <slug>
zcp monitoring create --instance <slug> --type cpu --threshold 80
zcp monitoring delete <slug>
```

---

## Project

```bash
zcp project list
zcp project create --name default --icon cloud-15 --purpose "Development"
zcp project update <slug> --name "New Name" --description "Updated description"
zcp project delete <slug>
zcp project dashboard <slug>

# Project users
zcp project user list <slug>
zcp project user add <slug> --email alice@example.com --role admin

# Project icons
zcp project icon list
```

---

## Kubernetes

```bash
# 'k8s' is accepted as an alias for 'kubernetes'
zcp kubernetes list
zcp kubernetes create \
  --name my-cluster \
  --version v1.36.1 \
  --plan k8s-li-yow-1 \
  --region yow-1 \
  --project default \
  --billing-cycle hourly \
  --workers 3 \
  --storage-category pro-nvme \
  --ssh-key mykey

# HA cluster with multiple control nodes
zcp kubernetes create \
  --name ha-cluster \
  --version v1.36.1 \
  --plan k8s-li-yow-1 \
  --region yow-1 \
  --project default \
  --billing-cycle hourly \
  --workers 3 \
  --control-nodes 3 \
  --ha \
  --storage-category pro-nvme \
  --ssh-key mykey

# Start / stop / upgrade
zcp kubernetes start <slug>
zcp kubernetes stop <slug>
zcp kubernetes upgrade <slug> --plan k8s-plan-2

# To cancel/delete a cluster, use billing cancel-service:
zcp billing cancel-service <subscription-slug> --service "Kubernetes" --reason not_needed_anymore
```

---

## Object Storage

```bash
# List and inspect
zcp object-storage list
zcp object-storage get <slug>

# Create an object storage instance — use an object-storage region (os-yul / os-yow)
# and a plan slug from `zcp plan object-storage`. The storage category is derived
# from the plan automatically, so you do not pass --storage-category.
zcp object-storage create \
  --name my-store \
  --project default \
  --region os-yow \
  --billing-cycle hourly \
  --plan o1100g

# Resize (change allocated GB)
zcp object-storage resize <slug> --storage-gb 200

# Show S3 credentials (access key + secret)
zcp object-storage credentials <slug>

# Delete an object storage instance
zcp object-storage delete <slug>

# Buckets
zcp object-storage bucket list <slug>
zcp object-storage bucket get <slug> <bucket-slug>
zcp object-storage bucket create <slug> --name my-bucket
zcp object-storage bucket delete <slug> <bucket-slug>

# Make a bucket public (anonymous read) or private again
zcp object-storage bucket set-acl <slug> <bucket-slug> --acl public-read
zcp object-storage bucket set-acl <slug> <bucket-slug> --acl private

# Object versioning
zcp object-storage bucket versioning status <slug> <bucket-slug>
zcp object-storage bucket versioning enable <slug> <bucket-slug>
zcp object-storage bucket versioning suspend <slug> <bucket-slug>

# Raw S3 bucket policy (fine-grained access; set-acl is the simple public/private button)
zcp object-storage bucket policy get <slug> <bucket-slug>
zcp object-storage bucket policy set <slug> <bucket-slug> --file policy.json
cat policy.json | zcp object-storage bucket policy set <slug> <bucket-slug> --file -
zcp object-storage bucket policy delete <slug> <bucket-slug>

# Bucket tags
zcp object-storage bucket tag get <slug> <bucket-slug>
zcp object-storage bucket tag set <slug> <bucket-slug> --tag env=prod --tag team=data
zcp object-storage bucket tag delete <slug> <bucket-slug>

# Default encryption (SSE-S3)
zcp object-storage bucket encryption status <slug> <bucket-slug>
zcp object-storage bucket encryption enable <slug> <bucket-slug>
zcp object-storage bucket encryption disable <slug> <bucket-slug>

# Lifecycle (auto-expire objects; current + old versions + incomplete uploads)
zcp object-storage bucket lifecycle expire <slug> <bucket-slug> --days 30 --prefix logs/
zcp object-storage bucket lifecycle expire <slug> <bucket-slug> --noncurrent-days 7 --abort-multipart-days 3
zcp object-storage bucket lifecycle get <slug> <bucket-slug>             # JSON; -o yaml for YAML
zcp object-storage bucket lifecycle delete <slug> <bucket-slug>

# Incomplete multipart uploads (storage consumed by failed large uploads)
zcp object-storage bucket uploads list <slug> <bucket-slug>
zcp object-storage bucket uploads abort <slug> <bucket-slug> <object-key>

# CORS (cross-origin access for browser apps)
zcp object-storage bucket cors set <slug> <bucket-slug> --origin '*' --method GET --method PUT --max-age 3600
zcp object-storage bucket cors get <slug> <bucket-slug>
zcp object-storage bucket cors delete <slug> <bucket-slug>

# Empty a bucket, or delete one that has object versions (versioning blocks plain delete)
zcp object-storage bucket empty <slug> <bucket-slug>
zcp object-storage bucket delete <slug> <bucket-slug> --purge

# Objects — list, inspect metadata, upload, download, share, delete
zcp object-storage object list <slug> <bucket>
zcp object-storage object get <slug> <bucket> <key>          # metadata only
zcp object-storage object put <slug> <bucket> ./photo.jpg
zcp object-storage object put <slug> <bucket> ./report.pdf --key reports/2026/q2.pdf --content-type application/pdf
zcp object-storage object download <slug> <bucket> <key>                 # writes ./<base-name>
zcp object-storage object download <slug> <bucket> images/logo.png --dest ./logo.png
zcp object-storage object put <slug> <bucket> ./data.bin --metadata owner=alice  # user metadata (x-amz-meta-*)
zcp object-storage object stat <slug> <bucket> <key>                     # full S3 metadata (size, content-type, ETag, meta)
zcp object-storage object url <slug> <bucket> <key>                      # pre-signed download URL (default 1h)
zcp object-storage object url <slug> <bucket> <key> --expires 24h        # share for 24h (max 168h)
zcp object-storage object put-url <slug> <bucket> <key> --expires 30m    # pre-signed UPLOAD url (curl -T)
zcp object-storage object copy <slug> <src-bucket> <src-key> <dst-bucket> <dst-key>   # server-side copy
zcp object-storage object move <slug> <src-bucket> <src-key> <dst-bucket> <dst-key>   # copy then delete source
zcp object-storage object tag set <slug> <bucket> <key> --tag env=prod   # object tags
zcp object-storage object tag get <slug> <bucket> <key>
zcp object-storage object tag delete <slug> <bucket> <key>
zcp object-storage object delete <slug> <bucket> <key>

# Versioning workflows (require `bucket versioning enable`)
zcp object-storage object versions <slug> <bucket> [--prefix p/]         # list versions + delete markers
zcp object-storage object download <slug> <bucket> <key> --version-id <id>
zcp object-storage object delete <slug> <bucket> <key> --version-id <id> # delete one version
zcp object-storage object restore <slug> <bucket> <key>                  # undelete (remove latest delete marker)
```

### Two backends — and what is CLI-only

Object storage spans two backends, and this determines what is reachable outside
the CLI:

- **ZCP REST API (also available in the Web UI / CMP):** instance lifecycle —
  `create`, `list`, `get`, `delete`, `resize`, `credentials` — and basic bucket
  management — `bucket create`, `bucket list`, `bucket get`, `bucket delete`.
  `object get` also goes through the REST API (it returns object metadata only).

- **Direct to the Ceph RADOS Gateway over the S3 protocol** (AWS Signature V4,
  via the [minio-go](https://github.com/minio/minio-go) client): **everything
  else.** This covers all remaining `object` operations (`list`, `put`,
  `download`, `url`, `put-url`, `stat`, `versions`, `restore`, `copy`, `move`,
  `tag`, `delete`) and all advanced `bucket` configuration (`set-acl`,
  `versioning`, `policy`, `tag`, `encryption`, `lifecycle`, `cors`, `uploads`,
  `empty`, and `bucket delete --purge`). The CLI derives the S3 endpoint and
  credentials from the same `object-storage get` response, so no separate S3
  configuration is needed.

> **CLI-only (not yet on the REST API or Web UI):** every S3-direct operation in
> the second group above is available **only through this CLI**. The CMP has not
> yet exposed these operations via the ZCP REST API or the Web UI, so they cannot
> be performed there — the CLI talks straight to Ceph RGW. Only the REST-backed
> operations in the first group are mirrored in the Web UI.

`object get` returns metadata only — use `object download` to fetch the contents,
or `object url` to mint a time-limited link a client can use without ZCP
credentials (works even when the bucket is private).

**Public vs private:** `bucket set-acl --acl public-read` applies an S3 bucket policy
that lets anyone download every object in the bucket; `--acl private` removes it. For
sharing a single object without making the whole bucket public, use `object url`.

ACL values for `bucket set-acl`: `private`, `public-read`, `public-read-write`.

---

## Billing and Admin

```bash
# Account balance and costs
zcp billing balance
zcp billing costs
zcp billing monthly-usage
zcp billing usage
zcp billing credit-limit
zcp billing service-counts
zcp billing free-credits

# Invoices and payments
zcp billing invoices
zcp billing invoices --page 2
zcp billing invoices-count
zcp billing payments

# Subscriptions
zcp billing subscriptions active
zcp billing subscriptions inactive
zcp billing contracts
zcp billing trials

# Cancel a service (instances, volumes, IPs, etc.)
zcp billing cancel-service <subscription-slug> --service "Virtual Machine" --reason not_needed_anymore
zcp billing cancel-service <subscription-slug> --service "Block Storage" --reason not_needed_anymore --type Immediate
zcp billing cancel-requests

# Coupons
zcp billing coupons
zcp billing redeem-coupon SAVE50

# Budget alerts
zcp billing budget-alert
zcp billing budget-alert-set --amount 500 --threshold 80 --enabled
```

---

## Support

```bash
zcp support list
zcp support get <ticket-id>
zcp support create --subject "Issue title" --description "Details"
zcp support close <ticket-id>
```

---

## Dashboard

```bash
zcp dashboard summary
zcp dashboard status
```

---

## Auth

```bash
# Validate that the active profile credentials are accepted by the API
zcp auth validate
```

---

## Output Formats

All listing commands support three output formats controlled by the `--output`
(or `-o`) flag. The keys in JSON/YAML are the lowercased column headers.

**Table (default)**

```bash
zcp region list
```

```
SLUG    NAME    COUNTRY  CONTINENT      STATUS
yul-1   YUL-1   Canada   North America  active
yow-1   YOW-1   Canada   North America  active
```

**JSON**

```bash
zcp region list --output json
```

```json
[
  {
    "slug": "yow-1",
    "name": "YOW-1",
    "country": "Canada",
    "continent": "North America",
    "status": "active"
  }
]
```

**YAML**

```bash
zcp region list --output yaml
```

```yaml
- slug: yow-1
  name: YOW-1
  country: Canada
  continent: North America
  status: active
```
