#!/usr/bin/env bash
set -euo pipefail

mkdir -p dist

go build -tags with_quic -o dist/easy_proxies_linux_amd64 ./cmd/easy_proxies

echo "Built: dist/easy_proxies_linux_amd64"
