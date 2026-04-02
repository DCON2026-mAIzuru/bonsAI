#!/usr/bin/env bash
set -eu

SCRIPT_DIR="$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)"
ROOT_DIR="$(CDPATH='' cd -- "$SCRIPT_DIR/.." && pwd)"
ENV_FILE="${BONSAI_LLM_ENV_FILE:-$ROOT_DIR/.env}"

if [ -f "$ENV_FILE" ]; then
  # shellcheck disable=SC1090
  . "$ENV_FILE"
elif [ -z "${BONSAI_LLM_SERVER_BIN:-}" ] && [ -z "${BONSAI_LLM_HF_REPO:-}" ] && [ -z "${BONSAI_LLM_MODEL_FILE:-}" ]; then
  echo "missing env file: $ENV_FILE" >&2
  echo "copy bonsAI_LLM/.env.example to bonsAI_LLM/.env first or provide BONSAI_LLM_* env vars" >&2
  exit 1
fi

LLAMA_CPP_DIR="${LLAMA_CPP_DIR:-$ROOT_DIR/vendor/llama.cpp}"
BUILD_DIR="${BONSAI_LLM_BUILD_DIR:-$LLAMA_CPP_DIR/build}"
SERVER_BIN="${BONSAI_LLM_SERVER_BIN:-$BUILD_DIR/bin/llama-server}"
MODEL_FILE="${BONSAI_LLM_MODEL_FILE:-}"
HF_REPO="${BONSAI_LLM_HF_REPO:-}"
HF_FILE="${BONSAI_LLM_HF_FILE:-}"
HOST="${BONSAI_LLM_HOST:-127.0.0.1}"
PORT="${BONSAI_LLM_PORT:-8081}"
ALIAS="${BONSAI_LLM_ALIAS:-qwen2.5-3b}"
CTX_SIZE="${BONSAI_LLM_CTX_SIZE:-2048}"
THREADS="${BONSAI_LLM_THREADS:-4}"
PARALLEL="${BONSAI_LLM_PARALLEL:-1}"
BATCH="${BONSAI_LLM_BATCH:-256}"
UBATCH="${BONSAI_LLM_UBATCH:-128}"
GPU_LAYERS="${BONSAI_LLM_GPU_LAYERS:-0}"

if [ ! -x "$SERVER_BIN" ]; then
  echo "llama-server not found: $SERVER_BIN" >&2
  echo "run ./scripts/bootstrap_llama_cpp.sh first" >&2
  exit 1
fi

set -- \
  "$SERVER_BIN" \
  --host "$HOST" \
  --port "$PORT" \
  -c "$CTX_SIZE" \
  -t "$THREADS" \
  -np "$PARALLEL" \
  -b "$BATCH" \
  -ub "$UBATCH" \
  -ngl "$GPU_LAYERS"

if [ -n "$HF_REPO" ]; then
  set -- "$@" --hf-repo "$HF_REPO"
  if [ -n "$HF_FILE" ]; then
    set -- "$@" --hf-file "$HF_FILE"
  fi
else
  if [ -z "$MODEL_FILE" ] || [ ! -f "$MODEL_FILE" ]; then
    echo "GGUF model not found: $MODEL_FILE" >&2
    echo "set BONSAI_LLM_MODEL_FILE or BONSAI_LLM_HF_REPO in $ENV_FILE" >&2
    exit 1
  fi
  set -- "$@" -m "$MODEL_FILE"
fi

if [ -n "$ALIAS" ]; then
  set -- "$@" --alias "$ALIAS"
fi

exec "$@"
