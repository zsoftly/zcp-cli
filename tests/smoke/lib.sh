#!/usr/bin/env bash
# lib.sh — shared framework for the ZCP CLI live smoke suite.
#
# This library drives the REAL `zcp` binary against a REAL ZCP API. It provides:
#   - result tracking (pass / fail / skip) and a final summary + exit code
#   - run_case / capture helpers that invoke the binary and inspect rc + output
#   - resource auto-detection (region, plans, slugs the CLI list views hide)
#   - a cleanup stack so every created resource is torn down in reverse order
#
# It is sourced by smoke.sh — do not run it directly.
#
# shellcheck shell=bash

# ─── strictness ──────────────────────────────────────────────────────────────
set -uo pipefail   # NOT -e: we inspect return codes ourselves.

# ─── colour (auto-off when not a tty or NO_COLOR set) ────────────────────────
if [[ -t 1 && -z "${NO_COLOR:-}" ]]; then
  C_RED=$'\033[31m'; C_GRN=$'\033[32m'; C_YEL=$'\033[33m'
  C_BLU=$'\033[34m'; C_DIM=$'\033[2m'; C_BLD=$'\033[1m'; C_RST=$'\033[0m'
else
  C_RED=""; C_GRN=""; C_YEL=""; C_BLU=""; C_DIM=""; C_BLD=""; C_RST=""
fi

# ─── globals ─────────────────────────────────────────────────────────────────
ZCP_BIN="${ZCP_BIN:-zcp}"                 # path to the binary under test
PASS_N=0; FAIL_N=0; SKIP_N=0
declare -a FAILED_CASES=()
declare -a SKIPPED_CASES=()
declare -a CLEANUP_STACK=()               # entries: "type|slug|extra"
SMOKE_RID=""                              # unique run id, set by smoke_init

# ─── logging ─────────────────────────────────────────────────────────────────
say()     { printf '%s\n' "$*"; }
hr()      { printf '%s\n' "${C_DIM}────────────────────────────────────────────────────────────${C_RST}"; }
section() { printf '\n%s\n' "${C_BLD}${C_BLU}### %s${C_RST}" >/dev/null; printf '\n%s### %s%s\n' "${C_BLD}${C_BLU}" "$*" "${C_RST}"; }
info()    { printf '%s  %s%s\n' "${C_DIM}" "$*" "${C_RST}"; }

_mark_pass() { PASS_N=$((PASS_N+1)); printf '  %sPASS%s %s\n' "${C_GRN}" "${C_RST}" "$1"; }
_mark_fail() { FAIL_N=$((FAIL_N+1)); FAILED_CASES+=("$1"); printf '  %sFAIL%s %s\n' "${C_RED}" "${C_RST}" "$1"; }
_mark_skip() { SKIP_N=$((SKIP_N+1)); SKIPPED_CASES+=("$1"); printf '  %sSKIP%s %s%s%s\n' "${C_YEL}" "${C_RST}" "$1" " ${C_DIM}" "${2:-}${C_RST}"; }

# ─── binary wrapper ──────────────────────────────────────────────────────────
# zcp <args...> — invoke the binary under test. Output goes wherever the caller
# redirects it.
zcp() { "$ZCP_BIN" "$@"; }

# _looks_like_decode_bug <text> — true if output carries a JSON-decode / panic
# signature. These are the class of breakage the read sweep exists to catch.
_looks_like_decode_bug() {
  grep -qiE 'cannot unmarshal|decoding response|json: |runtime error|invalid memory|nil pointer' <<<"$1"
}

# run_case <name> -- <cmd...>
# Runs a command, captures combined output, and records pass/fail.
#   pass  → rc == 0
#   fail  → rc != 0  (decode-bug signatures are flagged explicitly)
# Returns the command's rc so callers can branch.
run_case() {
  local name="$1"; shift
  [[ "${1:-}" == "--" ]] && shift
  local out rc
  out="$("$@" 2>&1)"; rc=$?
  if [[ $rc -eq 0 ]]; then
    _mark_pass "$name"
  else
    if _looks_like_decode_bug "$out"; then
      _mark_fail "$name ${C_RED}[decode bug]${C_RST}"
    else
      _mark_fail "$name"
    fi
    # Indented first 3 lines of output for triage.
    sed -n '1,3p' <<<"$out" | sed 's/^/        /'
  fi
  return $rc
}

