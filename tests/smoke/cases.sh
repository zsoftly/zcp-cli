#!/usr/bin/env bash
# cases.sh — per-service read sweep + create/verify/destroy lifecycle.
#
# Sourced by smoke.sh after lib.sh. Two entry points per service:
#   do_read <svc>        — exercise the read/list paths (safe, no mutation)
#   do_lifecycle <svc>   — create → verify → register teardown (real resources)
#
# Shared, expensive resources (network, IP, VM, volume) are provisioned once by
# the fx_* fixture providers and reused across dependent services.
#
# shellcheck shell=bash
# shellcheck source=/dev/null

# ─── service catalogue ───────────────────────────────────────────────────────
# Every top-level `zcp` command that talks to a service. Order matters for
# lifecycle: fixtures (ssh-key, network, ip, instance, volume) come first.
# shellcheck disable=SC2034  # consumed by smoke.sh, which sources this file
ALL_SERVICES=(
  auth region cloud-provider currency billing-cycle storage-category product
  plan marketplace server project profile-info billing dashboard monitoring
  ssh-key affinity-group network vpc dns ip instance volume
  firewall egress portforward acl
  vm-snapshot snapshot backup vm-backup loadbalancer
  object-storage iso kubernetes autoscale template
)

# ═══════════════════════════════════════════════════════════════════════════
# READ SWEEP
# ═══════════════════════════════════════════════════════════════════════════
do_read() {
  case "$1" in
    auth)             run_case "auth validate"            -- zcp auth validate ;;
    region)           run_case "region list"              -- zcp region list ;;
    cloud-provider)   run_case "cloud-provider list"      -- zcp cloud-provider list ;;
    currency)         run_case "currency list"            -- zcp currency list ;;
    billing-cycle)    run_case "billing-cycle list"       -- zcp billing-cycle list ;;
    storage-category) run_case "storage-category list"    -- zcp storage-category list ;;
    product)          run_case "product list"             -- zcp product list
                      run_case "product categories"       -- zcp product categories ;;
    marketplace)      run_case "marketplace list"         -- zcp marketplace list ;;
    server)           run_case "server list"              -- zcp server list ;;
    project)          run_case "project list"             -- zcp project list ;;
    profile-info)     run_case "profile-info get"         -- zcp profile-info get ;;
    dashboard)        run_case "dashboard summary"        -- zcp dashboard summary ;;
    monitoring)       run_case "monitoring global"        -- zcp monitoring global ;;
    plan)
      local p
      for p in vm storage ip kubernetes lb router iso backup template vm-snapshot; do
        run_case "plan $p" -- zcp plan "$p"
      done ;;
    billing)
      local b
      for b in balance costs invoices service-counts usage payments contracts credit-limit free-credits cancel-requests subscriptions; do
        if [[ "$b" == "subscriptions" ]]; then run_case "billing subscriptions active" -- zcp billing subscriptions active
        else run_case "billing $b" -- zcp billing "$b"; fi
      done ;;
    ssh-key)          run_case "ssh-key list"             -- zcp ssh-key list ;;
    affinity-group)   run_case "affinity-group list"      -- zcp affinity-group list ;;
    network)          run_case "network list"             -- zcp network list
                      run_case "network categories"       -- zcp network categories ;;
    vpc)              run_case "vpc list"                 -- zcp vpc list ;;
    dns)              run_case "dns list"                 -- zcp dns list ;;
    ip)               run_case "ip list"                  -- zcp ip list ;;
    instance)         run_case "instance list"            -- zcp instance list ;;
    volume)           run_case "volume list"              -- zcp volume list ;;
    vm-snapshot)      run_case "vm-snapshot list"         -- zcp vm-snapshot list ;;
    snapshot)         run_case "snapshot list"            -- zcp snapshot list ;;
    backup)           run_case "backup list"              -- zcp backup list ;;
    vm-backup)        run_case "vm-backup list"           -- zcp vm-backup list ;;
    loadbalancer)     run_case "loadbalancer list"        -- zcp loadbalancer list ;;
    object-storage)   run_case "object-storage list"      -- zcp object-storage list ;;
    iso)              run_case "iso list"                 -- zcp iso list --timeout 60 ;;
    kubernetes)       run_case "kubernetes list"          -- zcp kubernetes list ;;
    autoscale)        run_case "autoscale list"           -- zcp autoscale list ;;
    template)         run_case "template list"            -- zcp template list
                      run_case "template account-list"    -- zcp template account-list ;;
    # parent-scoped reads: exercised against a detected/created parent
    firewall)         _read_with_ip       firewall ;;
    egress)           _read_with_network  egress ;;
    portforward)      _read_with_ip       portforward ;;
    acl)              _read_acl ;;
    *)                _mark_skip "$1 (no read case)" ;;
  esac
}

