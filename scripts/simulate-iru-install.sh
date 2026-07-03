#!/usr/bin/env bash
#
# simulate-iru-install.sh — reproduce, locally, exactly what the Iru custom app
# does when a dev installs "Terraform Provider: dmcphearson/iru" from Self Service:
#
#   1. run preinstall.sh (as root)
#   2. unzip the mirror zip into the MACHINE-WIDE Terraform filesystem mirror
#      /Library/Application Support/io.terraform/plugins  (Iru's unzip_location)
#   3. run postinstall.sh (as root) — quarantine strip, perms, codesign fallback
#
# To make the Gatekeeper test real, we set com.apple.quarantine on the binaries
# BEFORE postinstall so we can prove the strip actually clears it (a locally built
# zip has no quarantine; an MDM-delivered one would).
#
# Run with sudo (writes /Library):  sudo scripts/simulate-iru-install.sh
set -euo pipefail

REPO="/Users/davidmcphearson/dev/gh-dmcphearson/terraform-provider-iru"
SCRIPTS="/Users/davidmcphearson/dev/github/fluency-iru-config/main/component/apps/scripts"
ZIP="${REPO}/dist/iru-provider-0.1.0-mirror.zip"
MIRROR="/Library/Application Support/io.terraform/plugins"
TREE="${MIRROR}/registry.terraform.io/dmcphearson"

hr() { echo "----------------------------------------------------------------"; }

[ "$(id -u)" -eq 0 ] || { echo "must run as root (sudo)"; exit 1; }
[ -f "$ZIP" ] || { echo "zip not found: $ZIP (run make mirror-zip)"; exit 1; }

hr; echo "STEP 1: preinstall.sh (as root)"; hr
bash "${SCRIPTS}/preinstall.sh"

hr; echo "STEP 2: unzip into machine-wide mirror ${MIRROR}"; hr
mkdir -p "$MIRROR"
unzip -o -q "$ZIP" -d "$MIRROR"
# Simulate MDM-delivered quarantine so the strip in postinstall is a real test.
find "$TREE" -type f -name 'terraform-provider-iru_v*' -exec \
  xattr -w com.apple.quarantine "0081;00000000;Simulated;" {} \;
echo "seeded com.apple.quarantine on the provider binaries:"
find "$TREE" -type f -name 'terraform-provider-iru_v*' -exec sh -c 'echo "  $1: $(xattr -p com.apple.quarantine "$1" 2>/dev/null || echo none)"' _ {} \;

hr; echo "STEP 3: postinstall.sh (as root)"; hr
bash "${SCRIPTS}/postinstall.sh"

hr; echo "VERIFY: quarantine cleared + perms + layout"; hr
find "$TREE" -type f -name 'terraform-provider-iru_v*' -exec sh -c '
  q=$(xattr -p com.apple.quarantine "$1" 2>/dev/null || echo CLEARED)
  echo "  $1"
  echo "    quarantine: $q"
  echo "    perms: $(stat -f "%Sp" "$1")"
' _ {} \;
echo ""
echo "mirror tree:"
find "${MIRROR}/registry.terraform.io" -type f | sed "s/^/  /"
