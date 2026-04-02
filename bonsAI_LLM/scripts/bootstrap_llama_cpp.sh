#!/usr/bin/env bash
set -eu

SCRIPT_DIR="$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)"
ROOT_DIR="$(CDPATH='' cd -- "$SCRIPT_DIR/.." && pwd)"
ENV_FILE="${BONSAI_LLM_ENV_FILE:-$ROOT_DIR/.env}"

if [ -f "$ENV_FILE" ]; then
  # shellcheck disable=SC1090
  . "$ENV_FILE"
fi

LLAMA_CPP_DIR="${LLAMA_CPP_DIR:-$ROOT_DIR/vendor/llama.cpp}"
BUILD_DIR="${BONSAI_LLM_BUILD_DIR:-$LLAMA_CPP_DIR/build}"
BUILD_JOBS="${BONSAI_BUILD_JOBS:-4}"

if [ ! -d "$LLAMA_CPP_DIR/.git" ]; then
  git clone --depth=1 https://github.com/ggml-org/llama.cpp.git "$LLAMA_CPP_DIR"
fi

cmake -S "$LLAMA_CPP_DIR" -B "$BUILD_DIR" -DCMAKE_BUILD_TYPE=Release
cmake --build "$BUILD_DIR" --config Release -j "$BUILD_JOBS"

echo "llama.cpp build completed at: $BUILD_DIR"