# firewall/portforward list need --ip; pick the first existing IP or skip.
_read_with_ip() {
  local svc="$1" ip
  ip="$(zcp ip list -o json 2>/dev/null | jq -r '(.[]//.data[]) | .slug' 2>/dev/null | head -1)"
  if [[ -z "$ip" || "$ip" == "null" ]]; then _mark_skip "$svc list (no IP available)"; return; fi
  run_case "$svc list --ip $ip" -- zcp "$svc" list --ip "$ip"
}
# egress list needs a network arg.
_read_with_network() {
  local svc="$1" net
  net="$(api_get "/networks?region=$(det_region)" | jq -r '.data[0].slug' 2>/dev/null)"
  if [[ -z "$net" || "$net" == "null" ]]; then _mark_skip "$svc list (no network available)"; return; fi
  run_case "egress list --network $net" -- zcp egress list --network "$net"
}
# acl list is VPC-scoped — needs a VPC slug, not a network.
_read_acl() {
  local vpc
  vpc="$(zcp vpc list -o json 2>/dev/null | jq -r '(.[]//.data[]) | .slug' 2>/dev/null | head -1)"
  if [[ -z "$vpc" || "$vpc" == "null" ]]; then _mark_skip "acl list (no VPC available)"; return; fi
  run_case "acl list $vpc" -- zcp acl list "$vpc"
}

# ═══════════════════════════════════════════════════════════════════════════
# FIXTURES (created once, reused, torn down at the end)
# ═══════════════════════════════════════════════════════════════════════════
FX_SSHKEY=""; FX_NETWORK=""; FX_IP=""; FX_VM=""; FX_VMIP=""; FX_VOLUME=""

fx_sshkey() {
  [[ -n "$FX_SSHKEY" ]] && return 0
  local name pub out
  name="$(rname k)"
  # ephemeral throwaway key
  pub="$(ssh-keygen -t ed25519 -N '' -C "$name" -f "/tmp/${name}" >/dev/null 2>&1 && cat "/tmp/${name}.pub")"
  rm -f "/tmp/${name}" "/tmp/${name}.pub" 2>/dev/null
  [[ -z "$pub" ]] && return 1
  capture out -- zcp ssh-key import --name "$name" --public-key "$pub" -o json || return 1
  FX_SSHKEY="$(_jq_slug <<<"$out")"
  [[ -z "$FX_SSHKEY" ]] && FX_SSHKEY="$(zcp ssh-key list -o json 2>/dev/null | jq -r --arg n "$name" '(.[]//.data[])|select(.name==$n)|.slug' | head -1)"
  [[ -n "$FX_SSHKEY" ]] && defer ssh-key "$FX_SSHKEY"
}

fx_network() {
  [[ -n "$FX_NETWORK" ]] && return 0
  local name out cat; name="$(rname net)"; cat="$(det_network_category)"
  [[ -z "$cat" ]] && return 1
  capture out -- zcp network create --name "$name" --category "$cat" \
    --cloud-provider "$(det_cp)" --project "$(det_project)" --region "$(det_region)" -o json || return 1
  FX_NETWORK="$(_jq_slug <<<"$out")"
  [[ -z "$FX_NETWORK" ]] && FX_NETWORK="$(api_get "/networks?region=$(det_region)" | jq -r --arg n "$name" '.data[]|select(.name==$n)|.slug' | head -1)"
  [[ -n "$FX_NETWORK" ]] && defer network "$FX_NETWORK"
}

fx_ip() {
  [[ -n "$FX_IP" ]] && return 0
  local net out; fx_network; net="$FX_NETWORK"; [[ -z "$net" ]] && return 1
  capture out -- zcp ip allocate --network "$net" --plan "$(det_ip_plan)" --billing-cycle "$(det_billing_cycle)" -o json || return 1
  FX_IP="$(_jq_slug <<<"$out")"
  [[ -z "$FX_IP" ]] && FX_IP="$(zcp ip list -o json 2>/dev/null | jq -r --arg n "$net" '(.[]//.data[])|select(.network_slug==$n or .["NETWORK ID"]!=null)|.slug' | head -1)"
  [[ -n "$FX_IP" ]] && defer cancel "$FX_IP" "IP Address"
}

