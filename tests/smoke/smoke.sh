#!/usr/bin/env bash
# smoke.sh — live smoke tests for the ZCP CLI.
#
# Runs the REAL `zcp` binary against a REAL ZCP API so that any release surfaces
# a service whose read or create/destroy path has broken. It is the safety net
# behind "whenever a new version ships, affected services are tested for real".
#
# USAGE
#   tests/smoke/smoke.sh [options]
#
# OPTIONS
#   --only a,b,c     Test only these services (default: all)
#   --all            Test every service (default)
#   --lifecycle      Also run create → verify → destroy (real, billable resources)
#   --read-only      Force read-only even if --lifecycle env is set
#   --list           Print the service catalogue and exit
#   --bin PATH       Path to the zcp binary (default: $ZCP_BIN or `zcp` on PATH)
#   -h, --help       This help
#
# AUTH (read by the binary itself)
#   ZCP_BEARER_TOKEN   API token              (required)
#   ZCP_API_URL        API base URL override  (optional)
#
# TUNING (all optional — auto-detected otherwise)
#   ZCP_SMOKE_REGION / _CLOUD_PROVIDER / _PROJECT / _TEMPLATE / _VM_PLAN /
#   _BLOCKSTORAGE_PLAN / _IP_PLAN / _NETWORK_PLAN / _STORAGE_CAT / _BILLING_CYCLE
#
# EXIT CODES
#   0  all selected cases passed   1  one or more failed   2  setup error
#
# shellcheck shell=bash
# shellcheck source=/dev/null

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${HERE}/lib.sh"
source "${HERE}/cases.sh"

# ─── args ────────────────────────────────────────────────────────────────────
MODE_LIFECYCLE=0
ONLY=""
print_usage() { sed -n '2,40p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'; }

while [[ $# -gt 0 ]]; do
  case "$1" in
    --only)       ONLY="$2"; shift 2 ;;
    --only=*)     ONLY="${1#*=}"; shift ;;
    --all)        ONLY=""; shift ;;
    --lifecycle)  MODE_LIFECYCLE=1; shift ;;
    --read-only)  MODE_LIFECYCLE=0; shift ;;
    --bin)        ZCP_BIN="$2"; shift 2 ;;
    --bin=*)      ZCP_BIN="${1#*=}"; shift ;;
    --list)       printf '%s\n' "${ALL_SERVICES[@]}"; exit 0 ;;
    -h|--help)    print_usage; exit 0 ;;
    *) say "${C_RED}unknown option: $1${C_RST}"; print_usage; exit 2 ;;
  esac
done

# env can also flip lifecycle on (used by CI dispatch / nightly)
[[ "${ZCP_SMOKE_LIFECYCLE:-}" == "1" ]] && MODE_LIFECYCLE=1

# ─── preflight ───────────────────────────────────────────────────────────────
require_jq
require_curl
if ! command -v "$ZCP_BIN" >/dev/null 2>&1 && [[ ! -x "$ZCP_BIN" ]]; then
  say "${C_RED}zcp binary not found: ${ZCP_BIN}${C_RST}  (build with: make build; use --bin ./bin/zcp)"; exit 2
fi
if [[ -z "${ZCP_BEARER_TOKEN:-}" ]]; then
  say "${C_RED}ZCP_BEARER_TOKEN is not set${C_RST}"; exit 2
fi

# selection
if [[ -n "$ONLY" ]]; then
  IFS=',' read -r -a SELECTED <<<"$ONLY"
else
  SELECTED=("${ALL_SERVICES[@]}")
fi

smoke_init
trap 'run_cleanup' EXIT INT TERM

# ─── banner ──────────────────────────────────────────────────────────────────
say ""
say "${C_BLD}ZCP CLI live smoke suite${C_RST}"
say "  binary    : $("$ZCP_BIN" version 2>/dev/null | head -1)"
say "  api       : $(api_base)"
say "  mode      : $([[ $MODE_LIFECYCLE -eq 1 ]] && echo 'read + LIFECYCLE (creates real resources)' || echo 'read-only')"
say "  services  : ${#SELECTED[@]} ($([[ -n "$ONLY" ]] && echo "$ONLY" || echo all))"
say "  region    : $(det_region)  cp=$(det_cp)  project=$(det_project)"
say "  run-id    : ${SMOKE_RID}"

# ─── run ─────────────────────────────────────────────────────────────────────
for svc in "${SELECTED[@]}"; do
  section "$svc"
  do_read "$svc"
  if [[ $MODE_LIFECYCLE -eq 1 ]]; then
    do_lifecycle "$svc"
  fi
done

# ─── summary ─────────────────────────────────────────────────────────────────
say ""
hr
say "${C_BLD}Summary${C_RST}  ${C_GRN}pass=${PASS_N}${C_RST}  ${C_RED}fail=${FAIL_N}${C_RST}  ${C_YEL}skip=${SKIP_N}${C_RST}"
if [[ $FAIL_N -gt 0 ]]; then
  say "${C_RED}Failures:${C_RST}"
  printf '  - %s\n' "${FAILED_CASES[@]}"
fi
hr

# cleanup runs via the EXIT trap
[[ $FAIL_N -eq 0 ]] && exit 0 || exit 1
