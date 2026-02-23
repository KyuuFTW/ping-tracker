#!/bin/bash
set -e

echo "Building ping-tracker for Windows (amd64)..."
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o ping-tracker.exe .
echo "Done: ping-tracker.exe ($(du -h ping-tracker.exe | cut -f1))"
