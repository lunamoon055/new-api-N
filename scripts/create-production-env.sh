#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${ROOT_DIR}/.env.production"
DEPLOY_CONFIG="${ROOT_DIR}/.deploy.env"

if [[ -f "${DEPLOY_CONFIG}" ]]; then
  set -a
  # shellcheck source=/dev/null
  source "${DEPLOY_CONFIG}"
  set +a
fi

PUBLIC_URL="${1:-${PUBLIC_URL:-}}"

usage() {
  cat <<'USAGE'
Usage:
  ./scripts/create-production-env.sh https://your-domain.example
  ./scripts/create-production-env.sh http://your-server-ip
  PUBLIC_URL=https://your-domain.example ./scripts/create-production-env.sh

This creates .env.production with random production secrets.
USAGE
}

random_secret() {
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -hex 32
  else
    od -An -N32 -tx1 /dev/urandom | tr -d ' \n'
    printf '\n'
  fi
}

strip_scheme() {
  local value="$1"
  value="${value#http://}"
  value="${value#https://}"
  value="${value%%/*}"
  printf '%s\n' "$value"
}

url_scheme() {
  local value="$1"
  case "${value}" in
    http://*) printf 'http\n' ;;
    https://*) printf 'https\n' ;;
    *) printf '\n' ;;
  esac
}

is_ipv4_host() {
  local value="$1"
  [[ "${value}" =~ ^([0-9]{1,3}\.){3}[0-9]{1,3}(:[0-9]+)?$ ]]
}

if [[ -z "${PUBLIC_URL}" || "${PUBLIC_URL}" == "-h" || "${PUBLIC_URL}" == "--help" ]]; then
  usage
  exit 1
fi

if [[ -f "${ENV_FILE}" && "${FORCE:-0}" != "1" ]]; then
  echo ".env.production already exists. Set FORCE=1 to overwrite it." >&2
  exit 1
fi

PUBLIC_HOST="$(strip_scheme "${PUBLIC_URL}")"
if [[ -z "${PUBLIC_HOST}" ]]; then
  echo "Could not derive PUBLIC_HOST from: ${PUBLIC_URL}" >&2
  exit 1
fi

PUBLIC_SCHEME="$(url_scheme "${PUBLIC_URL}")"

if [[ "${PUBLIC_HOST}" == *:* && "${PUBLIC_HOST}" != \[*\] ]]; then
  TRUSTED_DOMAIN="${PUBLIC_HOST%%:*}"
else
  TRUSTED_DOMAIN="${PUBLIC_HOST}"
fi

if [[ "${PUBLIC_SCHEME}" == "http" && "$(is_ipv4_host "${PUBLIC_HOST}" && printf yes)" == "yes" ]]; then
  PUBLIC_HOST=":80"
fi

cat >"${ENV_FILE}" <<EOF
PUBLIC_HOST=${PUBLIC_HOST}
TRUSTED_REDIRECT_DOMAINS=${TRUSTED_DOMAIN}

POSTGRES_USER=newapi
POSTGRES_PASSWORD=$(random_secret)
POSTGRES_DB=new_api

REDIS_PASSWORD=$(random_secret)
SESSION_SECRET=$(random_secret)
CRYPTO_SECRET=$(random_secret)

TZ=Asia/Shanghai
NODE_NAME=new-api-prod-1
EOF

chmod 600 "${ENV_FILE}"
echo "Created ${ENV_FILE}"
echo "Review it before deploying, especially PUBLIC_HOST and TRUSTED_REDIRECT_DOMAINS."
