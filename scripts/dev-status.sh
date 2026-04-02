#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)"
RUN_DIR="$ROOT_DIR/.run"

print_status() {
  local name="$1"
  local pid_file="$2"
  local url="$3"

  if [ -f "$pid_file" ]; then
    local pid
    pid="$(cat "$pid_file" 2>/dev/null || true)"
    if [ -n "$pid" ] && kill -0 "$pid" >/dev/null 2>&1; then
      echo "$name: running (pid=$pid, $url)"
      return
    fi
  fi

  echo "$name: stopped"
}

print_status "llama.cpp server" "$RUN_DIR/llm.pid" "http://127.0.0.1:8081/health"
print_status "Go backend" "$RUN_DIR/backend.pid" "http://127.0.0.1:8082/healthz"
print_status "Vite frontend" "$RUN_DIR/frontend.pid" "http://127.0.0.1:5173"
