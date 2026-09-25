#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

# Feiniu NAS / invalid HOME compatibility:
# Some NAS shells expose HOME=/home/fnroot even when that path does not
# exist or is not writable. Keep all temporary Go build state locally.
BUILD_STATE_DIR="$ROOT/.build-state"
SAFE_HOME="$BUILD_STATE_DIR/home"
SAFE_GOCACHE="$BUILD_STATE_DIR/go-build"
SAFE_GOPATH="$BUILD_STATE_DIR/gopath"
SAFE_GOMODCACHE="$BUILD_STATE_DIR/gomod"

mkdir -p "$SAFE_HOME" "$SAFE_GOCACHE" "$SAFE_GOPATH" "$SAFE_GOMODCACHE"

if [ -z "${HOME:-}" ] || [ ! -d "${HOME:-}" ] || [ ! -w "${HOME:-}" ]; then
  export HOME="$SAFE_HOME"
fi

# Always use writable build-local Go caches. This avoids /home/fnroot entirely.
export GOCACHE="$SAFE_GOCACHE"
export GOPATH="$SAFE_GOPATH"
export GOMODCACHE="$SAFE_GOMODCACHE"
export GOTELEMETRY=off

echo "Build environment:"
echo "  HOME=$HOME"
echo "  GOCACHE=$GOCACHE"
echo "  GOPATH=$GOPATH"
echo "  GOMODCACHE=$GOMODCACHE"
echo

command -v go >/dev/null || { echo "错误：未找到 Go。需要 Go 1.23+。"; exit 1; }
command -v curl >/dev/null || { echo "错误：未找到 curl。"; exit 1; }
command -v openssl >/dev/null || { echo "错误：未找到 openssl。"; exit 1; }
command -v python3 >/dev/null || { echo "错误：未找到 python3。"; exit 1; }

go version
echo

mkdir -p pkg/pluginapi/v1
BASE="https://raw.githubusercontent.com/MACOS-DO/sub4api/v1.1.4/backend/pkg/pluginapi/v1"
for f in plugin.proto plugin.pb.go plugin_grpc.pb.go runtime.go; do
  echo "fetch $f"
  curl --fail --location --retry 3 --retry-delay 2 \
    "$BASE/$f" \
    -o "pkg/pluginapi/v1/$f"
done

echo
echo "Resolving Go modules..."
go mod tidy

echo
echo "Formatting source..."
gofmt -w cmd internal

rm -rf build dist
mkdir -p build/linux-amd64 build/linux-arm64 dist

echo
echo "build linux-amd64"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -trimpath -ldflags="-s -w" \
  -o build/linux-amd64/basispoints-transport \
  ./cmd/basispoints-transport

echo
echo "build linux-arm64"
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
  go build -trimpath -ldflags="-s -w" \
  -o build/linux-arm64/basispoints-transport \
  ./cmd/basispoints-transport

if [ ! -f .publisher-key.pem ] && [ -n "${PUBLISHER_KEY_FILE:-}" ]; then
  echo
  echo "importing existing publisher key: $PUBLISHER_KEY_FILE"
  cp "$PUBLISHER_KEY_FILE" .publisher-key.pem
  chmod 600 .publisher-key.pem
fi

if [ ! -f .publisher-key.pem ]; then
  echo
  echo "generating local Ed25519 publisher key"
  openssl genpkey -algorithm ED25519 -out .publisher-key.pem
  chmod 600 .publisher-key.pem
else
  echo
  echo "using existing local publisher key: .publisher-key.pem"
fi

echo
echo "Packaging plugin..."
python3 tools/package.py

echo
echo "========================================"
echo "完成"
echo "========================================"
echo "插件包:"
echo "  dist/basispoints-transport-0.2.1.s2plugin"
echo
echo "Sub4API 信任公钥配置:"
echo "  dist/trusted-publisher.yaml"
echo
echo "发布私钥:"
echo "  .publisher-key.pem"
echo "  请保留并妥善保存，不要上传或分享。以后升级插件需要继续使用同一私钥。"
echo
echo "临时 Go 缓存保存在:"
echo "  .build-state/"
echo "如需清理编译缓存，可删除该目录，不影响插件源码和发布私钥。"
