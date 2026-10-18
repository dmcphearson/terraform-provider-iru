#!/bin/bash
#
# Preinstall for the "Terraform Provider: dmcphearson/iru" Iru custom app.
#
# Ensures Terraform is available before the provider binary is dropped into the
# machine-wide filesystem mirror. Users typically self-install Terraform; if it is
# already on PATH we leave their install untouched. Only when it is missing do we
# install a pinned version from releases.hashicorp.com, verified against a pinned
# SHA256 checksum.
#
# Runs as root under the Iru agent.

set -euo pipefail

TF_VERSION="1.14.8"
INSTALL_DIR="/usr/local/bin"

# Pinned SHA256 checksums for terraform_${TF_VERSION}_darwin_<arch>.zip
# (from https://releases.hashicorp.com/terraform/${TF_VERSION}/terraform_${TF_VERSION}_SHA256SUMS).
SHA256_ARM64="5593670a2d42323847bfb216db17c73a44df201a62f7587928bae16adeabba23"
SHA256_AMD64="26dd7593d22e9d99ec09380f0869718f649be7b7f954d888611335e6a84961f8"

log() { echo "[iru-provider preinstall] $*"; }
fail() { echo "[iru-provider preinstall] ERROR: $*" >&2; exit 1; }

# Leave an existing Terraform install alone.
if command -v terraform >/dev/null 2>&1; then
  log "Terraform already present ($(command -v terraform)); leaving it untouched."
  exit 0
fi

log "Terraform not found; installing pinned version ${TF_VERSION}."

case "$(uname -m)" in
  arm64) TF_ARCH="arm64"; TF_SHA256="${SHA256_ARM64}" ;;
  x86_64) TF_ARCH="amd64"; TF_SHA256="${SHA256_AMD64}" ;;
  *) fail "unsupported architecture: $(uname -m)" ;;
esac

ZIP_NAME="terraform_${TF_VERSION}_darwin_${TF_ARCH}.zip"
URL="https://releases.hashicorp.com/terraform/${TF_VERSION}/${ZIP_NAME}"

TMPDIR="$(mktemp -d)"
trap 'rm -rf "${TMPDIR}"' EXIT

log "Downloading ${URL}"
# --fail: error on HTTP >=400; --retry: transient network resilience.
curl --fail --location --silent --show-error \
  --retry 3 --retry-delay 2 \
  -o "${TMPDIR}/${ZIP_NAME}" "${URL}" \
  || fail "download failed"

log "Verifying checksum"
ACTUAL_SHA256="$(shasum -a 256 "${TMPDIR}/${ZIP_NAME}" | awk '{print $1}')"
if [ "${ACTUAL_SHA256}" != "${TF_SHA256}" ]; then
  fail "checksum mismatch: expected ${TF_SHA256}, got ${ACTUAL_SHA256}"
fi

log "Installing to ${INSTALL_DIR}/terraform"
mkdir -p "${INSTALL_DIR}"
unzip -o -q "${TMPDIR}/${ZIP_NAME}" -d "${TMPDIR}"
install -m 0755 "${TMPDIR}/terraform" "${INSTALL_DIR}/terraform"

log "Installed $("${INSTALL_DIR}/terraform" version | head -1)"
