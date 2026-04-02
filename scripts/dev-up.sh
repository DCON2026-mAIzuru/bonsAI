#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)"
RUN_DIR="$ROOT_DIR/.run"
LOG_DIR="$RUN_DIR/logs"
mkdir -p "$LOG_DIR"

LLM_PID_FILE="$RUN_DIR/llm.pid"
BACKEND_PID_FILE="$RUN_DIR/backend.pid"
FRONTEND_PID_FILE="$RUN_DIR/frontend.pid"

LLM_HOST="${BONSAI_LLM_HOST:-127.0.0.1}"
LLM_PORT="${BONSAI_LLM_PORT:-8081}"
BACKEND_PORT="${BONSAI_BACKEND_PORT:-8082}"
FRONTEND_PORT="${BONSAI_FRONTEND_PORT:-5173}"

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

wait_for_http() {
  local url="$1"
  local name="$2"

  for _ in $(seq 1 120); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      echo "$name is ready: $url"
      return 0
    fi
    sleep 1
  done

  echo "$name did not become ready: $url" >&2
  return 1
}

ensure_command curl
ensure_command npm
ensure_command go

if ! command -v llama-server >/dev/null 2>&1 && [ ! -x "$ROOT_DIR/bonsAI_LLM/scripts/start_qwen4b_server.sh" ]; then
  echo "llama-server is not available" >&2
  exit 1
fi

if ! is_running_pidfile "$LLM_PID_FILE"; then
  (
    cd "$ROOT_DIR/bonsAI_LLM"
    nohup ./scripts/start_qwen4b_server.sh >"$LOG_DIR/llm.log" 2>&1 &
    echo $! >"$LLM_PID_FILE"
  )
  echo "started llama.cpp server"
else
  echo "llama.cpp server already running"
fi

wait_for_http "http://$LLM_HOST:$LLM_PORT/health" "llama.cpp"

if ! is_running_pidfile "$BACKEND_PID_FILE"; then
  (
    cd "$ROOT_DIR/bonsAI_server"
    if command -v air >/dev/null 2>&1; then
      nohup env \
        BONSAI_SERVER_ADDR=":$BACKEND_PORT" \
        BONSAI_LLM_CHAT_STREAM_URL="http://$LLM_HOST:$LLM_PORT/v1/chat/completions" \
        BONSAI_LLM_MODEL="qwen2.5-3b" \
        air >"$LOG_DIR/backend.log" 2>&1 &
    else
      nohup env \
        BONSAI_SERVER_ADDR=":$BACKEND_PORT" \
        BONSAI_LLM_CHAT_STREAM_URL="http://$LLM_HOST:$LLM_PORT/v1/chat/completions" \
        BONSAI_LLM_MODEL="qwen2.5-3b" \
        go run . >"$LOG_DIR/backend.log" 2>&1 &
    fi
    echo $! >"$BACKEND_PID_FILE"
  )
  echo "started Go backend"
else
  echo "Go backend already running"
fi

wait_for_http "http://127.0.0.1:$BACKEND_PORT/healthz" "Go backend"

if ! is_running_pidfile "$FRONTEND_PID_FILE"; then
  (
    cd "$ROOT_DIR/bonsAI_front"
    nohup env BONSAI_DEV_API_TARGET="http://127.0.0.1:$BACKEND_PORT" npm run dev >"$LOG_DIR/frontend.log" 2>&1 &
    echo $! >"$FRONTEND_PID_FILE"
  )
  echo "started Vite frontend"
else
  echo "Vite frontend already running"
fi

wait_for_http "http://127.0.0.1:$FRONTEND_PORT" "Vite frontend"

cat <<EOF

bonsAI dev stack is up
- Frontend:  http://127.0.0.1:$FRONTEND_PORT
- Backend:   http://127.0.0.1:$BACKEND_PORT
- Qwen API:  http://$LLM_HOST:$LLM_PORT

Logs:
- $LOG_DIR/frontend.log
- $LOG_DIR/backend.log
- $LOG_DIR/llm.log
EOF
