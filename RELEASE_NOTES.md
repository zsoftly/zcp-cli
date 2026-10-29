# zcp v1.2.0 Release Notes

zcp v1.2.0 adds optional server-side encryption for object storage. You can
encrypt uploads, copies and moves with the standard S3 settings, set a default
encryption for a bucket, and confirm that each encrypted copy landed encrypted.
Moves also keep the source safe when a copy fails. The examples use the region
and project in your active profile. Replace each quoted placeholder with a value
from your account.

## Highlights

**Encrypted uploads.** `--sse AES256` encrypts an object with a key the service
manages for your bucket (SSE-S3). `--sse aws:kms --sse-kms-key-id` encrypts it
with a service-managed key ID (SSE-KMS). Downloads decrypt automatically for
anyone allowed to read the object.

```bash
zcp object-storage object put "<store-slug>" "<bucket>" ./report.pdf --sse AES256
zcp object-storage object put "<store-slug>" "<bucket>" ./report.pdf \
  --sse aws:kms --sse-kms-key-id "<service-key-id>"
```

**Bucket default encryption.** A bucket default encrypts every later upload that
does not ask for its own setting. Existing objects and versions stay as they are.

```bash
zcp object-storage bucket encryption enable "<store-slug>" "<bucket-slug>"
zcp object-storage bucket encryption enable "<store-slug>" "<bucket-slug>" \
  --sse aws:kms --sse-kms-key-id "<service-key-id>"
zcp object-storage bucket encryption status "<store-slug>" "<bucket-slug>"
zcp object-storage bucket encryption disable "<store-slug>" "<bucket-slug>"
```

**Verified encrypted copies and moves.** With `--sse`, `copy` and `move` use a
multipart copy and then check the destination's encryption before they finish.
Without `--sse`, a copy of an encrypted object keeps the source's encryption.

```bash
zcp object-storage object copy "<store-slug>" "<src-bucket>" "<src-key>" \
  "<dst-bucket>" "<dst-key>" --sse AES256
zcp object-storage object move "<store-slug>" "<src-bucket>" "<src-key>" \
  "<dst-bucket>" "<dst-key>" --sse aws:kms --sse-kms-key-id "<service-key-id>"
```

### Things to know

- Encryption depends on the region's key management service. Check availability
  with ZCP support before you rely on it.
- Other S3 clients, such as the AWS CLI, must use a multipart copy to produce an
  encrypted copy, or download and re-upload the object. The service refuses a
  single-request copy that asks for encryption.
- A copy made without encryption settings does not pick up the destination
  bucket's default.
- zcp never re-encrypts existing objects for you, and in-place re-encryption is
  not supported.

## Fixed

- A move keeps its source when the copy fails, returns an incomplete result, or
  fails the encryption check.
- When an encryption check fails, zcp deletes only the destination version this
  command wrote. For unversioned buckets it leaves the destination in place and
  warns you instead.
- Encrypted copies keep Expires, Cache-Control, Content-Disposition,
  Content-Encoding, Content-Language, Content-Type, user metadata and tags.
- Copy errors that the service returns inside a successful HTTP response now
  keep their error code and message.
- Bucket encryption commands accept the bucket slug shown by
  `zcp object-storage bucket list`.

## Installation and upgrade

The install scripts install the latest release and upgrade an existing
installation in place.

**Linux / macOS**

```bash
curl -fsSL https://github.com/zsoftly/zcp-cli/releases/latest/download/install.sh | bash
```

**Windows (PowerShell)**

```powershell
irm https://github.com/zsoftly/zcp-cli/releases/latest/download/install.ps1 | iex
```

**Manual download:** download the binary or archive for your platform from the
[Releases](https://github.com/zsoftly/zcp-cli/releases) page. Extract an archive
first. Rename the platform binary to `zcp` (or `zcp.exe` on Windows). On Linux
or macOS, run `chmod +x zcp`, then place it on your `PATH`.

**Verify:**

```bash
zcp version   # zcp version v1.2.0
```
