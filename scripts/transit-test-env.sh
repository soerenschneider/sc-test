#!/usr/bin/env bash
#
# Spins up a disposable OpenBao dev server in Docker with a couple of transit
# mounts and keys to test 'sc vault transit' subcommands against.
#
# Usage:
#   scripts/transit-test-env.sh [up|down]
#
# Environment:
#   OPENBAO_IMAGE      container image to use (default: quay.io/openbao/openbao:latest)
#   OPENBAO_PORT       host port to expose OpenBao on (default: 8200)
#   OPENBAO_CONTAINER  container name (default: sc-transit-test)

set -euo pipefail

IMAGE="${OPENBAO_IMAGE:-quay.io/openbao/openbao:latest}"
PORT="${OPENBAO_PORT:-8200}"
CONTAINER="${OPENBAO_CONTAINER:-sc-transit-test}"
ROOT_TOKEN="root"

# "mount key-name:type[:derived] ..." per line
MOUNTS="
transit app:aes256-gcm96 db:chacha20-poly1305 tenant:aes256-gcm96:derived
transit-team-a backup:aes256-gcm96 legacy:aes128-gcm96 rsa:rsa-4096
transit-team-b signing:ed25519 payments:aes256-gcm96
"

bao() {
  docker exec ${BAO_EXEC_FLAGS:-} \
    -e BAO_ADDR="http://127.0.0.1:8200" \
    -e BAO_TOKEN="${ROOT_TOKEN}" \
    "${CONTAINER}" bao "$@"
}

down() {
  docker rm -f "${CONTAINER}" >/dev/null 2>&1 || true
  echo "Removed container ${CONTAINER}"
}

up() {
  down

  echo "Starting ${IMAGE} as ${CONTAINER} on port ${PORT}..."
  docker run -d --rm \
    --name "${CONTAINER}" \
    --cap-add IPC_LOCK \
    -p "127.0.0.1:${PORT}:8200" \
    "${IMAGE}" \
    server -dev -dev-root-token-id="${ROOT_TOKEN}" -dev-listen-address=0.0.0.0:8200 >/dev/null

  echo -n "Waiting for OpenBao to become ready"
  for _ in $(seq 1 30); do
    if bao status >/dev/null 2>&1; then
      echo " done"
      break
    fi
    echo -n "."
    sleep 1
  done
  bao status >/dev/null || { echo " failed"; docker logs "${CONTAINER}"; exit 1; }

  while read -r mount keys; do
    [[ -z "${mount}" ]] && continue
    bao secrets enable -path="${mount}" transit >/dev/null
    echo "Enabled transit mount ${mount}"
    for spec in ${keys}; do
      IFS=: read -r name type derived <<<"${spec}"
      args=(type="${type}")
      [[ "${derived:-}" == "derived" ]] && args+=(derived=true)
      bao write -f "${mount}/keys/${name}" "${args[@]}" >/dev/null
      echo "  created key ${name} (${type}${derived:+, derived})"
    done
  done <<<"${MOUNTS}"

  # A token that can use transit but can not list sys/mounts, to test the
  # fallback to manual mount input.
  BAO_EXEC_FLAGS=-i bao policy write transit-only - >/dev/null <<'POLICY'
path "+/keys" {
  capabilities = ["list"]
}
path "+/encrypt/*" {
  capabilities = ["update"]
}
path "+/decrypt/*" {
  capabilities = ["update"]
}
POLICY
  limited_token="$(bao token create -policy=transit-only -field=token)"

  cat <<INFO

OpenBao is ready. Use it with:

  export VAULT_ADDR=http://127.0.0.1:${PORT}
  export VAULT_TOKEN=${ROOT_TOKEN}

Token without access to sys/mounts (tests manual mount input):

  export VAULT_TOKEN=${limited_token}

Notes:
  - transit/tenant is derived and needs --context
  - transit-team-b/signing is ed25519 and does not support encryption

Tear down with: $0 down
INFO
}

case "${1:-up}" in
  up) up ;;
  down) down ;;
  *) echo "usage: $0 [up|down]" >&2; exit 1 ;;
esac
