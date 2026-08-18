#!/usr/bin/env bash
set -euo pipefail
umask 077

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
deploy_dir="$(cd -- "${script_dir}/.." && pwd)"
backup_dir="${BACKUP_DIR:-${deploy_dir}/backups}"
timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
target="${backup_dir}/hospital-${timestamp}.sql.gz"

cd "${deploy_dir}"
test -f .env.production || {
  echo "Missing ${deploy_dir}/.env.production" >&2
  exit 1
}

mkdir -p "${backup_dir}"

docker compose --env-file .env.production -f docker-compose.yml exec -T mysql \
  sh -c 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" exec mysqldump -uroot --single-transaction --routines --triggers --events --databases hospital_identity hospital_appointment hospital_guidance' \
  | gzip -9 >"${target}"

gzip -t "${target}"
echo "Backup created: ${target}"
