#!/bin/bash
set -euo pipefail

VERSION="${1:-snapshot}"
ARTIFACT_DIR="dist/release"
ARTIFACT_NAME="ping-tracker-windows-amd64-${VERSION}.exe"

echo "Building ping-tracker Wails app for Windows (amd64)..."
wails build -platform windows/amd64 -clean

mkdir -p "${ARTIFACT_DIR}"
cp "build/bin/ping-tracker.exe" "${ARTIFACT_DIR}/${ARTIFACT_NAME}"

if command -v sha256sum >/dev/null 2>&1; then
  sha256sum "${ARTIFACT_DIR}/${ARTIFACT_NAME}" > "${ARTIFACT_DIR}/${ARTIFACT_NAME}.sha256"
fi

echo "Done: ${ARTIFACT_DIR}/${ARTIFACT_NAME}"
if [ -f "${ARTIFACT_DIR}/${ARTIFACT_NAME}.sha256" ]; then
  echo "Checksum: ${ARTIFACT_DIR}/${ARTIFACT_NAME}.sha256"
fi
