#!/usr/bin/env sh
set -eu

PORT="${PORT:-8080}"
BASE_URL_OVERRIDE="${BASE_URL_OVERRIDE:-}"
STOP_FIRST="${STOP_FIRST:-0}"

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
REPO_ROOT="$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

if [ ! -f .env ]; then
  echo ".env not found. Copy .env.example to .env first." >&2
  exit 1
fi

mkdir -p data

if ! command -v go >/dev/null 2>&1; then
  echo "go not found in PATH" >&2
  exit 1
fi

if [ "$STOP_FIRST" = "1" ]; then
  if command -v lsof >/dev/null 2>&1; then
    pids="$(lsof -ti tcp:"$PORT" || true)"
    if [ -n "$pids" ]; then
      kill $pids || true
    fi
  elif command -v fuser >/dev/null 2>&1; then
    fuser -k "${PORT}/tcp" || true
  fi
fi

set -a
. ./.env
set +a

export PORT
if [ -n "$BASE_URL_OVERRIDE" ]; then
  export BASE_URL="$BASE_URL_OVERRIDE"
fi

go build -o oidc-bridge ./cmd/bridge
nohup ./oidc-bridge >/tmp/oidc-bridge.log 2>&1 &
PID=$!
sleep 2

echo "OIDC Bridge started"
echo "PID: $PID"
echo "Port: $PORT"
echo "Base URL: ${BASE_URL:-}"
echo "Log: /tmp/oidc-bridge.log"