# capture <var> -- <cmd...>
# Runs a command, stores stdout in <var>. Returns rc. Used to grab slugs/ids.
capture() {
  local __var="$1"; shift
  [[ "${1:-}" == "--" ]] && shift
  local __out __rc __tmp
  __tmp="$(mktemp)"
  __out="$("$@" 2>"$__tmp")"; __rc=$?
  [[ -s "$__tmp" ]] && cat "$__tmp" >&2
  rm -f "$__tmp"
  printf -v "$__var" '%s' "$__out"
  return $__rc
}

# ─── cleanup stack ───────────────────────────────────────────────────────────
# defer <type> <slug> [extra] — register a resource for teardown. LIFO order.
defer() {
  local type="$1" slug="$2" extra="${3:-}"
  [[ -z "$slug" ]] && return 0
  CLEANUP_STACK+=("${type}|${slug}|${extra}")
}

# destroy_one <type> <slug> <extra> — best-effort teardown of a single resource.
# Many ZCP services have no native delete verb; those route through
# `billing cancel-service`. Failures are logged but never abort cleanup.
destroy_one() {
  local type="$1" slug="$2" extra="${3:-}"
  info "teardown ${type} ${slug}"
  case "$type" in
    # These were torn down via raw API calls to work around a `-y` shorthand
    # panic in their `delete` subcommands. That panic is fixed, and the hand-
    # written API paths had drifted from the real ones (dns and ssh-key were
    # wrong, leaking resources), so use the CLI, which knows the correct
    # endpoints and identifiers.
    ssh-key)        zcp ssh-key delete "$slug" -y          >/dev/null 2>&1 ;;
    affinity-group) zcp affinity-group delete "$slug" -y   >/dev/null 2>&1 ;;
    role)           zcp role delete "$slug" -y             >/dev/null 2>&1 ;;
    sub-user)       zcp sub-user delete "$slug" -y         >/dev/null 2>&1 ;;
    dns)            zcp dns delete "$slug" -y               >/dev/null 2>&1 ;;
    iso)            zcp iso delete "$slug" -y               >/dev/null 2>&1 ;;
    network)        zcp network delete "$slug" -y          >/dev/null 2>&1 ;;
    vpc)            zcp vpc delete "$slug" -y               >/dev/null 2>&1 ;;
    dns-record)     zcp dns record-delete "$slug" --domain "$extra" -y >/dev/null 2>&1 ;;
    firewall)       zcp firewall delete "$slug" -y         >/dev/null 2>&1 ;;
    egress)         zcp egress delete "$slug" --network "$extra" -y >/dev/null 2>&1 ;;
    portforward)    zcp portforward delete "$slug" -y      >/dev/null 2>&1 ;;
    loadbalancer)
      # `loadbalancer delete` routes through the service-cancel workflow; --release-ip also
      # frees the LB's dedicated STATIC IP (a network source-NAT IP is auto-skipped). Retry
      # once; deletion is async.
      zcp loadbalancer delete "$slug" --release-ip -y >/dev/null 2>&1 || true
      sleep 5
      zcp loadbalancer delete "$slug" --release-ip -y >/dev/null 2>&1 || true
      ;;
    # `instance delete` routes through the service-cancel workflow (releases the
    # VM's auto-assigned public IP); it supersedes the raw `cancel` path for VMs.
    vm)             zcp instance delete "$slug" -y         >/dev/null 2>&1 ;;
    vm-snapshot)    zcp vm-snapshot delete "$slug" -y      >/dev/null 2>&1 ;;
    object-storage) zcp object-storage delete "$slug" -y   >/dev/null 2>&1 ;;
    template-acct)  zcp template account-delete "$slug" -y >/dev/null 2>&1 ;;
    # cancel-service teardown: extra holds the billing "service" label.
    cancel)         zcp billing cancel-service "$slug" --service "$extra" --type Immediate --delete-public-ip -y >/dev/null 2>&1 ;;
    *)              info "  (no teardown handler for type '$type')"; return 0 ;;
  esac
}

