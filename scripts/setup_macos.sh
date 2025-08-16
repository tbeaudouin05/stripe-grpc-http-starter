#!/usr/bin/env bash
# Setup script for ai-mails-backend on macOS
# - Installs required CLI tools: pnpm, buf, sqlc, mockgen
# - Installs Node dev deps (Prisma)
# - Creates a pre-commit hook to run `make generate` and `make test`

set -euo pipefail

PROJECT_ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$PROJECT_ROOT_DIR"

log() { echo "[setup] $*"; }
warn() { echo "[setup][warn] $*" >&2; }
err() { echo "[setup][error] $*" >&2; }

detected_os="$(uname -s || true)"
if [[ "$detected_os" != "Darwin" ]]; then
  warn "This script targets macOS (Darwin). Detected: $detected_os"
fi

require_cmd() {
  # usage: require_cmd <cmd> <install_instructions>
  local cmd="$1"; shift
  local how="$*"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    err "Missing required command: $cmd";
    err "Install hint: $how";
    return 1
  fi
}

ensure_homebrew() {
  if command -v brew >/dev/null 2>&1; then
    return 0
  fi
  warn "Homebrew is not installed. Some tools will not be installed automatically."
  warn "Install Homebrew from https://brew.sh and re-run this script, or install tools manually."
  return 1
}

ensure_go() {
  if command -v go >/dev/null 2>&1; then
    return 0
  fi
  warn "Go is not installed. Install via Homebrew: brew install go (or use goenv), then re-run."
  return 1
}

ensure_node_pnpm() {
  if command -v pnpm >/dev/null 2>&1; then
    return 0
  fi
  # Try corepack first
  if command -v corepack >/dev/null 2>&1; then
    log "Enabling corepack and activating pnpm..."
    corepack enable || true
    corepack prepare pnpm@9 --activate || true
  fi
  if command -v pnpm >/dev/null 2>&1; then
    return 0
  fi
  # Fallback to installing Node via Homebrew to get corepack
  if ensure_homebrew; then
    log "Installing Node (with corepack) via Homebrew..."
    brew install node || true
    if command -v corepack >/dev/null 2>&1; then
      corepack enable || true
      corepack prepare pnpm@9 --activate || true
    fi
  fi
  require_cmd pnpm "corepack enable && corepack prepare pnpm@9 --activate (Node >= 16 required)"
}

ensure_buf() {
  if command -v buf >/dev/null 2>&1; then return 0; fi
  if ensure_homebrew; then
    log "Installing buf via Homebrew..."
    brew install bufbuild/buf/buf || true
  fi
  require_cmd buf "brew install bufbuild/buf/buf"
}

ensure_sqlc() {
  if command -v sqlc >/dev/null 2>&1; then return 0; fi
  if ensure_homebrew; then
    log "Installing sqlc via Homebrew..."
    brew install sqlc || true
  fi
  require_cmd sqlc "brew install sqlc (or see https://docs.sqlc.dev/en/latest/overview/install.html)"
}

ensure_mockgen() {
  ensure_go || true
  if command -v mockgen >/dev/null 2>&1; then return 0; fi
  log "Installing mockgen via 'go install'..."
  # Pin to version used in go.mod major version (v1.6.0 is current dep)
  GO111MODULE=on go install github.com/golang/mock/mockgen@v1.6.0 || true
  require_cmd mockgen "GO111MODULE=on go install github.com/golang/mock/mockgen@v1.6.0"
}

install_node_deps() {
  log "Installing Node dev dependencies with pnpm..."
  pnpm install --silent
}

setup_pre_commit_hook() {
  local hooks_dir=".git/hooks"
  local hook_file="$hooks_dir/pre-commit"
  local marker_begin="# === ai-mails-backend pre-commit BEGIN ==="
  local marker_end="# === ai-mails-backend pre-commit END ==="

  if [[ ! -d ".git" ]]; then
    warn ".git directory not found. Skipping git hook setup."
    return 0
  fi
  mkdir -p "$hooks_dir"

  local hook_content
  hook_content="#!/bin/sh
set -e
$marker_begin
# Run code generation and tests before committing
# You can bypass this hook with: git commit --no-verify

echo '[pre-commit] make generate'
make generate

echo '[pre-commit] make test'
make test
$marker_end
"

  if [[ -f "$hook_file" ]]; then
    if grep -q "$marker_begin" "$hook_file" 2>/dev/null; then
      log "Pre-commit hook already contains our block. Refreshing..."
      # Replace existing block between markers
      awk -v begin="$marker_begin" -v end="$marker_end" -v repl="$hook_content" '
        BEGIN{printed=0}
        {
          if($0==begin){inblock=1; if(!printed){printf("%s", repl); printed=1}}
          else if($0==end){inblock=0}
          else if(!inblock && !printed){print}
        }
        END{if(!printed) printf("%s", repl)}
      ' "$hook_file" > "$hook_file.tmp" && mv "$hook_file.tmp" "$hook_file"
    else
      log "Backing up existing pre-commit hook and installing ours..."
      cp "$hook_file" "$hook_file.backup" || true
      printf "%s" "$hook_content" > "$hook_file"
    fi
  else
    log "Installing new pre-commit hook..."
    printf "%s" "$hook_content" > "$hook_file"
  fi
  chmod +x "$hook_file"
}

main() {
  log "Starting setup in $PROJECT_ROOT_DIR"
  ensure_go || true
  ensure_node_pnpm
  ensure_buf
  ensure_sqlc
  ensure_mockgen
  install_node_deps
  setup_pre_commit_hook
  log "Setup completed successfully. You can now run: make generate && make test"
}

main "$@"
