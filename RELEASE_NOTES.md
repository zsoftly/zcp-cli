# zcp 0.0.12 Release Notes

## New

### `zcp kubernetes upgrade-version` — upgrade Kubernetes cluster version

Upgrade a running cluster from one Kubernetes version to the next:

```bash
zcp kubernetes upgrade-version my-cluster --version v1.35.1
zcp kubernetes upgrade-version my-cluster --version v1.36.1
```

The CLI resolves the correct version slug for the cluster's region automatically — no need to look up region-specific slugs manually. An error is returned if the requested version is not available in the cluster's region.

Supported upgrade path (must be sequential — no skipping):

```
v1.34.3 → v1.35.1 → v1.36.1
```

---

## Bug Fixes

### `zcp kubernetes scale` — misleading state-guard error

The pre-flight check that blocks scale operations on non-running clusters was emitting:

```
cluster "my-cluster" is in state "Stopped" — scale requires Running state
```

The guard actually accepts both `Running` and `Scaling`. The error message now reads:

```
cluster "my-cluster" is in state "Stopped" — scale requires Running or Scaling state
```

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
# zcp version 0.0.12
```

---

## Full Changelog

https://github.com/zsoftly/zcp-cli/compare/0.0.11...0.0.12
