# zcp v1.3.0 Release Notes

zcp v1.3.0 makes credential selection predictable and fixes subscription
listing when the API returns a numeric price.

## Highlights

**Clear credential selection.** `ZCP_BEARER_TOKEN` remains the preferred
credential for CI and scripted use. A named profile must exist when you select
it with `--profile` or `ZCP_PROFILE`, even if `ZCP_BEARER_TOKEN` is set. This
catches a misspelled profile before a command runs with the environment token.
If the API rejects the selected credential, zcp reports that failure and does
not try another profile.

```bash
# Use the active profile.
zcp auth validate

# Select one configured profile for this command.
zcp auth validate --profile staging

# Use a token supplied by the environment for automation.
export ZCP_BEARER_TOKEN='<token>'
zcp auth validate
```

`zcp auth validate` states whether it used `ZCP_BEARER_TOKEN` or a profile. It
never prints the token.

**Reliable subscription listings.** `zcp billing subscriptions inactive` now
accepts a subscription price returned by the API as either a JSON number or a
JSON string. Numeric prices keep their reported precision in CLI output.

```bash
zcp billing subscriptions inactive
```

## Fixed

- A missing profile selected by `--profile` or `ZCP_PROFILE` no longer falls
  back to `ZCP_BEARER_TOKEN`.
- Subscription prices encoded as JSON numbers no longer cause inactive
  subscription listings to fail.
- Escaped JSON price strings decode correctly. Invalid JSON types for a price
  return a clear error.

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
zcp version   # zcp version v1.3.0
```
