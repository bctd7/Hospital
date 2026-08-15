#!/usr/bin/env bash
set -euo pipefail

# 仅用于已经清空并完成迁移、超级管理员初始化的体验环境。
# 脚本不会清库；检测到已有组织子节点或 Appointment 项目时会拒绝执行。
if [[ "${EXPERIENCE_SEED_ACKNOWLEDGE:-}" != "fresh-experience-database" ]]; then
  echo "Set EXPERIENCE_SEED_ACKNOWLEDGE=fresh-experience-database before loading experience data." >&2
  exit 1
fi

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
deploy_dir="$(cd -- "${script_dir}/.." && pwd)"
repository_root="$(cd -- "${deploy_dir}/../.." && pwd)"
seed_file="${repository_root}/scripts/seed-comprehensive-test-data.sql"

cd "${deploy_dir}"
test -f .env.production || { echo "Missing ${deploy_dir}/.env.production" >&2; exit 1; }
test -f "${seed_file}" || { echo "Missing ${seed_file}" >&2; exit 1; }

compose=(docker compose --env-file .env.production -f docker-compose.yml)
"${compose[@]}" exec -T mysql sh -c 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql --protocol=socket -uroot -N -e "
SELECT IF(
  (SELECT COUNT(*) FROM hospital_identity.identity_organization_units WHERE unit_type = '\''hospital'\'') = 1
  AND (SELECT COUNT(*) FROM hospital_identity.identity_organization_units WHERE unit_type <> '\''hospital'\'') = 0
  AND (SELECT COUNT(*) FROM hospital_identity.identity_account_roles ar JOIN hospital_identity.identity_roles r ON r.id = ar.role_id WHERE r.code = '\''super_admin'\'') = 2
  AND (SELECT COUNT(*) FROM hospital_appointment.appointment_examination_items) = 0,
  '\''ready'\'', '\''refuse'\'');"' | grep -qx ready || {
    echo "Experience seed refused: expected one hospital, two bootstrap administrators, and no business data." >&2
    exit 1
  }

fingerprint() {
  local normalized_phone="$1"
  "${compose[@]}" exec -T mysql sh -c '
    key_hex=$(printf "%s" "$IDENTITY_PHONE_LOOKUP_KEY_BASE64" | base64 -d | od -An -v -tx1 | tr -d " \n")
    printf "%s" "$1" | openssl dgst -sha256 -mac HMAC -macopt "hexkey:${key_hex}" -binary | od -An -v -tx1 | tr -d " \n"
  ' sh "${normalized_phone}"
}

phone_doctor_1="$(fingerprint +8613800000001)"
phone_doctor_2="$(fingerprint +8613800000002)"
phone_patient_1="$(fingerprint +8615363658538)"
phone_patient_2="$(fingerprint +8613900000002)"
phone_patient_3="$(fingerprint +8613900000003)"
phone_patient_4="$(fingerprint +8613900000004)"

{
  printf "SET @phone_doctor_1 = UNHEX('%s');\n" "${phone_doctor_1}"
  printf "SET @phone_doctor_2 = UNHEX('%s');\n" "${phone_doctor_2}"
  printf "SET @phone_patient_1 = UNHEX('%s');\n" "${phone_patient_1}"
  printf "SET @phone_patient_2 = UNHEX('%s');\n" "${phone_patient_2}"
  printf "SET @phone_patient_3 = UNHEX('%s');\n" "${phone_patient_3}"
  printf "SET @phone_patient_4 = UNHEX('%s');\n" "${phone_patient_4}"
  cat "${seed_file}"
} | "${compose[@]}" exec -T mysql sh -c 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql --protocol=socket -uroot --default-character-set=utf8mb4'

"${compose[@]}" exec -T mysql sh -c 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql --protocol=socket -uroot -N -e "
SELECT CONCAT('\''super_admins='\'', COUNT(*)) FROM hospital_identity.identity_account_roles ar JOIN hospital_identity.identity_roles r ON r.id = ar.role_id WHERE r.code = '\''super_admin'\'';
SELECT CONCAT('\''organization_units='\'', COUNT(*)) FROM hospital_identity.identity_organization_units;
SELECT CONCAT('\''rooms='\'', COUNT(*)) FROM hospital_appointment.appointment_rooms;
SELECT CONCAT('\''items='\'', COUNT(*)) FROM hospital_appointment.appointment_examination_items;
SELECT CONCAT('\''bookings='\'', COUNT(*)) FROM hospital_appointment.appointment_bookings;
SELECT CONCAT('\''booking_statuses='\'', GROUP_CONCAT(status, '\''='\'', total ORDER BY status SEPARATOR '\'','\'')) FROM (SELECT status, COUNT(*) total FROM hospital_appointment.appointment_bookings GROUP BY status) statuses;
SELECT CONCAT('\''reports='\'', COUNT(*)) FROM hospital_appointment.appointment_examination_reports;"'

echo "Experience test data loaded. Bootstrap administrator phone numbers were preserved from .env.production."
