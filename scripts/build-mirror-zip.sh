#!/usr/bin/env bash
#
# build-mirror-zip.sh — build a Terraform filesystem-mirror zip of this provider
# for internal distribution via an Iru custom app.
#
# The zip contains the darwin_arm64 + darwin_amd64 provider binaries laid out in
# Terraform's "unpacked" mirror structure:
#
#   registry.terraform.io/dmcphearson/iru/<version>/darwin_arm64/terraform-provider-iru_v<version>
#   registry.terraform.io/dmcphearson/iru/<version>/darwin_amd64/terraform-provider-iru_v<version>
#
# The Iru custom app unzips this into the machine-wide mirror
# /Library/Application Support/io.terraform/plugins, after which any user's
# `terraform init` resolves dmcphearson/iru locally (no registry, no dev_overrides).
#
# Usage: scripts/build-mirror-zip.sh [version]   (version defaults to 0.1.0)

set -euo pipefail

VERSION="${1:-0.1.0}"
HOSTNAME="registry.terraform.io"
NAMESPACE="dmcphearson"
TYPE="iru"
BINARY="terraform-provider-${TYPE}_v${VERSION}"

# Resolve repo root (this script lives in <root>/scripts).
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${ROOT_DIR}"

BUILD_DIR="${ROOT_DIR}/dist/mirror"
ZIP_PATH="${ROOT_DIR}/dist/iru-provider-${VERSION}-mirror.zip"
TREE_BASE="${HOSTNAME}/${NAMESPACE}/${TYPE}/${VERSION}"

echo "Building ${NAMESPACE}/${TYPE} ${VERSION} mirror zip..."

# Clean and recreate the staging tree.
rm -rf "${BUILD_DIR}" "${ZIP_PATH}"
mkdir -p "${BUILD_DIR}"

for arch in arm64 amd64; do
  target="darwin_${arch}"
  out="${BUILD_DIR}/${TREE_BASE}/${target}/${BINARY}"
  mkdir -p "$(dirname "${out}")"
  echo "  compiling ${target}..."
  GOOS=darwin GOARCH="${arch}" CGO_ENABLED=0 \
    go build -ldflags "-s -w -X main.version=${VERSION}" -o "${out}" .
  chmod +x "${out}"
  # Ad-hoc sign now so the binary's bytes are final. Go already ad-hoc-signs the
  # native (arm64) build, but the cross-compiled amd64 binary is unsigned; if we
  # left it unsigned the Iru postinstall would re-sign it in place, changing its
  # bytes AFTER the lockfile was generated from this zip and breaking
  # `terraform init` checksum verification on Intel Macs. Signing here keeps the
  # zip, the installed mirror, and .terraform.lock.hcl referencing identical bytes.
  if command -v codesign >/dev/null 2>&1; then
    codesign --force --sign - "${out}" >/dev/null 2>&1 || echo "  WARN: codesign failed for ${target}"
  fi
done

# Zip from inside the staging dir so the archive root is the HOSTNAME dir.
echo "  packaging ${ZIP_PATH}..."
( cd "${BUILD_DIR}" && zip -qr "${ZIP_PATH}" "${HOSTNAME}" )

echo ""
echo "Built: ${ZIP_PATH}"
echo "Layout:"
( cd "${BUILD_DIR}" && find "${HOSTNAME}" -type f | sed 's/^/  /' )
echo ""
# sha256 of the zip — record alongside the file_key when uploading to Iru.
if command -v shasum >/dev/null 2>&1; then
  echo "sha256: $(shasum -a 256 "${ZIP_PATH}" | awk '{print $1}')"
fi
