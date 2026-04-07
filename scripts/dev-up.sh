#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)"
RUN_DIR="$ROOT_DIR/.run"
LOG_DIR="$RUN_DIR/logs"
mkdir -p "$LOG_DIR"

LLM_ENV_FILE="${BONSAI_LLM_ENV_FILE:-$ROOT_DIR/bonsAI_LLM/.env}"
if [ -f "$LLM_ENV_FILE" ]; then
  # shellcheck disable=SC1090
  . "$LLM_ENV_FILE"
fi

LLM_PID_FILE="$RUN_DIR/llm.pid"
BACKEND_PID_FILE="$RUN_DIR/backend.pid"
FRONTEND_PID_FILE="$RUN_DIR/frontend.pid"

LLM_HOST="${BONSAI_LLM_HOST:-127.0.0.1}"
LLM_PORT="${BONSAI_LLM_PORT:-8081}"
BACKEND_PORT="${BONSAI_BACKEND_PORT:-8082}"
FRONTEND_PORT="${BONSAI_FRONTEND_PORT:-5173}"
LLM_WAIT_SECONDS="${BONSAI_LLM_WAIT_SECONDS:-900}"
BACKEND_WAIT_SECONDS="${BONSAI_BACKEND_WAIT_SECONDS:-120}"
FRONTEND_WAIT_SECONDS="${BONSAI_FRONTEND_WAIT_SECONDS:-120}"

ensure_command() {
  local cmd="$1"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "missing command: $cmd" >&2
    exit 1
  fi
}

is_running_pidfile() {
  local pid_file="$1"
  if [ ! -f "$pid_file" ]; then
    return 1
  fi

  local pid
  pid="$(cat "$pid_file" 2>/dev/null || true)"
  if [ -z "$pid" ]; then
    return 1
  fi

  kill -0 "$pid" >/dev/null 2>&1
}

print_recent_log() {
  local log_file="$1"
  if [ -f "$log_file" ]; then
    echo "---- recent log: $log_file ----" >&2
    tail -n 40 "$log_file" >&2 || true
    echo "--------------------------------" >&2
  fi
}

wait_for_http() {
  local url="$1"
  local name="$2"
  local pid_file="$3"
  local log_file="$4"
  local timeout_seconds="$5"

  for _ in $(seq 1 "$timeout_seconds"); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      echo "$name is ready: $url"
      return 0
    fi

    if [ -n "$pid_file" ] && [ -f "$pid_file" ] && ! is_running_pidfile "$pid_file"; then
      echo "$name exited before becoming ready: $url" >&2
      print_recent_log "$log_file"
      return 1
    fi

    sleep 1
  done

  echo "$name did not become ready within ${timeout_seconds}s: $url" >&2
  print_recent_log "$log_file"
  return 1
}

ensure_command curl
ensure_command go

FRONTEND_RUNNER=""
if command -v bun >/dev/null 2>&1; then
  FRONTEND_RUNNER="bun"
elif command -v npm >/dev/null 2>&1; then
  FRONTEND_RUNNER="npm"
else
  echo "missing command: bun or npm" >&2
  exit 1
fi

if ! command -v llama-server >/dev/null 2>&1 && [ ! -x "$ROOT_DIR/bonsAI_LLM/scripts/start_llm_server.sh" ]; then
  echo "llama-server is not available" >&2
  exit 1
fi

if ! is_running_pidfile "$LLM_PID_FILE"; then
  (
    cd "$ROOT_DIR/bonsAI_LLM"
    nohup ./scripts/start_llm_server.sh >"$LOG_DIR/llm.log" 2>&1 &
    echo $! >"$LLM_PID_FILE"
  )
  echo "started llama.cpp server"
else
  echo "llama.cpp server already running"
fi

wait_for_http \
  "http://$LLM_HOST:$LLM_PORT/health" \
  "llama.cpp" \
  "$LLM_PID_FILE" \
  "$LOG_DIR/llm.log" \
  "$LLM_WAIT_SECONDS"

if ! is_running_pidfile "$BACKEND_PID_FILE"; then
  (
    cd "$ROOT_DIR/bonsAI_server"
    if command -v air >/dev/null 2>&1; then
      nohup env \
        BONSAI_SERVER_ADDR=":$BACKEND_PORT" \
        BONSAI_LLM_CHAT_STREAM_URL="http://$LLM_HOST:$LLM_PORT/v1/chat/completions" \
        BONSAI_LLM_MODEL="${BONSAI_LLM_ALIAS:-gemma-4-e2b-it}" \
        air >"$LOG_DIR/backend.log" 2>&1 &
    else
      nohup env \
        BONSAI_SERVER_ADDR=":$BACKEND_PORT" \
        BONSAI_LLM_CHAT_STREAM_URL="http://$LLM_HOST:$LLM_PORT/v1/chat/completions" \
        BONSAI_LLM_MODEL="${BONSAI_LLM_ALIAS:-gemma-4-e2b-it}" \
        go run . >"$LOG_DIR/backend.log" 2>&1 &
    fi
    echo $! >"$BACKEND_PID_FILE"
  )
  echo "started Go backend"
else
  echo "Go backend already running"
fi

wait_for_http \
  "http://127.0.0.1:$BACKEND_PORT/healthz" \
  "Go backend" \
  "$BACKEND_PID_FILE" \
  "$LOG_DIR/backend.log" \
  "$BACKEND_WAIT_SECONDS"

if ! is_running_pidfile "$FRONTEND_PID_FILE"; then
  (
    cd "$ROOT_DIR/bonsAI_front"
    if [ "$FRONTEND_RUNNER" = "bun" ]; then
      nohup env BONSAI_DEV_API_TARGET="http://127.0.0.1:$BACKEND_PORT" bun run dev >"$LOG_DIR/frontend.log" 2>&1 &
    else
      nohup env BONSAI_DEV_API_TARGET="http://127.0.0.1:$BACKEND_PORT" npm run dev -- --host 0.0.0.0 >"$LOG_DIR/frontend.log" 2>&1 &
    fi
    echo $! >"$FRONTEND_PID_FILE"
  )
  echo "started Vite frontend ($FRONTEND_RUNNER)"
else
  echo "Vite frontend already running"
fi

wait_for_http \
  "http://127.0.0.1:$FRONTEND_PORT" \
  "Vite frontend" \
  "$FRONTEND_PID_FILE" \
  "$LOG_DIR/frontend.log" \
  "$FRONTEND_WAIT_SECONDS"

cat <<EOF

bonsAI dev stack is up
- Frontend:  http://127.0.0.1:$FRONTEND_PORT
- Backend:   http://127.0.0.1:$BACKEND_PORT
- LLM API:   http://$LLM_HOST:$LLM_PORT

Logs:
- $LOG_DIR/frontend.log
- $LOG_DIR/backend.log
- $LOG_DIR/llm.log
EOF
