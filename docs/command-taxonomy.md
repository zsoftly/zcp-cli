# ZCP CLI Command Taxonomy (v0.0.6)

**CLI name**: `zcp`
**Base URL**: `https://portal.webberstop.com/backend/api`
**Authentication**: Bearer token (`--bearer-token` during profile add)

---

## Global Flags

These flags apply to every command in the CLI:

| Flag             | Type   | Default          | Description                                   |
| ---------------- | ------ | ---------------- | --------------------------------------------- |
| `--profile`      | string | (active profile) | Named credential profile to use               |
| `--output`, `-o` | string | `table`          | Output format: `table`, `json`, or `yaml`     |
| `--api-url`      | string | (from profile)   | Override the API base URL                     |
| `--timeout`      | int    | `30`             | HTTP request timeout in seconds               |
| `--debug`        | bool   | `false`          | Print HTTP request/response details to stderr |
| `--no-color`     | bool   | `false`          | Disable terminal color output                 |
| `--pager`        | bool   | `false`          | Pipe table output through less                |

---

## Output Format Conventions

- **table** (default) -- human-readable fixed-width columns; intended for interactive use
- **json** -- raw JSON response object; suitable for `jq` pipelines and scripting
- **yaml** -- YAML rendering of the same response object; suitable for config-driven workflows
- All three formats are fully machine-parseable: no extra prose is written to stdout
- Errors are always written to stderr regardless of `--output`
- Exit codes: `0` = success, `1` = API error or CLI error

---

## Full Command Tree

