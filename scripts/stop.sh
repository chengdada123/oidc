#!/usr/bin/env sh
set -eu

PORT="${PORT:-8080}"

if command -v lsof >/dev/null 2>&1; then
  pids="$(lsof -ti tcp:"$PORT" || true)"
  if [ -z "$pids" ]; then
    echo "No process is listening on port $PORT"
    exit 0
  fi
  kill $pids || true
  echo "Stopped process(es) on port $PORT"
  exit 0
fi

if command -v fuser >/dev/null 2>&1; then
  fuser -k "${PORT}/tcp" || true
  echo "Stopped process(es) on port $PORT"
  exit 0
fi

echo "Neither lsof nor fuser is available; stop the process manually." >&2
exit 1
