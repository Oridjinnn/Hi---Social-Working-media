#!/usr/bin/env bash
set -euo pipefail

OUTDIR="$(dirname "$0")/../bin"
mkdir -p "$OUTDIR"

TARGET="${1:-all}"
ARCH="${2:-amd64}"

case "$TARGET" in
  all)
    GOOS_LIST=("linux" "darwin" "windows")
    ;;
  linux|darwin|windows)
    GOOS_LIST=("$TARGET")
    ;;
  *)
    echo "Usage: $0 [all|linux|darwin|windows] [amd64|arm64]"
    exit 1
    ;;
esac

for GOOS in "${GOOS_LIST[@]}"; do
  OUTFILE="$OUTDIR/hi-${GOOS}-${ARCH}"
  EXT=""
  if [ "$GOOS" = "windows" ]; then EXT=".exe"; fi
  echo "Building for $GOOS/$ARCH -> ${OUTFILE}${EXT}"
  GOOS="$GOOS" GOARCH="$ARCH" go build -o "${OUTFILE}${EXT}" .
done

echo "Build complete. Binaries in: $OUTDIR"