```
zcp
├── version                            Print CLI version and build info
├── completion <shell>                 Generate shell completion script (bash/zsh/fish/powershell)
│
├── profile                            Manage credential profiles
│   ├── add                            Add a new named profile (prompts for --bearer-token)
│   ├── list                           List all saved profiles
│   ├── use                            Set the active default profile
│   ├── delete                         Remove a profile
│   ├── show                           Show profile details (redacts token)
│   ├── update                         Update an existing profile
│   └── rename                         Rename a profile
│
├── auth                               Authentication utilities
│   └── validate                       Validate that the active profile credentials are accepted
│
├── region                             Region operations
│   └── list                           List available regions
│
├── template                           VM template operations
│   ├── list                           List available public templates
│   ├── account-list                   List account (user-created) templates
│   ├── account-create                 Create an account template
│   └── account-delete                 Delete an account template
│
├── instance                           VM instance operations
│   ├── list                           List instances
│   ├── get                            Show details for a single instance
│   ├── create                         Create a new instance
│   ├── start                          Start a stopped instance
│   ├── stop                           Stop a running instance
│   ├── reboot                         Reboot a running instance
│   ├── reset                          Reset (hard reboot) an instance
│   ├── logs                           View instance console/activity logs
│   ├── tag-create                     Add a tag to an instance
│   ├── tag-delete                     Remove a tag from an instance
│   ├── change-hostname                Change the hostname of an instance
│   ├── change-password                Change the root password of an instance
│   ├── change-plan                    Resize an instance (change compute plan)
│   ├── change-os                      Reinstall with a different OS template
│   ├── change-script                  Update the startup script on an instance
│   ├── add-network                    Attach an additional network to an instance
│   ├── addons                         List available addons for an instance
│   ├── purchase-addon                 Purchase an addon for an instance
│   └── ssh                            Open an SSH session to an instance
│
├── volume                             Block storage volume operations
│   ├── list                           List volumes
│   ├── create                         Create a new volume
│   ├── attach                         Attach a volume to an instance
│   └── detach                         Detach a volume from an instance
│
├── snapshot                           Block storage snapshot operations
│   ├── list                           List snapshots
│   ├── create                         Create a snapshot of a volume
│   └── revert                         Revert a volume to a snapshot (destructive)
│
├── vm-snapshot                        VM (instance-level) snapshot operations
│   ├── list                           List VM snapshots
│   ├── create                         Create a VM snapshot
│   ├── delete                         Delete a VM snapshot
│   └── revert                         Revert an instance to a VM snapshot
│
├── network                            Isolated network operations
│   ├── list                           List networks
│   ├── create                         Create a network
│   ├── update                         Update a network
│   └── categories                     List network categories
│
├── vpc                                VPC operations
│   ├── list                           List VPCs (--zone filter)
│   ├── get                            Get a VPC by slug
│   ├── create                         Create a VPC
│   ├── update                         Update VPC name/description
│   ├── delete                         Delete a VPC
│   ├── restart                        Restart a VPC
│   ├── acl-list                       List ACL rules for a VPC
│   ├── acl-create-rule                Create an ACL rule in a VPC
│   ├── acl-replace                    Replace the ACL on a VPC network
│   └── vpn-gateway                    VPN gateway operations within a VPC
│       ├── list                       List VPN gateways
│       ├── create                     Create a VPN gateway
│       └── delete                     Delete a VPN gateway
│
├── acl                                Network ACL operations
│   ├── list                           List network ACLs
│   ├── create-rule                    Create an ACL rule
│   └── replace                        Replace the ACL on a network
│
├── ip                                 Public IP address operations
│   ├── list                           List IP addresses (--vpc filter)
│   ├── allocate                       Allocate a new public IP address
│   ├── static-nat                     Static NAT operations
│   │   └── enable                     Enable static NAT for an IP
│   └── vpn                            Remote access VPN on an IP
│       ├── list                       List VPN users for an IP
│       ├── enable                     Enable remote VPN access on an IP
│       └── disable                    Disable remote VPN access on an IP
│
├── firewall                           Firewall rule operations
│   ├── list                           List firewall rules (--ip required)
│   ├── create                         Create a firewall rule
│   └── delete                         Delete a firewall rule
│
├── egress                             Egress rule operations
│   ├── list                           List egress rules
│   ├── create                         Create an egress rule
│   └── delete                         Delete an egress rule
│
├── portforward                        Port forwarding rule operations
│   ├── list                           List port forwarding rules
│   ├── create                         Create a port forwarding rule
│   └── delete                         Delete a port forwarding rule
│
├── loadbalancer                       Load balancer operations
│   ├── list                           List load balancers
│   ├── create                         Create a load balancer
│   ├── create-rule                    Create a load balancer rule
│   └── attach-vm                      Attach a VM to a load balancer rule
│
├── ssh-key                            SSH key operations
│   ├── list                           List SSH keys
│   ├── import                         Import an SSH public key
│   └── delete                         Delete an SSH key
│
├── vpn                                VPN operations
│   ├── customer-gateway               VPN customer gateway operations
│   │   ├── list                       List VPN customer gateways
│   │   ├── create                     Add a VPN customer gateway
│   │   ├── update                     Update a VPN customer gateway
│   │   └── delete                     Delete a VPN customer gateway
│   └── user                           VPN remote-access user operations
│       ├── list                       List VPN users
│       ├── create                     Add a VPN user
│       └── delete                     Delete a VPN user
│
├── kubernetes (alias: k8s)            Kubernetes cluster operations
│   ├── list                           List Kubernetes clusters
│   ├── create                         Create a Kubernetes cluster
│   ├── start                          Start a stopped cluster
│   ├── stop                           Stop a running cluster
│   └── upgrade                        Upgrade a Kubernetes cluster version
│
├── dns                                DNS domain and record operations
│   ├── list                           List DNS domains
│   ├── show                           Show DNS domain details and records
│   ├── create                         Create a DNS domain
│   ├── delete                         Delete a DNS domain (removes all records)
│   ├── record-create                  Create a DNS record
│   └── record-delete                  Delete a DNS record
│
├── project                            Project management
│   ├── list                           List all projects
│   ├── create                         Create a new project
│   ├── update                         Update an existing project
│   ├── dashboard                      Show project dashboard services
│   ├── icon                           Project icon operations
│   │   └── list                       List available project icons
│   └── user                           Project user operations
│       ├── list                       List users in a project
│       └── add                        Add a user to a project
│
├── monitoring                         Resource monitoring and VM metrics
│   ├── global                         Show global resource monitoring overview
│   ├── cpu                            Show CPU usage metrics for a VM
│   ├── memory                         Show memory usage metrics for a VM
│   ├── disk                           Show disk read/write metrics for a VM
│   ├── disk-io                        Show disk IO read/write metrics for a VM
│   ├── network                        Show network traffic metrics for a VM
│   └── charts                         Show monitoring charts data
│
├── billing                            Billing, costs, usage, invoices, and payments
│   ├── balance                        Show account balance summary
│   ├── costs                          Show per-service cost breakdown
│   ├── monthly-usage                  Show month-by-month usage history
│   ├── service-counts                 Show active service counts by type
│   ├── credit-limit                   Show account credit limit
│   ├── invoices                       List billing invoices (--page)
│   ├── invoices-count                 Show total number of invoices
│   ├── usage                          Show detailed account usage
│   ├── free-credits                   Show available free credits
│   ├── subscriptions                  Subscription management
│   │   ├── active                     List active service subscriptions
│   │   └── inactive                   List inactive service subscriptions
│   ├── contracts                      List service contracts
│   ├── trials                         List active free trials
│   ├── cancel-requests                List scheduled cancellation requests
│   ├── cancel-service                 Submit a service cancellation request
│   ├── payments                       List payment transactions (--page)
│   ├── coupons                        List coupons associated with the account
│   ├── redeem-coupon                  Apply a coupon code to the account
│   ├── budget-alert                   Show current budget alert settings
│   └── budget-alert-set               Configure budget alert settings
│
├── support                            Support tickets and FAQs
│   ├── ticket                         Ticket management
│   │   ├── list                       List support tickets
│   │   ├── create                     Create a support ticket
│   │   ├── show                       Show a support ticket
│   │   ├── delete                     Delete a support ticket
│   │   ├── summary                    Show ticket count summary
│   │   ├── reply                      Reply to a support ticket
│   │   ├── replies                    List replies for a support ticket
│   │   ├── feedback                   Get feedback for a support ticket
│   │   └── feedback-submit            Submit feedback for a support ticket
│   └── faq                            FAQ management
│       └── list                       List FAQs
│
├── autoscale                          VM autoscale groups, policies, and conditions
│   ├── list                           List autoscale groups
│   ├── create                         Create a new autoscale group
│   ├── enable                         Enable an autoscale group
│   ├── disable                        Disable an autoscale group
│   ├── change-plan                    Change the compute plan of a group
│   ├── change-template                Change the template of a group
│   ├── policy                         Scale-up policy management
│   │   ├── create                     Create a scale-up policy
│   │   ├── update                     Update a scale-up policy
│   │   └── delete                     Delete a scale-up policy
│   └── condition                      Scale-down condition management
│       ├── create                     Create a scale-down condition
│       ├── update                     Update a scale-down condition
│       └── delete                     Delete a scale-down condition
│
├── dashboard                          Account dashboard and service management
│   ├── summary                        Show a summary of active service counts
│   └── cancel-service                 Submit a service cancellation request
│
├── plan                               List service plans and pricing
│   ├── vm                             List Virtual Machine plans
│   ├── router                         List Virtual Router plans
│   ├── storage                        List Block Storage plans
│   ├── lb                             List Load Balancer plans
│   ├── kubernetes (alias: k8s)        List Kubernetes plans
│   ├── ip                             List IP Address plans
│   ├── vm-snapshot                    List VM Snapshot plans
│   ├── template                       List My Template plans
│   ├── iso                            List ISO plans
│   └── backup                         List Backup plans
│
├── store                              Store and checkout
│   ├── list                           List store items (--sort, --page, --limit)
│   └── checkout                       Purchase a store product
│
├── marketplace (alias: apps)          Marketplace applications
│   └── list                           List marketplace applications (--region, --include)
│
├── product                            Products and product categories
│   ├── categories                     List product categories
│   └── list                           List all products (--card-type, --card-slug, --include)
│
├── iso                                ISO image management
│   ├── list                           List ISO images
│   ├── create                         Create (register) an ISO image
│   ├── update                         Update ISO permissions
│   └── delete                         Delete an ISO image
│
├── affinity-group                     Affinity group management
│   ├── list                           List affinity groups
│   ├── create                         Create an affinity group
│   └── delete                         Delete an affinity group
│
├── backup                             Block storage backup operations
│   ├── list                           List block storage backups
│   └── create                         Create a block storage backup
│
├── profile-info                       User profile management (2FA status shown via get, not managed)
│   ├── get                            Show user profile (includes 2FA status)
│   ├── update                         Update user profile
│   ├── company                        Update company/billing details
│   ├── time-settings                  Update time/timezone settings
│   ├── enable-api                     Enable API access
│   ├── disable-api                    Disable API access
│   ├── login-activity <crn>           Show login activity for a CRN
│   └── activity-logs <crn>            Show activity logs for a CRN
│
├── vm-backup                          VM backup operations
│   ├── list                           List VM backups
│   └── create                         Create a VM backup
│
├── cloud-provider                     Cloud provider operations
│   └── list                           List available cloud providers
│
├── server                             Server operations
│   └── list                           List available servers
│
├── currency                           Currency operations
│   └── list                           List available currencies
│
├── billing-cycle                      Billing cycle operations
│   └── list                           List available billing cycles
│
└── storage-category                   Storage category operations
    └── list                           List available storage categories
```

