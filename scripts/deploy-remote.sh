#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEPLOY_CONFIG="${ROOT_DIR}/.deploy.env"

if [[ -f "${DEPLOY_CONFIG}" ]]; then
  set -a
  # shellcheck source=/dev/null
  source "${DEPLOY_CONFIG}"
  set +a
fi

REMOTE_HOST="${NEW_API_HOST:-}"
REMOTE_PATH="${NEW_API_PATH:-/opt/new-api}"
SSH_PORT="${NEW_API_SSH_PORT:-}"
SYNC_ENV="${NEW_API_SYNC_ENV:-0}"
SKIP_PREFLIGHT="${NEW_API_SKIP_PREFLIGHT:-0}"
BUN_BIN="${BUN_BIN:-}"
REMOTE_ATTEMPTS="${NEW_API_REMOTE_ATTEMPTS:-5}"
REMOTE_RETRY_DELAY="${NEW_API_REMOTE_RETRY_DELAY:-5}"

usage() {
  cat <<'USAGE'
Usage:
  NEW_API_HOST=root@your-server NEW_API_PATH=/opt/new-api ./scripts/deploy-remote.sh
  cp .deploy.env.example .deploy.env && edit .deploy.env && ./scripts/deploy-remote.sh

Optional:
  NEW_API_SSH_PORT=22          SSH port
  NEW_API_SYNC_ENV=1           overwrite remote .env.production from local file
  NEW_API_SKIP_PREFLIGHT=1     skip remote Docker checks
  BUN_BIN=/path/to/bun         Bun binary for local frontend builds
  NEW_API_REMOTE_ATTEMPTS=5    retry transient SSH/rsync failures
  NEW_API_REMOTE_RETRY_DELAY=5 seconds between remote retries

Before the first deploy:
  ./scripts/create-production-env.sh https://your-domain.example
USAGE
}

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required command: $1" >&2
    exit 1
  fi
}

retry_remote() {
  local attempt=1

  while true; do
    if "$@"; then
      return 0
    fi
    if (( attempt >= REMOTE_ATTEMPTS )); then
      return 1
    fi

    attempt=$((attempt + 1))
    echo "Remote command failed. Retrying in ${REMOTE_RETRY_DELAY}s (${attempt}/${REMOTE_ATTEMPTS})..."
    sleep "${REMOTE_RETRY_DELAY}"
  done
}

find_bun() {
  if [[ -n "${BUN_BIN}" && -x "${BUN_BIN}" ]]; then
    printf '%s' "${BUN_BIN}"
    return
  fi
  if command -v bun >/dev/null 2>&1; then
    command -v bun
    return
  fi
  if [[ -x "${HOME}/.bun/bin/bun" ]]; then
    printf '%s' "${HOME}/.bun/bin/bun"
    return
  fi
  echo "Missing Bun. Install Bun or set BUN_BIN=/path/to/bun." >&2
  exit 1
}

build_frontend() {
  local bun_bin version
  bun_bin="$(find_bun)"
  version="$(cat "${ROOT_DIR}/VERSION")"

  echo "Building frontend assets locally with ${bun_bin}..."
  (
    cd "${ROOT_DIR}/web/default"
    "${bun_bin}" install
    DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION="${version}" "${bun_bin}" --bun run build
  )
  (
    cd "${ROOT_DIR}/web/classic"
    "${bun_bin}" install
    VITE_REACT_APP_VERSION="${version}" "${bun_bin}" --bun run build
  )
}

remote_quote() {
  printf "'%s'" "$(printf '%s' "$1" | sed "s/'/'\\\\''/g")"
}

if [[ -z "${REMOTE_HOST}" || "${REMOTE_HOST}" == "-h" || "${REMOTE_HOST}" == "--help" ]]; then
  usage
  exit 1
fi

require_command ssh
require_command rsync
build_frontend

SSH_ARGS=(-o BatchMode=yes -o ConnectTimeout=20 -o ConnectionAttempts=3 -o ServerAliveInterval=30 -o ServerAliveCountMax=3)
RSYNC_SSH=(ssh "${SSH_ARGS[@]}")
if [[ -n "${SSH_PORT}" ]]; then
  SSH_ARGS+=(-p "${SSH_PORT}")
  RSYNC_SSH+=( -p "${SSH_PORT}" )
fi

REMOTE_PATH_Q="$(remote_quote "${REMOTE_PATH}")"

echo "Ensuring remote directory exists: ${REMOTE_HOST}:${REMOTE_PATH}"
retry_remote ssh "${SSH_ARGS[@]}" "${REMOTE_HOST}" "mkdir -p ${REMOTE_PATH_Q}"

echo "Syncing source to server..."
retry_remote rsync -az --delete \
  -e "${RSYNC_SSH[*]}" \
  --exclude '.git/' \
  --exclude '.env' \
  --exclude '.env.production' \
  --exclude '.deploy.env' \
  --exclude '.cache/' \
  --exclude '.gocache/' \
  --exclude '.gomodcache/' \
  --exclude '.gopath/' \
  --exclude '.local-*' \
  --exclude 'data/' \
  --exclude 'logs/' \
  --exclude 'web/default/node_modules/' \
  --exclude 'web/classic/node_modules/' \
  "${ROOT_DIR}/" "${REMOTE_HOST}:${REMOTE_PATH}/"

if [[ "${SYNC_ENV}" == "1" ]]; then
  if [[ ! -f "${ROOT_DIR}/.env.production" ]]; then
    echo "NEW_API_SYNC_ENV=1 was set, but local .env.production does not exist." >&2
    exit 1
  fi
  echo "Uploading .env.production because NEW_API_SYNC_ENV=1..."
  retry_remote rsync -az -e "${RSYNC_SSH[*]}" "${ROOT_DIR}/.env.production" "${REMOTE_HOST}:${REMOTE_PATH}/.env.production"
else
  if ! retry_remote ssh "${SSH_ARGS[@]}" "${REMOTE_HOST}" "test -f ${REMOTE_PATH_Q}/.env.production"; then
    if [[ ! -f "${ROOT_DIR}/.env.production" ]]; then
      echo "Remote .env.production is missing and local .env.production does not exist." >&2
      echo "Run: ./scripts/create-production-env.sh https://your-domain.example" >&2
      exit 1
    fi
    echo "Uploading initial .env.production..."
    retry_remote rsync -az -e "${RSYNC_SSH[*]}" "${ROOT_DIR}/.env.production" "${REMOTE_HOST}:${REMOTE_PATH}/.env.production"
  else
    echo "Keeping existing remote .env.production. Set NEW_API_SYNC_ENV=1 to overwrite it."
  fi
fi

if [[ "${SKIP_PREFLIGHT}" != "1" ]]; then
  echo "Running remote preflight checks..."
  retry_remote ssh "${SSH_ARGS[@]}" "${REMOTE_HOST}" "cd ${REMOTE_PATH_Q} && command -v docker >/dev/null && docker compose version >/dev/null && docker info >/dev/null"
fi

echo "Building and starting production stack..."
retry_remote ssh "${SSH_ARGS[@]}" "${REMOTE_HOST}" "cd ${REMOTE_PATH_Q} && docker compose --env-file .env.production -f docker-compose.prod.yml up -d --build"

echo "Current production services:"
retry_remote ssh "${SSH_ARGS[@]}" "${REMOTE_HOST}" "cd ${REMOTE_PATH_Q} && docker compose --env-file .env.production -f docker-compose.prod.yml ps"

echo "Deployment command finished."
