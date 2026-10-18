#!/bin/bash
#
# Postinstall for the "Terraform Provider: dmcphearson/iru" Iru custom app.
#
# The app has already unzipped the provider binaries into the machine-wide
# Terraform filesystem mirror:
#
#   /Library/Application Support/io.terraform/plugins/registry.terraform.io/dmcphearson/iru/<ver>/darwin_<arch>/terraform-provider-iru_v<ver>
#
# This script makes those binaries usable by every user's Terraform:
#   1. Strip the Gatekeeper quarantine xattr so the provider can be exec'd.
#   2. Ensure world-readable/executable permissions (mirror is shared).
#   3. Fallback: ad-hoc codesign if quarantine removal alone is insufficient.
#
# Runs as root under the Iru agent.

set -euo pipefail

MIRROR_ROOT="/Library/Application Support/io.terraform/plugins/registry.terraform.io/dmcphearson"

log() { echo "[iru-provider postinstall] $*"; }
fail() { echo "[iru-provider postinstall] ERROR: $*" >&2; exit 1; }

[ -d "${MIRROR_ROOT}" ] || fail "mirror dir not found: ${MIRROR_ROOT} (did the unzip run?)"

log "Stripping Gatekeeper quarantine from ${MIRROR_ROOT}"
# -r recurse, -d delete attr. Ignore "attribute not found" — a clean tree is fine.
xattr -dr com.apple.quarantine "${MIRROR_ROOT}" 2>/dev/null || true

log "Setting world read/execute on the mirror tree"
chmod -R a+rX "${MIRROR_ROOT}"

# Ad-hoc codesign fallback: if a binary is still Gatekeeper-blocked (unsigned +
# quarantine reapplied), an ad-hoc signature ("-") lets it run locally. This is a
# safety net; real Developer ID signing is the escalation if ad-hoc proves
# insufficient in the field.
while IFS= read -r -d '' bin; do
  if ! codesign -v "${bin}" >/dev/null 2>&1; then
    log "Ad-hoc signing ${bin}"
    codesign --force --sign - "${bin}" >/dev/null 2>&1 || log "WARN: codesign failed for ${bin}"
  fi
done < <(find "${MIRROR_ROOT}" -type f -name 'terraform-provider-iru_v*' -print0)

log "Done. Provider available in the machine-wide Terraform mirror."
