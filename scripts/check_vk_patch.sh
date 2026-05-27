#!/usr/bin/env bash
set -euo pipefail

WORKDIR="${WORKDIR:-/tmp/olcrtc-vk-check}"
PANEL_REPO="${PANEL_REPO:-https://github.com/BigDaddy3334/olcrtc-manager-panel.git}"
OLCRTC_REPO="${OLCRTC_REPO:-https://github.com/openlibrecommunity/olcrtc.git}"
PATCH_REPO="${PATCH_REPO:-https://github.com/dobezor/turnable-manager-panel.git}"

rm -rf "$WORKDIR"
mkdir -p "$WORKDIR"
cd "$WORKDIR"

git clone --depth 1 "$PANEL_REPO" panel
git clone --depth 1 "$OLCRTC_REPO" olcrtc
git clone --depth 1 "$PATCH_REPO" patch

python3 patch/scripts/patch_olcrtc_vk.py --panel ./panel --olcrtc ./olcrtc

cd "$WORKDIR/olcrtc"
go test ./...
go build -o /tmp/olcrtc-vk ./cmd/olcrtc

cd "$WORKDIR/panel"
go build -o /tmp/olcrtc-manager-vk ./cmd/olcrtc-manager

echo "OK: patched olcrtc and manager build successfully"
echo "olcrtc: /tmp/olcrtc-vk"
echo "manager: /tmp/olcrtc-manager-vk"