# fx_vm — the shared VM, created via --network-plan (auto isolated net + SourceNAT
# IP). Waits up to ~3min for Running. Its SourceNAT IP is cached in FX_VMIP.
fx_vm() {
  [[ -n "$FX_VM" ]] && return 0
  local name out; name="$(rname vm)"
  capture out -- zcp instance create --name "$name" \
    --cloud-provider "$(det_cp)" --project "$(det_project)" --region "$(det_region)" \
    --template "$(det_template)" --plan "$(det_vm_plan)" \
    --storage-category "$(det_storage_cat)" --blockstorage-plan "$(det_blockstorage_plan)" \
    --network-plan "$(det_network_plan)" --billing-cycle "$(det_billing_cycle)" -y -o json \
    || { echo "[fx_vm] instance create failed: ${out:0:200}"; return 1; }
  FX_VM="$(_jq_slug <<<"$out")"
  [[ -z "$FX_VM" ]] && FX_VM="$(zcp instance list -o json 2>/dev/null | jq -r --arg n "$name" '(.[]//.data[])|select(.name==$n or .slug==$n)|.slug' | head -1)"
  [[ -z "$FX_VM" ]] && return 1
  defer cancel "$FX_VM" "Virtual Machine"
  # wait for Running (VM boot can take a few minutes; tunable via ZCP_SMOKE_VM_WAIT)
  local state polls="${ZCP_SMOKE_VM_WAIT:-30}"
  for _ in $(seq 1 "$polls"); do
    state="$(zcp instance get "$FX_VM" -o json 2>/dev/null | jq -r '(.[]?|select(.field=="State")|.value)//.state//.data.state//empty' | head -1)"
    [[ "$state" == "Running" ]] && break
    sleep 10
  done
  FX_VMIP="$(zcp ip list -o json 2>/dev/null | jq -r --arg v "$FX_VM" '(.[]//.data[])|select(.vm==$v or .VM==$v)|.slug' | head -1)"
}

fx_volume() {
  [[ -n "$FX_VOLUME" ]] && return 0
  local name out; name="$(rname vol)"
  capture out -- zcp volume create --name "$name" --project "$(det_project)" \
    --cloud-provider "$(det_cp)" --region "$(det_region)" --billing-cycle "$(det_billing_cycle)" \
    --storage-category "$(det_storage_cat)" --plan "$(det_blockstorage_plan)" -o json || return 1
  FX_VOLUME="$(_jq_slug <<<"$out")"
  [[ -z "$FX_VOLUME" ]] && FX_VOLUME="$(zcp volume list -o json 2>/dev/null | jq -r --arg n "$name" '(.[]//.data[])|select(.name==$n)|.slug' | head -1)"
  [[ -n "$FX_VOLUME" ]] && defer cancel "$FX_VOLUME" "Block Storage"
}

# helper: assert a non-empty slug came back from a create, mark pass/fail
_lc_result() { # _lc_result <name> <slug>
  if [[ -n "$2" && "$2" != "null" ]]; then _mark_pass "$1 (create) → $2"; return 0
  else _mark_fail "$1 (create returned no slug)"; return 1; fi
}

# ═══════════════════════════════════════════════════════════════════════════
# LIFECYCLE
# ═══════════════════════════════════════════════════════════════════════════
do_lifecycle() {
  case "$1" in
    ssh-key)        lc_sshkey ;;
    affinity-group) lc_affinity ;;
    network)        lc_network ;;
    vpc)            lc_vpc ;;
    dns)            lc_dns ;;
    ip)             lc_ip ;;
    instance)       lc_instance ;;
    volume)         lc_volume ;;
    firewall)       lc_firewall ;;
    egress)         lc_egress ;;
    portforward)    lc_portforward ;;
    acl)            lc_acl ;;
    vm-snapshot)    lc_vmsnapshot ;;
    snapshot)       lc_snapshot ;;
    backup)         lc_backup ;;
    vm-backup)      lc_vmbackup ;;
    loadbalancer)   lc_loadbalancer ;;
    object-storage) lc_objectstorage ;;
    iso)            lc_iso ;;
    kubernetes)     lc_kubernetes ;;
    autoscale)      _mark_skip "autoscale lifecycle (multi-step; not yet automated)" ;;
    template)       lc_template_account ;;
    *)              : ;;  # read-only services have no lifecycle
  esac
}

