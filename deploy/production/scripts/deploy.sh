#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
deploy_dir="$(cd -- "${script_dir}/.." && pwd)"

cd "${deploy_dir}"

test -f .env.production || {
  echo "Missing ${deploy_dir}/.env.production" >&2
  exit 1
}

compose=(docker compose --env-file .env.production -f docker-compose.yml)
"${compose[@]}" config --quiet

if [[ "${HOSPITAL_PRELOADED_IMAGES:-0}" == "1" ]]; then
  "${compose[@]}" up -d --no-build --pull never --remove-orphans
else
  "${compose[@]}" up -d --build --remove-orphans
fi

for attempt in $(seq 1 30); do
  if curl --fail --silent --show-error http://127.0.0.1:8888/api/v1/health >/dev/null; then
    "${compose[@]}" ps
    echo "Deployment is healthy."
    exit 0
  fi
  sleep 2
done

"${compose[@]}" ps
"${compose[@]}" logs --tail=100 app-api identity-rpc
echo "Deployment did not become healthy in time." >&2
exit 1