---

## Authentication

Profiles store a **bearer token** for authenticating with the ZCP API. When creating
a profile, the token is provided via `--bearer-token` or entered interactively:

```
zcp profile add default
  Bearer Token: ********
```

Each API request sends the token as an `Authorization: Bearer <token>` header.

---

## Identifier Conventions

v0.0.6 uses **slug-based identifiers** for most resources. Slugs are human-readable
strings assigned by the API (e.g., `my-vm-123`, `root-4153`, `example-com-1`).

| Context         | Flag / Argument                   | Example                                  |
| --------------- | --------------------------------- | ---------------------------------------- |
| VM instance     | positional `<slug>` or `--vm`     | `zcp instance get my-vm-123`             |
| Volume          | `--volume`                        | `zcp snapshot create --volume root-4153` |
| DNS domain      | positional `<slug>` or `--domain` | `zcp dns show example-com-1`             |
| Project         | `--project`                       | `--project default-60`                   |
| Region          | `--region`                        | `--region yow-1`                         |
| VPC             | `--vpc`                           | `zcp ip list --vpc my-vpc`               |
| IP              | `--ip`                            | `zcp firewall list --ip my-ip-slug`      |
| Autoscale group | positional `<slug>`               | `zcp autoscale enable web-group`         |

All commands use slug-based identifiers.

