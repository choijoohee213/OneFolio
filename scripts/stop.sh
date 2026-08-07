#!/usr/bin/env bash
# 개발 서버 종료.
set -euo pipefail

BACKEND_PORT="${BACKEND_PORT:-8080}"
FRONTEND_PORT="${FRONTEND_PORT:-5173}"

for port in "$BACKEND_PORT" "$FRONTEND_PORT"; do
  pids="$(lsof -ti:"$port" 2>/dev/null || true)"
  if [ -n "$pids" ]; then
    kill -9 $pids 2>/dev/null || true
    echo "✓ :$port 종료"
  else
    echo "· :$port 실행 중 아님"
  fi
done
