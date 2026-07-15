#!/bin/bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
VERSION="$(cat "$ROOT_DIR/VERSION")"

echo "==> Building frontend..."
cd "$ROOT_DIR/client/web"
npm install
npm run build

echo "==> Copying frontend dist to backend..."
mkdir -p "$ROOT_DIR/backend/web/dist"
cp -r "$ROOT_DIR/client/web/dist/"* "$ROOT_DIR/backend/web/dist/"

echo "==> Building backend..."
cd "$ROOT_DIR/backend"
go build -ldflags="-s -w -X main.Version=${VERSION}" -o kh-e-library ./cmd/server

echo "==> Done! Binary: backend/e-library"
