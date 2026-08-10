#!/bin/bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
# 项目版本号统一由环境变量 E_LIBRARY_VERSION 提供，未设置时回退到 VERSION 文件。
E_LIBRARY_VERSION="${E_LIBRARY_VERSION:-$(cat "$ROOT_DIR/VERSION")}"
VERSION="$E_LIBRARY_VERSION"

echo "==> Building frontend..."
cd "$ROOT_DIR/client/web"
npm install
npm run build

echo "==> Copying frontend dist to backend..."
rm -rf "$ROOT_DIR/backend/web/dist"
mkdir -p "$ROOT_DIR/backend/web/dist"
cp -r "$ROOT_DIR/client/web/dist/"* "$ROOT_DIR/backend/web/dist/"
# Keep the tracked go:embed placeholder so a fresh checkout still compiles.
touch "$ROOT_DIR/backend/web/dist/.gitkeep"

echo "==> Building backend..."
cd "$ROOT_DIR/backend"
CGO_ENABLED=0 go build -ldflags="-s -w -X main.Version=${VERSION}" -o kh-e-library ./cmd/server

echo "==> Packaging archive..."
tar czf "kh-e-library-${VERSION}-$(go env GOOS)-$(go env GOARCH).tar.gz" kh-e-library application.yml
rm -f application.yml kh-e-library

echo "==> Done! Archive: backend/kh-e-library-${VERSION}-$(go env GOOS)-$(go env GOARCH).tar.gz"