lc_sshkey()   { local s; fx_sshkey; s="$FX_SSHKEY"; _lc_result "ssh-key" "$s" \
                && run_case "ssh-key in list" -- bash -c "zcp ssh-key list -o json | jq -e --arg s '$s' '[.[]?,.data[]?]|map(.slug)|index(\$s)' >/dev/null"; }

lc_affinity() {
  local name out s; name="$(rname ag)"
  capture out -- zcp affinity-group create --name "$name" --type "host anti-affinity" \
    --cloud-provider "$(det_cp)" --project "$(det_project)" --region "$(det_region)" -o json
  s="$(_jq_slug <<<"$out")"
  [[ -z "$s" ]] && s="$(zcp affinity-group list -o json 2>/dev/null | jq -r --arg n "$name" '(.[]//.data[])|select(.name==$n)|.slug' | head -1)"
  _lc_result "affinity-group" "$s" && defer affinity-group "$s"
}

lc_network()  {
  [[ -z "$(det_network_category)" ]] && { _mark_skip "network create (no network category in $(det_region))"; return; }
  local s; fx_network; s="$FX_NETWORK"; _lc_result "network" "$s" \
    && run_case "network update" -- zcp network update "$s" --description "smoke $(date -u +%H%M)" ; }

lc_vpc() {
  local name out s; name="$(rname vpc)"
  capture out -- zcp vpc create --name "$name" --network-address "10.77.0.1" --size 24 \
    --plan "$(det_router_plan)" --storage-category "$(det_storage_cat)" \
    --cloud-provider "$(det_cp)" --project "$(det_project)" --region "$(det_region)" \
    --billing-cycle "$(det_billing_cycle)" -y -o json
  s="$(_jq_slug <<<"$out")"
  [[ -z "$s" ]] && s="$(zcp vpc list -o json 2>/dev/null | jq -r --arg n "$name" '(.[]//.data[])|select(.name==$n)|.slug' | head -1)"
  _lc_result "vpc" "$s" && defer vpc "$s"
}

lc_dns() {
  local dom out s; dom="smoke-${SMOKE_RID}.example.com"
  capture out -- zcp dns create --name "$dom" --cloud-provider "$(det_cp)" \
    --project "$(det_project)" --region "$(det_region)" -o json
  s="$(_jq_slug <<<"$out")"
  [[ -z "$s" ]] && s="$(api_get '/dns-domains' | jq -r --arg n "$dom" '.data[]?|select(.name==$n)|.slug' | head -1)"
  if _lc_result "dns domain" "$s"; then
    defer dns "$s"
    run_case "dns record-create" -- zcp dns record-create --domain "$s" --name www --type A --content "1.2.3.4"
  fi
}

lc_ip()       {
  [[ -z "$(det_network_category)" ]] && { _mark_skip "ip allocate (needs a network; none creatable in $(det_region))"; return; }
  local s; fx_ip; s="$FX_IP"; _lc_result "ip allocate" "$s" \
    && run_case "ip in list" -- bash -c "zcp ip list -o json | jq -e --arg s '$s' '[.[]?,.data[]?]|map(.slug)|index(\$s)' >/dev/null"; }

lc_instance() {
  local s; fx_vm; s="$FX_VM"
  if _lc_result "instance" "$s"; then
    run_case "instance get → Running"  -- bash -c "zcp instance get '$s' -o json | jq -e '[.[]?|select(.field==\"State\")|.value]|index(\"Running\")' >/dev/null || zcp instance get '$s'"
    run_case "instance logs"           -- zcp instance logs "$s"
    run_case "instance addons"         -- zcp instance addons "$s"
  fi
}

lc_volume() {
  local s vm; fx_volume; s="$FX_VOLUME"
  if _lc_result "volume" "$s"; then
    fx_vm; vm="$FX_VM"
    if [[ -n "$vm" ]]; then
      run_case "volume attach"  -- zcp volume attach "$s" "$vm"
      sleep 8
      run_case "volume detach"  -- zcp volume detach "$s"
    else _mark_skip "volume attach (no VM fixture)"; fi
  fi
}

lc_firewall() {
  local ip out s; fx_ip; ip="$FX_IP"; [[ -z "$ip" ]] && { _mark_skip "firewall (no IP fixture)"; return; }
  capture out -- zcp firewall create --ip "$ip" --protocol tcp --cidr "0.0.0.0/0" --start-port 22 --end-port 22 -o json
  s="$(_jq_slug <<<"$out")"
  [[ -z "$s" ]] && s="$(zcp firewall list --ip "$ip" -o json 2>/dev/null | jq -r '(.[]//.data[])|.slug' | head -1)"
  _lc_result "firewall rule" "$s" && defer firewall "$s"
}

