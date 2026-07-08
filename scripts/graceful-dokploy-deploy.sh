#!/usr/bin/env bash
set -euo pipefail

API_ORIGIN="${API_ORIGIN:-https://camp.sitcon.party}"
DEPLOY_COMMAND="${DOKPLOY_DEPLOY_COMMAND:-docker compose up -d --build --remove-orphans}"
POLL_INTERVAL_SECONDS="${MAINTENANCE_POLL_INTERVAL_SECONDS:-5}"
TIMEOUT_SECONDS="${MAINTENANCE_TIMEOUT_SECONDS:-900}"
MAINTENANCE_MESSAGE="${MAINTENANCE_MESSAGE:-系統正在更新，請完成目前對戰並暫停新的操作。}"

if [[ -z "${ADMIN_PASSWORD:-}" ]]; then
  echo "ADMIN_PASSWORD is required" >&2
  exit 1
fi

cookie_jar="$(mktemp)"
settings_file="$(mktemp)"
body_file="$(mktemp)"
cleanup() {
  rm -f "$cookie_jar" "$settings_file" "$body_file"
}
trap cleanup EXIT

json_string() {
  python3 -c 'import json,sys; print(json.dumps(sys.argv[1]))' "$1"
}

admin_login() {
  local password_json
  password_json="$(json_string "$ADMIN_PASSWORD")"
  curl -fsS -c "$cookie_jar" \
    -H "Content-Type: application/json" \
    -d "{\"password\":${password_json}}" \
    "$API_ORIGIN/api/admin/login" >/dev/null
}

set_maintenance_mode() {
  local mode="$1"
  local message="$2"

  curl -fsS -b "$cookie_jar" "$API_ORIGIN/api/admin/settings" >"$settings_file"
  python3 - "$mode" "$message" "$settings_file" >"$body_file" <<'PY'
import json
import sys

mode, message, settings_path = sys.argv[1], sys.argv[2], sys.argv[3]
with open(settings_path, encoding="utf-8") as fh:
    settings = json.load(fh)

settings["maintenanceMode"] = mode
settings["maintenanceMessage"] = message if mode != "off" else ""
settings.pop("battleOpeningLocked", None)
settings.pop("maintenanceActive", None)
print(json.dumps(settings, ensure_ascii=False))
PY

  curl -fsS -b "$cookie_jar" \
    -H "Content-Type: application/json" \
    -X PUT \
    --data-binary "@$body_file" \
    "$API_ORIGIN/api/admin/settings" >/dev/null
}

active_match_count() {
  curl -fsS "$API_ORIGIN/api/maintenance" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("activeMatchCount", 0))'
}

maintenance_api_available() {
  curl -fsS "$API_ORIGIN/api/maintenance" >/dev/null
}

wait_for_active_matches() {
  local started
  started="$(date +%s)"

  while true; do
    local count
    count="$(active_match_count)"
    if [[ "$count" == "0" ]]; then
      return
    fi

    local now
    now="$(date +%s)"
    if (( now - started >= TIMEOUT_SECONDS )); then
      echo "timed out waiting for active matches to finish; activeMatchCount=$count" >&2
      exit 1
    fi

    echo "waiting for active matches to finish; activeMatchCount=$count"
    sleep "$POLL_INTERVAL_SECONDS"
  done
}

wait_for_maintenance_api() {
  local started
  started="$(date +%s)"

  while true; do
    if maintenance_api_available; then
      return
    fi

    local now
    now="$(date +%s)"
    if (( now - started >= TIMEOUT_SECONDS )); then
      echo "timed out waiting for maintenance API after deploy" >&2
      exit 1
    fi

    echo "waiting for maintenance API"
    sleep "$POLL_INTERVAL_SECONDS"
  done
}

if ! maintenance_api_available; then
  echo "maintenance API is unavailable; running deploy command directly"
  exec sh -c "$DEPLOY_COMMAND"
fi

admin_login
set_maintenance_mode "draining" "$MAINTENANCE_MESSAGE"
wait_for_active_matches
set_maintenance_mode "active" "$MAINTENANCE_MESSAGE"

deploy_status=0
sh -c "$DEPLOY_COMMAND" || deploy_status=$?

wait_for_maintenance_api
admin_login
set_maintenance_mode "off" ""
exit "$deploy_status"
