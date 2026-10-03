# zcp v0.0.30 Release Notes

v0.0.30 adds custom VM plans, load balancer rule listing, and object-storage
key management. It also aligns object-storage reads and S3 commands with the
current platform API response shape.

## Highlights

**Create a custom VM plan.** Omit `--plan` and provide a CPU count plus memory
and disk in GB. Existing catalogue-plan creates keep their current form.

```bash
zcp instance create \
  --name my-custom-vm \
  --project default-9 \
  --region yul-1 \
  --template ubuntu-2604-lts-1 \
  --cpu 2 \
  --memory 4 \
  --disk 45 \
  --billing-cycle hourly \
  --network-plan pnet-yul \
  --storage-category pro-nvme
```

**List load balancer rules before attaching a VM.** The output includes each
rule ID required by `attach-vm` and `detach-vm`.

```bash
zcp loadbalancer list-rule <load-balancer-slug>
zcp loadbalancer attach-vm <load-balancer-slug> <rule-id> --vm <vm-slug>
```

**Manage object-storage access keys.** A store can have one or two active
keys. Create a second key, copy its secret within five minutes, update your
applications, then revoke the old key. The platform blocks a third active key
and revocation of the final active key.

```bash
zcp object-storage keys list <store-slug>
zcp object-storage keys create <store-slug> --output json
# Store the returned pair in your secret manager, then update consumers.
zcp object-storage keys delete <store-slug> <old-key-id> -y
```

The CLI only displays plaintext credentials while the platform reports their
visibility window as open. It does not decrypt or recover a secret after its
visibility window closes.
`object-storage create --output json` includes its initial key only when the
platform exposes that secret.
For direct S3 commands, set both variables from the pair you saved when the
key was created. The CLI verifies that the access key is active for the named
store before using it.

```bash
# Example values only. Keep the secret out of shell history where possible.
export ZCP_S3_ACCESS_KEY='<access-key>'
export ZCP_S3_SECRET_KEY='<secret-key>'
zcp object-storage bucket versioning status <store-slug> <bucket-name>
```

## Fixed

- Object-storage list and get output now reads allocation from the attached
  offering and usage from the current statistics field. Missing status values
  display as `-`; normal reads do not print secrets.
- Debug redaction now covers `api_secret`.

## Go library consumers

`loadbalancer.Service.Get(ctx, slug)` is available for library consumers.
Object-storage users should review serialized output before upgrading.
`ObjectStorage.APIKey` and `APISecret` are excluded from JSON and YAML encoding
and decoding. `OSStats.TotalSize` is now `int64`, and storage allocation is
reported in `Offering.Storage`. Object-storage YAML output now uses API-style
snake_case field names. Direct S3 methods require the
`ZCP_S3_ACCESS_KEY` and `ZCP_S3_SECRET_KEY` environment-variable pair. The
object-storage service also adds key-management methods.

---

## Installation and upgrade

The install script installs the latest release and upgrades an existing
installation in place.

**Linux / macOS**

```bash
curl -fsSL https://github.com/zsoftly/zcp-cli/releases/latest/download/install.sh | bash
```

**Windows (PowerShell)**

```powershell
irm https://github.com/zsoftly/zcp-cli/releases/latest/download/install.ps1 | iex
```

**Manual download:** grab your platform's binary from the
[Releases](https://github.com/zsoftly/zcp-cli/releases) page, `chmod +x`, and
place it on your `PATH`.

**Verify:**

```bash
zcp version   # zcp version v0.0.30
```

First-time setup after installing:

```bash
zcp profile add default --region yul-1 --project default-9   # prompts for bearer token
zcp auth validate
```