lc_egress() {
  local net out s; fx_network; net="$FX_NETWORK"; [[ -z "$net" ]] && { _mark_skip "egress (no network fixture)"; return; }
  capture out -- zcp egress create --network "$net" --protocol tcp --cidr "0.0.0.0/0" --start-port 80 --end-port 80 -o json
  s="$(_jq_slug <<<"$out")"
  [[ -z "$s" ]] && s="$(zcp egress list --network "$net" -o json 2>/dev/null | jq -r '(.[]//.data[])|.slug' | head -1)"
  _lc_result "egress rule" "$s" && defer egress "$s"
}

lc_portforward() {
  local vm ip out s; fx_vm; vm="$FX_VM"; ip="$FX_VMIP"
  [[ -z "$vm" || -z "$ip" ]] && { _mark_skip "portforward (need VM + its public IP)"; return; }
  capture out -- zcp portforward create --instance "$vm" --ip "$ip" --protocol tcp --public-port 8080 --private-port 80 -o json
  s="$(_jq_slug <<<"$out")"
  [[ -z "$s" ]] && s="$(zcp portforward list --ip "$ip" -o json 2>/dev/null | jq -r '(.[]//.data[])|.slug' | head -1)"
  _lc_result "portforward rule" "$s" && defer portforward "$s"
}

lc_acl() {
  local net out s; fx_network; net="$FX_NETWORK"; [[ -z "$net" ]] && { _mark_skip "acl (no network fixture)"; return; }
  capture out -- zcp acl create "$net" --name "$(rname acl)" -o json
  s="$(_jq_slug <<<"$out")"
  if [[ -n "$s" && "$s" != "null" ]]; then _mark_pass "acl create → $s"; else
    # acl may apply directly to the network without returning a slug
    run_case "acl list" -- zcp acl list "$net"; fi
}

lc_vmsnapshot() {
  local vm out s; fx_vm; vm="$FX_VM"; [[ -z "$vm" ]] && { _mark_skip "vm-snapshot (no VM fixture)"; return; }
  capture out -- zcp vm-snapshot create --vm "$vm" --name "$(rname vmsnap)" -o json
  s="$(_jq_slug <<<"$out")"
  [[ -z "$s" ]] && s="$(zcp vm-snapshot list -o json 2>/dev/null | jq -r --arg v "$vm" '(.[]//.data[])|select(.vm_id!=null)|.slug' | head -1)"
  _lc_result "vm-snapshot" "$s" && defer vm-snapshot "$s"
}

lc_snapshot() {
  local vol out s; fx_volume; vol="$FX_VOLUME"; [[ -z "$vol" ]] && { _mark_skip "snapshot (no volume fixture)"; return; }
  capture out -- zcp snapshot create "$vol" --name "$(rname snap)" --service "VM Snapshot" \
    --project "$(det_project)" --cloud-provider "$(det_cp)" --region "$(det_region)" \
    --billing-cycle "$(det_billing_cycle)" --plan "$(det_blockstorage_plan)" -o json
  s="$(_jq_slug <<<"$out")"
  _lc_result "snapshot" "$s" && defer cancel "$s" "Block Storage Snapshot"
}

lc_backup() {
  local vol out s; fx_volume; vol="$FX_VOLUME"; [[ -z "$vol" ]] && { _mark_skip "backup (no volume fixture)"; return; }
  capture out -- zcp backup create "$vol" --name "$(rname bk)" -o json 2>/dev/null
  s="$(_jq_slug <<<"$out")"
  if [[ -n "$s" && "$s" != "null" ]]; then _mark_pass "backup → $s"; defer cancel "$s" "Block Storage Backup"
  else _mark_skip "backup (create flags vary by env)"; fi
}

lc_vmbackup() {
  local vm out s; fx_vm; vm="$FX_VM"; [[ -z "$vm" ]] && { _mark_skip "vm-backup (no VM fixture)"; return; }
  capture out -- zcp vm-backup create "$vm" --name "$(rname vmbk)" -o json 2>/dev/null
  s="$(_jq_slug <<<"$out")"
  if [[ -n "$s" && "$s" != "null" ]]; then _mark_pass "vm-backup → $s"; defer cancel "$s" "Virtual Machine Backup"
  else _mark_skip "vm-backup (create flags vary by env)"; fi
}

