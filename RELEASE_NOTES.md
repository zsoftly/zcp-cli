# zcp 0.0.10 Release Notes

## What's New

### Object Storage

Full Ceph/S3 object storage management is now built into the CLI.

```bash
# Storage instances
zcp object-storage list
zcp object-storage create --name my-storage --region yul-1 --billing-cycle hourly --storage-gb 100
zcp object-storage create --name my-storage --region yul-1 --billing-cycle hourly --storage-gb 100 --project my-project

# Buckets
zcp object-storage bucket list my-storage
zcp object-storage bucket create my-storage --name my-bucket

# Objects
zcp object-storage object list my-storage my-bucket
zcp object-storage object upload my-storage my-bucket ./report.pdf
zcp object-storage object delete my-storage my-bucket report.pdf

# S3 credentials (access key + secret + endpoint)
zcp object-storage credentials my-storage
```

`--cloud-provider` defaults to `ceph` and should be omitted — object storage is backed by Ceph RGW, not the CloudStack compute provider (`nimbo`). The upload and object-list commands talk directly to Ceph's S3-compatible API via the credentials stored on the instance — no separate `aws` CLI needed.

---

### Delete operations across all major resources

Every resource type now has a `delete` command. All prompts require an explicit `y` or `yes` confirmation (or `--yes` to skip in scripts).

| Command                                          | What it deletes                                    |
| ------------------------------------------------ | -------------------------------------------------- |
| `zcp instance delete <slug>`                     | Virtual machine (`--force` for immediate expunge)  |
| `zcp volume delete <slug>`                       | Block storage volume (detach first)                |
| `zcp snapshot delete <slug>`                     | Block storage snapshot                             |
| `zcp backup delete <slug>`                       | Block storage backup schedule                      |
| `zcp vm-backup delete <slug>`                    | VM backup schedule                                 |
| `zcp network delete <slug>`                      | Isolated network (also releases its SOURCE-NAT IP) |
| `zcp vpc delete <slug>`                          | VPC                                                |
| `zcp kubernetes delete <slug>`                   | Kubernetes cluster                                 |
| `zcp lb delete-rule <lb-slug> <rule-id>`         | Load balancer rule                                 |
| `zcp lb detach-vm <lb-slug> <rule-id> <vm-slug>` | VM attachment from LB rule                         |

---

### Kubernetes — scale and kubeconfig download

```bash
# Scale workers (fires and returns a follow-up command to check)
zcp kubernetes scale my-cluster --workers 5

# Scale and wait until done (10-min timeout)
zcp kubernetes scale my-cluster --workers 3 --wait

# Print kubeconfig to stdout
zcp kubernetes get-config my-cluster

# Save to a file (creates the directory if needed)
zcp kubernetes get-config my-cluster --output ~/.kube/zcp-yow
```

`kubernetes get` also now shows accurate values for version, worker count, IP, and API endpoint — it previously showed zeros for Running clusters because the data lives in CloudStack meta fields, not top-level API fields.

---

### Smoke testing framework

A full suite of live end-to-end test scripts is now in `tests/smoke/`. Run against a real ZCP account to validate the full resource lifecycle:

```bash
cd tests/smoke
export ZCP_BEARER_TOKEN=your-token
bash smoke.sh --lifecycle      # full create/verify/delete cycle
bash smoke.sh --service k8s    # test a specific service only
```

---

## Bug Fixes

- **`vm-backup create`** — `--psudo-service` flag misspelling corrected to `--pseudo-service`
- **`egress create`** — protocol values normalized to uppercase (`tcp` → `TCP`); the API rejected lowercase silently
- **`kubernetes scale --wait`** — previously used an unbounded `context.Background()` loop; now respects the command context with a 10-minute hard cap; non-transient states (e.g. `Error`) return an error immediately
- **Confirmation prompts** — 15 destructive prompts across 9 command files were using `fmt.Scanln` on stdout, accepting only `y`. All now use `bufio.Scanner` on stderr and accept both `y` and `yes`

---

## Breaking Changes

- **`zcp vm-backup create`** — the `--psudo-service` flag has been renamed to `--pseudo-service`. If you have scripts using the old name, update them.

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
# zcp version 0.0.10
```

---

## Full Changelog

https://github.com/zsoftly/zcp-cli/compare/0.0.9...0.0.10
