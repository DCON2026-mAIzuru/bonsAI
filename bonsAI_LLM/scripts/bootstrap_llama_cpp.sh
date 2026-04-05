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

find_cmake() {
  if command -v cmake >/dev/null 2>&1; then
    command -v cmake
    return 0
  fi

  for brew_bin in "$(command -v brew 2>/dev/null || true)" /opt/homebrew/bin/brew /usr/local/bin/brew; do
    if [ -x "$brew_bin" ]; then
      local brew_cmake
      brew_cmake="$("$brew_bin" --prefix cmake 2>/dev/null || true)"
      if [ -n "$brew_cmake" ] && [ -x "$brew_cmake/bin/cmake" ]; then
        printf '%s\n' "$brew_cmake/bin/cmake"
        return 0
      fi
    fi
  done

  return 1
}

CMAKE_BIN="$(find_cmake || true)"

if [ -z "$CMAKE_BIN" ]; then
  echo "cmake not found" >&2
  echo "install cmake or add it to PATH first" >&2
  exit 1
fi

if [ ! -d "$LLAMA_CPP_DIR/.git" ]; then
  git clone --depth=1 https://github.com/ggml-org/llama.cpp.git "$LLAMA_CPP_DIR"
fi

"$CMAKE_BIN" -S "$LLAMA_CPP_DIR" -B "$BUILD_DIR" -DCMAKE_BUILD_TYPE=Release
"$CMAKE_BIN" --build "$BUILD_DIR" --config Release -j "$BUILD_JOBS"

echo "llama.cpp build completed at: $BUILD_DIR"
