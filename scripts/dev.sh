#!/usr/bin/env bash
# 로컬 개발 서버 기동.
#   ./scripts/dev.sh            둘 다
#   ./scripts/dev.sh backend    백엔드만
#   ./scripts/dev.sh frontend   프론트만
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# backend/.env 가 있으면 환경변수로 불러온다 (GEMINI_API_KEY 등)
[ -f "$ROOT/backend/.env" ] && set -a && . "$ROOT/backend/.env" && set +a
LOGS="$ROOT/.dev"
BACKEND_PORT="${BACKEND_PORT:-8080}"
FRONTEND_PORT="${FRONTEND_PORT:-5173}"
TARGET="${1:-all}"

mkdir -p "$LOGS"

# 포트에 남은 프로세스를 먼저 정리한다. 이게 없으면 새 서버가 뜨지 못하고
# 예전 프로세스가 계속 응답해서 코드를 고쳐도 반영이 안 된 것처럼 보인다.
free_port() {
  local pids
  pids="$(lsof -ti:"$1" 2>/dev/null || true)"
  [ -n "$pids" ] && kill -9 $pids 2>/dev/null || true
}

wait_for() {
  local url="$1" name="$2" log="$3"
  for _ in $(seq 1 60); do
    curl -sS -o /dev/null "$url" 2>/dev/null && return 0
    sleep 0.5
  done
  echo "✗ $name 기동 실패. 로그: $log" >&2
  tail -20 "$log" >&2
  return 1
}

start_backend() {
  command -v go >/dev/null || { echo "✗ go 가 필요합니다: brew install go" >&2; exit 1; }
  free_port "$BACKEND_PORT"
  echo "· 백엔드 기동 중 (:$BACKEND_PORT)"
  (cd "$ROOT/backend" && PORT="$BACKEND_PORT" nohup go run ./cmd/server >"$LOGS/backend.log" 2>&1 </dev/null &)
  wait_for "http://localhost:$BACKEND_PORT/health" 백엔드 "$LOGS/backend.log"
  echo "✓ 백엔드  http://localhost:$BACKEND_PORT"
}

start_frontend() {
  command -v npm >/dev/null || { echo "✗ npm 이 필요합니다: brew install node" >&2; exit 1; }
  [ -d "$ROOT/frontend/node_modules" ] || (cd "$ROOT/frontend" && npm install)
  free_port "$FRONTEND_PORT"
  echo "· 프론트 기동 중 (:$FRONTEND_PORT)"
  (cd "$ROOT/frontend" && nohup npm run dev >"$LOGS/frontend.log" 2>&1 </dev/null &)
  wait_for "http://localhost:$FRONTEND_PORT" 프론트 "$LOGS/frontend.log"
  echo "✓ 프론트  http://localhost:$FRONTEND_PORT"
}

case "$TARGET" in
  backend) start_backend ;;
  frontend) start_frontend ;;
  all) start_backend; start_frontend ;;
  *) echo "사용법: $0 [all|backend|frontend]" >&2; exit 1 ;;
esac

echo
echo "로그: $LOGS/*.log    종료: ./scripts/stop.sh"
