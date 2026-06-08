#!/usr/bin/env bash
# affected.sh — print the smoke services impacted by a code change.
#
#   tests/smoke/affected.sh <base-ref>
#
# Maps every changed file between <base-ref> and HEAD to the `zcp` service(s)
# it backs, and prints them comma-separated on one line. A change to shared
# infrastructure (http client, config, root command, version) prints `all`,
# since it can affect every service.
#
# Used by the CI workflow to scope PR smoke runs to just what changed.
#
# shellcheck shell=bash
set -uo pipefail

BASE="${1:-origin/main}"

# Normalise an api package dir or command file basename to a CLI service name.
norm() {
  case "$1" in
    affinitygroup)   echo affinity-group ;;
    billingcycle)    echo billing-cycle ;;
    cloudprovider)   echo cloud-provider ;;
    ipaddress|ip)    echo ip ;;
    objectstorage)   echo object-storage ;;
    sshkey)          echo ssh-key ;;
    storagecategory) echo storage-category ;;
    virtualrouter)   echo vpc ;;       # router endpoints are reached via vpc
    vmbackup)        echo vm-backup ;;
    vmsnapshot)      echo vm-snapshot ;;
    userprofile|profileinfo) echo profile-info ;;
    # core/shared → everything is potentially affected
    apierrors|response|httpclient|config|version|root|auth|client) echo all ;;
    # pass-through (already a valid service name)
    *) echo "$1" ;;
  esac
}

changed="$(git diff --name-only "${BASE}...HEAD" 2>/dev/null || git diff --name-only "${BASE}" 2>/dev/null)"

svcs=""
all=0
while IFS= read -r f; do
  [[ -z "$f" ]] && continue
  case "$f" in
    internal/api/*/*)
      pkg="${f#internal/api/}"; pkg="${pkg%%/*}"
      svc="$(norm "$pkg")" ;;
    internal/commands/*.go)
      svc="$(norm "$(basename "$f" .go)")" ;;
    cmd/zcp/*|internal/httpclient/*|internal/config/*|internal/version/*|go.mod|go.sum)
      svc="all" ;;
    tests/smoke/*)
      svc="all" ;;   # changing the harness itself → run everything
    *)
      continue ;;
  esac
  [[ "$svc" == "all" ]] && { all=1; break; }
  svcs="${svcs}${svc}"$'\n'
done <<<"$changed"

if [[ $all -eq 1 || -z "$svcs" ]]; then
  echo "all"
else
  printf '%s' "$svcs" | sort -u | paste -sd, - | sed 's/,$//'
fi