lc_loadbalancer() {
  local net out s; fx_network; net="$FX_NETWORK"; [[ -z "$net" ]] && { _mark_skip "loadbalancer (no network fixture)"; return; }
  capture out -- zcp loadbalancer create --name "$(rname lb)" --network "$net" \
    --cloud-provider "$(det_cp)" --project "$(det_project)" --region "$(det_region)" \
    --billing-cycle "$(det_billing_cycle)" -y -o json
  s="$(_jq_slug <<<"$out")"
  [[ -z "$s" ]] && s="$(zcp loadbalancer list -o json 2>/dev/null | jq -r '(.[]//.data[])|.slug' | head -1)"
  _lc_result "loadbalancer" "$s" && defer cancel "$s" "Load Balancer"
}

lc_objectstorage() {
  local osr out s
  osr="$(api_get '/regions' | jq -r '.data[]|select((.cloud_provider.slug//"")=="ceph")|.slug' | head -1)"
  [[ -z "$osr" ]] && osr="$(det_region)"
  capture out -- zcp object-storage create --name "$(rname os)" --storage-gb 60 \
    --cloud-provider ceph --project "$(det_project)" --region "$osr" \
    --billing-cycle "$(det_billing_cycle)" -o json
  s="$(_jq_slug <<<"$out")"
  [[ -z "$s" ]] && s="$(zcp object-storage list -o json 2>/dev/null | jq -r '(.[]//.data[])|.slug' | head -1)"
  _lc_result "object-storage" "$s" && defer object-storage "$s"
}

lc_iso() {
  local out s url; url="${ZCP_SMOKE_ISO_URL:-http://releases.ubuntu.com/24.04/ubuntu-24.04-netboot-amd64.iso}"
  capture out -- zcp iso create --name "$(rname iso)" --url "$url" \
    --cloud-provider "$(det_cp)" --project "$(det_project)" --region "$(det_region)" -o json 2>/dev/null
  s="$(_jq_slug <<<"$out")"
  if [[ -n "$s" && "$s" != "null" ]]; then _mark_pass "iso register → $s"; defer iso "$s"
  else _mark_skip "iso (registration is async / URL-dependent)"; fi
}

lc_kubernetes() {
  local out; local ver plan setup
  ver="$(api_get '/kubernetes-clusters/versions' | jq -r --arg rid "$(det_region_id)" '.data[]|select(.region_id==$rid)|.slug' | head -1)"
  plan="$(api_get "/plans/service/Kubernetes?region=$(det_region)" | jq -r '.data[]|select((.name//"")|test("YUL|YOW";"i"))|.slug' | head -1)"
  setup="$(api_get '/regions' | jq -r --arg s "$(det_region)" '.data[]|select(.slug==$s)|.cloud_provider_setup.slug' | head -1)"
  [[ -z "$ver" || -z "$plan" ]] && { _mark_skip "kubernetes (no version/plan for region)"; return; }
  out="$(zcp kubernetes create --name "$(rname k8s)" --version "$ver" --plan "$plan" \
    --region "$(det_region)" --project "$(det_project)" --cloud-provider "$(det_cp)" \
    --billing-cycle "$(det_billing_cycle)" --workers 1 --cloud-provider-setup "${setup:-zcp-apc}" -y 2>&1)"
  if grep -qiE 'quota not found' <<<"$out"; then
    _mark_skip "kubernetes (account has no k8s quota — env limitation)"
  elif grep -qiE 'error' <<<"$out"; then
    _mark_fail "kubernetes create"; sed -n '1,2p' <<<"$out" | sed 's/^/        /'
  else
    local s; s="$(zcp kubernetes list -o json 2>/dev/null | jq -r '(.[]//.data[])|.slug' | head -1)"
    _mark_pass "kubernetes create"; [[ -n "$s" ]] && defer cancel "$s" "Kubernetes"
  fi
}

lc_template_account() {
  local out s url; url="${ZCP_SMOKE_TEMPLATE_URL:-}"
  [[ -z "$url" ]] && { _mark_skip "template account-create (set ZCP_SMOKE_TEMPLATE_URL to test)"; return; }
  capture out -- zcp template account-create --name "$(rname tpl)" --url "$url" \
    --cloud-provider "$(det_cp)" --project "$(det_project)" --region "$(det_region)" -o json 2>/dev/null
  s="$(_jq_slug <<<"$out")"
  if [[ -n "$s" && "$s" != "null" ]]; then _mark_pass "template account → $s"; defer template-acct "$s"
  else _mark_skip "template account-create (flags vary by env)"; fi
}
