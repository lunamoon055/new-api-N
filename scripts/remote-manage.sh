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
ACTION="${1:-}"
SERVICE="${2:-new-api}"

usage() {
  cat <<'USAGE'
Usage:
  NEW_API_HOST=root@your-server NEW_API_PATH=/opt/new-api ./scripts/remote-manage.sh ps
  cp .deploy.env.example .deploy.env && edit .deploy.env && ./scripts/remote-manage.sh ps
  NEW_API_HOST=root@your-server NEW_API_PATH=/opt/new-api ./scripts/remote-manage.sh logs [service]
  NEW_API_HOST=root@your-server NEW_API_PATH=/opt/new-api ./scripts/remote-manage.sh restart [service]
  NEW_API_HOST=root@your-server NEW_API_PATH=/opt/new-api ./scripts/remote-manage.sh pull-up
  NEW_API_HOST=root@your-server NEW_API_PATH=/opt/new-api ./scripts/remote-manage.sh down

Services:
  new-api, caddy, postgres, redis
USAGE
}

remote_quote() {
  printf "'%s'" "$(printf '%s' "$1" | sed "s/'/'\\\\''/g")"
}

if [[ -z "${REMOTE_HOST}" || -z "${ACTION}" || "${ACTION}" == "-h" || "${ACTION}" == "--help" ]]; then
  usage
  exit 1
fi

if [[ ! "${SERVICE}" =~ ^[A-Za-z0-9_.-]+$ ]]; then
  echo "Invalid service name: ${SERVICE}" >&2
  exit 1
fi

SSH_ARGS=(-o BatchMode=yes -o ConnectTimeout=20 -o ConnectionAttempts=3 -o ServerAliveInterval=30 -o ServerAliveCountMax=3)
if [[ -n "${SSH_PORT}" ]]; then
  SSH_ARGS+=(-p "${SSH_PORT}")
fi

REMOTE_PATH_Q="$(remote_quote "${REMOTE_PATH}")"
COMPOSE="docker compose --env-file .env.production -f docker-compose.prod.yml"

case "${ACTION}" in
  ps)
    ssh "${SSH_ARGS[@]}" "${REMOTE_HOST}" "cd ${REMOTE_PATH_Q} && ${COMPOSE} ps"
    ;;
  logs)
    ssh "${SSH_ARGS[@]}" "${REMOTE_HOST}" "cd ${REMOTE_PATH_Q} && ${COMPOSE} logs -f --tail=200 ${SERVICE}"
    ;;
  restart)
    ssh "${SSH_ARGS[@]}" "${REMOTE_HOST}" "cd ${REMOTE_PATH_Q} && ${COMPOSE} restart ${SERVICE}"
    ;;
  pull-up)
    ssh "${SSH_ARGS[@]}" "${REMOTE_HOST}" "cd ${REMOTE_PATH_Q} && ${COMPOSE} up -d --build"
    ;;
  down)
    ssh "${SSH_ARGS[@]}" "${REMOTE_HOST}" "cd ${REMOTE_PATH_Q} && ${COMPOSE} down"
    ;;
  *)
    usage
    exit 1
    ;;
esac