# run_cleanup — drain the cleanup stack in reverse (LIFO) order.
run_cleanup() {
  [[ ${#CLEANUP_STACK[@]} -eq 0 ]] && return 0
  section "Teardown (${#CLEANUP_STACK[@]} resource(s))"
  local i entry type slug extra
  for (( i=${#CLEANUP_STACK[@]}-1; i>=0; i-- )); do
    entry="${CLEANUP_STACK[$i]}"
    type="${entry%%|*}"; entry="${entry#*|}"
    slug="${entry%%|*}";  extra="${entry#*|}"
    destroy_one "$type" "$slug" "$extra"
  done
  CLEANUP_STACK=()
}

# ─── raw API helper (slug discovery only) ────────────────────────────────────
# The CLI's plan/template list views display human names, not the slugs the
# create endpoints require, and templates/plans are region-scoped. We resolve
# those via a thin read-only API call. Everything actually *under test* still
# goes through the binary.
api_base() {
  if [[ -n "${ZCP_API_URL:-}" ]]; then printf '%s' "${ZCP_API_URL%/}"; return; fi
  printf '%s' "https://api.zcp.zsoftly.ca/api"
}
api_get() {  # api_get <path-with-leading-slash>
  curl -fsS -H "Authorization: Bearer ${ZCP_BEARER_TOKEN}" "$(api_base)$1" 2>/dev/null
}
# api_delete <path...> — teardown fallback for resources whose CLI `delete`
# subcommand is currently broken (the `-y` shorthand collision panic). Tries
# each candidate path, succeeds on the first 2xx.
api_delete() {
  local p code
  for p in "$@"; do
    code="$(curl -s -o /dev/null -w '%{http_code}' -X DELETE -H "Authorization: Bearer ${ZCP_BEARER_TOKEN}" "$(api_base)$p")"
    [[ "$code" =~ ^2 ]] && return 0
  done
  return 1
}

# ─── detection (env override wins; else auto-detect) ─────────────────────────
# Each detector echoes a value and caches it in a global so repeat calls are free.
_DET_REGION=""; _DET_CP=""; _DET_PROJECT=""; _DET_REGION_ID=""

det_region() {
  if [[ -n "$_DET_REGION" ]]; then printf '%s' "$_DET_REGION"; return; fi
  if [[ -n "${ZCP_SMOKE_REGION:-}" ]]; then _DET_REGION="$ZCP_SMOKE_REGION"
  else
    # first active region whose provider is a real compute provider (not Ceph/Dns)
    _DET_REGION="$(api_get "/regions" | jq -r '.data[] | select(.status==true) | select((.cloud_provider.slug//"")|test("ceph|dns")|not) | .slug' | head -1)"
  fi
  printf '%s' "$_DET_REGION"
}
det_region_id() {
  if [[ -n "$_DET_REGION_ID" ]]; then printf '%s' "$_DET_REGION_ID"; return; fi
  local r; r="$(det_region)"
  _DET_REGION_ID="$(api_get "/regions" | jq -r --arg s "$r" '.data[] | select(.slug==$s) | .id' | head -1)"
  printf '%s' "$_DET_REGION_ID"
}
det_cp() {
  if [[ -n "$_DET_CP" ]]; then printf '%s' "$_DET_CP"; return; fi
  if [[ -n "${ZCP_SMOKE_CLOUD_PROVIDER:-}" ]]; then _DET_CP="$ZCP_SMOKE_CLOUD_PROVIDER"
  else
    local r; r="$(det_region)"
    _DET_CP="$(api_get "/regions" | jq -r --arg s "$r" '.data[] | select(.slug==$s) | .cloud_provider.slug' | head -1)"
  fi
  printf '%s' "$_DET_CP"
}
det_project() {
  if [[ -n "$_DET_PROJECT" ]]; then printf '%s' "$_DET_PROJECT"; return; fi
  if [[ -n "${ZCP_SMOKE_PROJECT:-}" ]]; then _DET_PROJECT="$ZCP_SMOKE_PROJECT"
  else _DET_PROJECT="$(zcp project list -o json 2>/dev/null | _jq_first_slug)"; fi
  printf '%s' "$_DET_PROJECT"
}

# det_template — region-scoped template slug (defaults to an Ubuntu image in the
# target region). ZCP_SMOKE_TEMPLATE overrides.
det_template() {
  if [[ -n "${ZCP_SMOKE_TEMPLATE:-}" ]]; then printf '%s' "$ZCP_SMOKE_TEMPLATE"; return; fi
  local rid; rid="$(det_region_id)"
  api_get "/templates?region=$(det_region)" \
    | jq -r --arg rid "$rid" '.data[] | select(.region_id==$rid) | select(.slug|test("ubuntu";"i")) | .slug' | head -1
}

# det_vm_plan — cheapest active VM plan slug (the slug differs from the display
# name the CLI shows). ZCP_SMOKE_VM_PLAN overrides.
det_vm_plan() {
  if [[ -n "${ZCP_SMOKE_VM_PLAN:-}" ]]; then printf '%s' "$ZCP_SMOKE_VM_PLAN"; return; fi
  api_get "/plans/service/Virtual%20Machine?region=$(det_region)" \
    | jq -r '.data | sort_by((.monthly|tonumber? // 1e9))[0].slug'
}
det_blockstorage_plan() {
  if [[ -n "${ZCP_SMOKE_BLOCKSTORAGE_PLAN:-}" ]]; then printf '%s' "$ZCP_SMOKE_BLOCKSTORAGE_PLAN"; return; fi
  api_get "/plans/service/Block%20Storage?region=$(det_region)" | jq -r '.data[0].slug // .data[0].name' | head -1
}
det_router_plan() {
  api_get "/plans/service/Virtual%20Router?region=$(det_region)" | jq -r '.data[0].slug // .data[0].name' | head -1
}
det_ip_plan() {
  if [[ -n "${ZCP_SMOKE_IP_PLAN:-}" ]]; then printf '%s' "$ZCP_SMOKE_IP_PLAN"; return; fi
  api_get "/plans/service/IP%20Address?region=$(det_region)" | jq -r '.data[0].slug // .data[0].name' | head -1
}
det_storage_cat() { printf '%s' "${ZCP_SMOKE_STORAGE_CAT:-pro-nvme}"; }
det_billing_cycle() { printf '%s' "${ZCP_SMOKE_BILLING_CYCLE:-hourly}"; }

# det_network_category — slug for the legacy `network create --category` flag
# (the live API returns no categories; network creation uses det_network_plan).
det_network_category() {
  zcp network categories -o json 2>/dev/null | _jq_first_slug
}
# det_network_plan — the internet/network plan slug for `instance create
# --network-plan` (e.g. inet-yul). Defaults to inet-<region-prefix>.
det_network_plan() {
  if [[ -n "${ZCP_SMOKE_NETWORK_PLAN:-}" ]]; then printf '%s' "$ZCP_SMOKE_NETWORK_PLAN"; return; fi
  local r; r="$(det_region)"; printf 'inet-%s' "${r%%-*}"
}

# ─── JSON shape helpers ──────────────────────────────────────────────────────
# CLI `-o json` list output is a top-level array; create output is an object;
# raw API output is wrapped in {data: ...}. These read stdin and tolerate all
# three shapes, never erroring on the wrong one.
_jq_first_slug() { jq -r 'if type=="array" then (.[0].slug? // empty)
                          else (.slug? // .data?.slug? // (.data? | if type=="array" then .[0].slug? else empty end) // empty) end' 2>/dev/null; }
# _jq_slug — slug from a create response (object, {data:{slug}}, array[0], or
# the CLI's FIELD/VALUE table rendered as JSON).
_jq_slug() { jq -r '(.slug? // .data?.slug? // (if type=="array" then (.[0].slug? // ([.[]?|select(.field?=="Slug")|.value][0])) else empty end)) // empty' 2>/dev/null; }

# ─── misc ────────────────────────────────────────────────────────────────────
require_jq() { command -v jq >/dev/null 2>&1 || { say "${C_RED}jq is required for the smoke suite${C_RST}"; exit 2; }; }
require_curl() { command -v curl >/dev/null 2>&1 || { say "${C_RED}curl is required for slug detection${C_RST}"; exit 2; }; }

# smoke_init <run-id-prefix> — set up the unique run id used for resource names.
smoke_init() {
  SMOKE_RID="sm$(date +%s | tail -c 6)${RANDOM:0:2}"
}
rname() { printf '%s-%s' "${1:-res}" "$SMOKE_RID"; }