---

## CLI Group to API Path Mapping

| CLI Group          | API Source        | Notes                                                    |
| ------------------ | ----------------- | -------------------------------------------------------- |
| `region`           | ZCP API (STKCNSL) | Region listing                                           |
| `plan`             | ZCP API (STKCNSL) | Service plans for all resource types                     |
| `template`         | ZCP API (STKCNSL) | Public and account templates                             |
| `instance`         | ZCP API (STKCNSL) | Full VM lifecycle                                        |
| `volume`           | ZCP API (STKCNSL) | Block storage CRUD                                       |
| `snapshot`         | ZCP API (STKCNSL) | Block storage snapshots                                  |
| `vm-snapshot`      | ZCP API (STKCNSL) | VM-level snapshots                                       |
| `network`          | ZCP API (STKCNSL) | Isolated networks                                        |
| `vpc`              | ZCP API (STKCNSL) | VPCs, ACLs, VPN gateways                                 |
| `acl`              | ZCP API (STKCNSL) | Network ACLs                                             |
| `ip`               | ZCP API (STKCNSL) | Public IPs, static NAT, VPN                              |
| `firewall`         | ZCP API (STKCNSL) | Firewall rules                                           |
| `egress`           | ZCP API (STKCNSL) | Egress rules                                             |
| `portforward`      | ZCP API (STKCNSL) | Port forwarding rules                                    |
| `loadbalancer`     | ZCP API (STKCNSL) | Load balancers and rules                                 |
| `ssh-key`          | ZCP API (STKCNSL) | SSH key management                                       |
| `vpn`              | ZCP API (STKCNSL) | Customer gateways, VPN users                             |
| `kubernetes`       | ZCP API (STKCNSL) | Kubernetes clusters                                      |
| `dns`              | ZCP API (STKCNSL) | DNS domains and records                                  |
| `project`          | ZCP API (STKCNSL) | Projects, icons, users                                   |
| `monitoring`       | ZCP API (STKCNSL) | Global and per-VM metrics                                |
| `billing`          | ZCP API (STKCNSL) | Balance, costs, invoices, subscriptions, coupons, budget |
| `support`          | ZCP API (STKCNSL) | Tickets, replies, feedback, FAQs                         |
| `autoscale`        | ZCP API (STKCNSL) | Autoscale groups, policies, conditions                   |
| `dashboard`        | ZCP API (STKCNSL) | Service counts, cancellations                            |
| `store`            | ZCP API (STKCNSL) | Store items and checkout                                 |
| `marketplace`      | ZCP API (STKCNSL) | Marketplace app listings                                 |
| `product`          | ZCP API (STKCNSL) | Product categories and catalog                           |
| `iso`              | ZCP API (STKCNSL) | ISO image management                                     |
| `affinity-group`   | ZCP API (STKCNSL) | Affinity/anti-affinity groups                            |
| `backup`           | ZCP API (STKCNSL) | Block storage backups                                    |
| `vm-backup`        | ZCP API (STKCNSL) | VM backups                                               |
| `profile-info`     | ZCP API (STKCNSL) | User profile, company, 2FA, time settings, API access    |
| `cloud-provider`   | ZCP API (STKCNSL) | Cloud provider listing                                   |
| `server`           | ZCP API (STKCNSL) | Server listing                                           |
| `currency`         | ZCP API (STKCNSL) | Currency listing                                         |
| `billing-cycle`    | ZCP API (STKCNSL) | Billing cycle listing                                    |
| `storage-category` | ZCP API (STKCNSL) | Storage category listing                                 |

---

## Async Operation Handling

All API responses use the STKCNSL `{status, message, data}` envelope format. List responses include pagination metadata.

---

## Notes

- All commands use slug-based identifiers (not UUIDs).
- Destructive commands (`delete`, `revert`, `cancel-service`) prompt for confirmation. Use `--auto-approve` / `-y` to skip (useful for CI/CD).
- The `--timeout` flag controls the maximum time for API requests (default: 30s).
