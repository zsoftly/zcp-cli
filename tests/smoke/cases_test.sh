#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/cases.sh"

rname() { printf 'owned-k8s'; }
fx_sshkey() { FX_SSHKEY_NAME='fixture-key'; }
det_region() { printf 'yul-1'; }
det_region_id() { printf 'region-id'; }
det_project() { printf 'project'; }
det_cp() { printf 'provider'; }
det_billing_cycle() { printf 'hourly'; }
det_root_disk_size() { printf '100'; }

api_get() {
  case "$1" in
    /kubernetes-clusters/versions)
      printf '%s' '{"data":[{"region_id":"region-id","version":"v1.37.0"}]}'
      ;;
    /plans/service/Kubernetes*)
      printf '%s' '{"data":[{"slug":"worker","attribute":{"package_for":"worker"},"storage_category":{"slug":"pro-nvme"}},{"slug":"control","attribute":{"package_for":"master"},"storage_category":{"slug":"pro-nvme"}}]}'
      ;;
    /plans/service/Block*)
      printf '%s' '{"data":[{"slug":"b2g1"}]}'
      ;;
    /regions)
      printf '%s' '{"data":[{"slug":"yul-1","cloud_provider_setup":{"slug":"setup"}}]}'
      ;;
  esac
}

zcp() {
  if [[ "$1" == kubernetes && "$2" == create ]]; then
    printf '%s' '{"data":{}}'
    return
  fi
  if [[ "$1" == kubernetes && "$2" == list ]]; then
    printf '%s' '[{"name":"existing-cluster","slug":"existing-cluster"},{"name":"owned-k8s","slug":"owned-cluster"}]'
    return
  fi
  return 1
}

lc_kubernetes

[[ ${#CLEANUP_STACK[@]} -eq 1 ]]
[[ "${CLEANUP_STACK[0]}" == 'cancel|owned-cluster|Kubernetes' ]]
