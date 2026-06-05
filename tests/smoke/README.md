# ZCP CLI live smoke suite

Real-world smoke tests that drive the **shipped `zcp` binary** against a **real
ZCP API**. Every release (and every PR that touches a service) is checked so
that a broken read, decode, or create/destroy path is caught for real — not
mocked.

This complements the Go integration test in `tests/integration/`, which
exercises the internal API packages. The smoke suite exercises the **compiled
CLI** exactly as a user runs it: flag parsing, output formatting, exit codes.

## What it does

For each selected service:

| Layer                         | What runs                                      | Safety                                           |
| ----------------------------- | ---------------------------------------------- | ------------------------------------------------ |
| **read** (always)             | the `list` / `get` paths (`zcp <svc> list`, …) | read-only, no mutation                           |
| **lifecycle** (`--lifecycle`) | `create → verify → destroy` of real resources  | **creates billable resources**, always torn down |

The read sweep alone catches the most common breakage: auth failures, dead
endpoints, timeouts, and JSON-decode regressions (a field that changes type
server-side). The lifecycle layer additionally proves the create/cancel paths
still work end to end, and every created resource is registered on a cleanup
stack that is drained in reverse order on exit (even on Ctrl-C / failure), using
the service's native `delete` or `billing cancel-service` where there is none.

## Requirements

- `bash`, `jq`, `curl`, `ssh-keygen` on PATH
- the `zcp` binary (`make build` → `./bin/zcp`)
- `ZCP_BEARER_TOKEN` for a **disposable** account (lifecycle creates/destroys
  real resources and incurs real, if tiny, cost)

## Run it

```bash
make build                       # produces ./bin/zcp
export ZCP_BEARER_TOKEN='<api-key>'

# read-only sweep of every service (~30–60s, zero resources created)
tests/smoke/smoke.sh --bin ./bin/zcp

# full create → verify → destroy for every service (real resources, minutes)
tests/smoke/smoke.sh --bin ./bin/zcp --lifecycle

# scope to specific services
tests/smoke/smoke.sh --bin ./bin/zcp --lifecycle --only instance,ip,network

# list the service catalogue
tests/smoke/smoke.sh --list
```

Exit code is `0` only if every selected case passed (`1` on any failure, `2` on
setup error). Cleanup always runs.

## Configuration

Auth (read by the binary itself):

| Var                | Purpose                          |
| ------------------ | -------------------------------- |
| `ZCP_BEARER_TOKEN` | API token (**required**)         |
| `ZCP_API_URL`      | API base URL override (optional) |

Resource selection — all optional; auto-detected from the API otherwise. Set
these to pin the suite to a known-good region/plan and skip discovery:

```
ZCP_SMOKE_REGION            compute region slug          (default: first active non-Ceph/Dns region)
ZCP_SMOKE_CLOUD_PROVIDER    cloud provider slug          (default: from region)
ZCP_SMOKE_PROJECT           project slug                 (default: first project)
ZCP_SMOKE_TEMPLATE          region-scoped template slug  (default: an Ubuntu image in region)
ZCP_SMOKE_VM_PLAN           VM plan slug                 (default: cheapest active plan)
ZCP_SMOKE_BLOCKSTORAGE_PLAN block storage plan slug
ZCP_SMOKE_IP_PLAN           IP plan slug
ZCP_SMOKE_NETWORK_PLAN      network/internet plan slug   (default: inet-<region-prefix>)
ZCP_SMOKE_STORAGE_CAT       storage category slug        (default: pro-nvme)
ZCP_SMOKE_BILLING_CYCLE     billing cycle slug           (default: hourly)
ZCP_SMOKE_LIFECYCLE=1       same as passing --lifecycle
```

> **Slug gotcha:** the CLI `plan` list views show a human _name_ (`ca2.s`), but
> create endpoints want the _slug_ (`ca2s`), and templates/plans are
> region-scoped (the YUL Ubuntu template is `ubuntu-2404-lts-1`, not
> `ubuntu-2404-lts`). The suite resolves these from the API automatically; the
> env overrides above are the escape hatch when auto-detection picks wrong.

## CI

`.github/workflows/smoke.yml` runs the suite on four triggers, gated on the
`ZCP_SMOKE_BEARER_TOKEN` secret being present (so it no-ops on forks):

| Trigger                             | Scope                                      | Depth                                            |
| ----------------------------------- | ------------------------------------------ | ------------------------------------------------ |
| **tag push** (`[0-9]*`)             | all services                               | **lifecycle** — a new version is tested for real |
| **pull_request** touching a service | only the affected services (`affected.sh`) | read-only (fast, safe, no cost)                  |
| **workflow_dispatch**               | services you pick                          | lifecycle optional (checkbox)                    |
| **schedule** (nightly)              | all services                               | lifecycle — catches live/server-side drift       |

### One-time setup

Add to the repo (Settings → Secrets and variables → Actions):

- **Secret** `ZCP_SMOKE_BEARER_TOKEN` — API key for a dedicated, disposable
  smoke account (ideally with its own project and a spend cap).
- _(optional)_ **Secret** `ZCP_SMOKE_API_URL` — non-prod API base URL.
- _(optional)_ **Variables** `ZCP_SMOKE_REGION`, `ZCP_SMOKE_PROJECT`,
  `ZCP_SMOKE_CLOUD_PROVIDER` to pin the environment.

## Files

```
tests/smoke/
  smoke.sh       entrypoint — arg parsing, orchestration, summary, exit code
  lib.sh         framework — run_case/asserts, resource detection, cleanup stack
  cases.sh       per-service read sweep + create/verify/destroy lifecycle
  affected.sh    maps a code diff → impacted services (used by CI on PRs)
  README.md      this file
```

## Adding a service

1. Add the command name to `ALL_SERVICES` in `cases.sh`.
2. Add its read path under `do_read()`.
3. (Optional) add a `lc_<svc>` function and wire it into `do_lifecycle()`;
   register every created resource with `defer <type> <slug> [extra]` so it is
   torn down. Use an existing `lc_*` as a template.
