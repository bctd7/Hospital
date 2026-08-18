#!/usr/bin/env bash
set -euo pipefail

# 仅用于已经清空并完成迁移、超级管理员初始化的体验环境。
# 脚本不会清库；检测到已有组织子节点或业务数据时会拒绝执行。
if [[ "${EXPERIENCE_SEED_ACKNOWLEDGE:-}" != "fresh-experience-database" ]]; then
  echo "Set EXPERIENCE_SEED_ACKNOWLEDGE=fresh-experience-database before loading experience data." >&2
  exit 1
fi

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
deploy_dir="$(cd -- "${script_dir}/.." && pwd)"
repository_root="$(cd -- "${deploy_dir}/../.." && pwd)"
seed_file="${repository_root}/scripts/database/seed-comprehensive-test-data.sql"

cd "${deploy_dir}"
test -f .env.production || { echo "Missing ${deploy_dir}/.env.production" >&2; exit 1; }
test -f "${seed_file}" || { echo "Missing ${seed_file}" >&2; exit 1; }

compose=(docker compose --env-file .env.production -f docker-compose.yml)

fingerprint() {
  local normalized_phone="$1"
  "${compose[@]}" exec -T mysql sh -c '
    key_hex=$(printf "%s" "$IDENTITY_PHONE_LOOKUP_KEY_BASE64" | base64 -d | od -An -v -tx1 | tr -d " \n")
    printf "%s" "$1" | openssl dgst -sha256 -mac HMAC -macopt "hexkey:${key_hex}" -binary | od -An -v -tx1 | tr -d " \n"
  ' sh "${normalized_phone}"
}

admin_phones_value="$(docker inspect hospital-production-identity-bootstrap-admin-1 --format '{{range .Config.Env}}{{println .}}{{end}}' | sed -n 's/^IDENTITY_BOOTSTRAP_ADMIN_PHONES=//p')"
real_doctor_phone="${EXPERIENCE_REAL_DOCTOR_PHONE:-}"
real_doctor_phone="${real_doctor_phone//[!0-9]/}"
[[ "${real_doctor_phone}" =~ ^1[3-9][0-9]{9}$ ]] || {
  echo "Experience seed refused: EXPERIENCE_REAL_DOCTOR_PHONE must be one mainland China mobile number." >&2
  exit 1
}

IFS=',' read -r -a configured_admin_phones <<<"${admin_phones_value}"
[[ "${#configured_admin_phones[@]}" -eq 2 ]] || {
  echo "Experience seed refused: exactly two bootstrap administrators are required." >&2
  exit 1
}
normalized_admin_phones=()
for configured_phone in "${configured_admin_phones[@]}"; do
  phone_digits="${configured_phone//[!0-9]/}"
  [[ "${phone_digits}" =~ ^86(1[3-9][0-9]{9})$ ]] && phone_digits="${BASH_REMATCH[1]}"
  [[ "${phone_digits}" =~ ^1[3-9][0-9]{9}$ ]] || {
    echo "Experience seed refused: invalid bootstrap administrator phone." >&2
    exit 1
  }
  [[ "${phone_digits}" != "${real_doctor_phone}" ]] || {
    echo "Experience seed refused: the real doctor must not also be a super administrator." >&2
    exit 1
  }
  normalized_admin_phones+=("${phone_digits}")
done

phone_doctor_1="$(fingerprint +8613800000001)"
phone_doctor_2="$(fingerprint +8613800000002)"
phone_guidance_admin="$(fingerprint "+86${normalized_admin_phones[0]}")"
phone_second_admin="$(fingerprint "+86${normalized_admin_phones[1]}")"
phone_patient_1="$(fingerprint "+86${real_doctor_phone}")"
phone_patient_2="$(fingerprint +8613900000002)"
phone_patient_3="$(fingerprint +8613900000003)"
phone_patient_4="$(fingerprint +8613900000004)"

"${compose[@]}" exec -T mysql sh -c "MYSQL_PWD=\"\$MYSQL_ROOT_PASSWORD\" mysql --protocol=socket -uroot -N -e \"
SELECT IF(
  (SELECT COUNT(*) FROM hospital_identity.identity_organization_units WHERE unit_type = 'hospital') = 1
  AND (SELECT COUNT(*) FROM hospital_identity.identity_organization_units WHERE unit_type <> 'hospital') = 0
  AND (SELECT COUNT(*) FROM hospital_identity.identity_account_roles ar JOIN hospital_identity.identity_roles r ON r.id = ar.role_id WHERE r.code = 'super_admin') = 2
  AND (SELECT COUNT(*) FROM hospital_identity.identity_account_phones p WHERE p.phone_fingerprint = UNHEX('${phone_patient_1}')) = 0
  AND (SELECT COUNT(*) FROM hospital_appointment.appointment_examination_items) = 0
  AND (SELECT COUNT(*) FROM hospital_guidance.guidance_item_configurations) = 0,
  'ready', 'refuse');\"" | grep -qx ready || {
    echo "Experience seed refused: expected a fresh database, two bootstrap administrators, and no existing real-doctor account." >&2
    exit 1
  }

{
  printf "SET @phone_doctor_1 = UNHEX('%s');\n" "${phone_doctor_1}"
  printf "SET @phone_doctor_2 = UNHEX('%s');\n" "${phone_doctor_2}"
  printf "SET @phone_guidance_admin = UNHEX('%s');\n" "${phone_guidance_admin}"
  printf "SET @phone_second_admin = UNHEX('%s');\n" "${phone_second_admin}"
  printf "SET @phone_patient_1 = UNHEX('%s');\n" "${phone_patient_1}"
  printf "SET @phone_guidance_admin_masked = '%s';\n" "${normalized_admin_phones[0]:0:3}****${normalized_admin_phones[0]:7:4}"
  printf "SET @phone_guidance_admin_last4 = '%s';\n" "${normalized_admin_phones[0]:7:4}"
  printf "SET @phone_patient_1_masked = '%s';\n" "${real_doctor_phone:0:3}****${real_doctor_phone:7:4}"
  printf "SET @phone_patient_1_last4 = '%s';\n" "${real_doctor_phone:7:4}"
  printf "SET @phone_patient_2 = UNHEX('%s');\n" "${phone_patient_2}"
  printf "SET @phone_patient_3 = UNHEX('%s');\n" "${phone_patient_3}"
  printf "SET @phone_patient_4 = UNHEX('%s');\n" "${phone_patient_4}"
  cat "${seed_file}"
} | "${compose[@]}" exec -T mysql sh -c 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql --protocol=socket -uroot --default-character-set=utf8mb4'

"${compose[@]}" exec -T mysql sh -c "MYSQL_PWD=\"\$MYSQL_ROOT_PASSWORD\" mysql --protocol=socket -uroot -N -e \"
SELECT IF(
  (SELECT COUNT(*) FROM hospital_identity.identity_account_roles ar JOIN hospital_identity.identity_roles r ON r.id = ar.role_id WHERE r.code = 'super_admin') = 2
  AND (SELECT COUNT(*) FROM hospital_identity.identity_account_phones p
       JOIN hospital_identity.identity_account_roles ar ON ar.account_id = p.account_id
       JOIN hospital_identity.identity_roles r ON r.id = ar.role_id AND r.code = 'department_doctor'
       JOIN hospital_identity.identity_staff_profiles s ON s.account_id = p.account_id AND s.staff_status = 'active'
       WHERE p.phone_fingerprint = UNHEX('${phone_patient_1}')) = 1
  AND (SELECT COUNT(*) FROM hospital_identity.identity_account_phones p
       JOIN hospital_identity.identity_account_roles ar ON ar.account_id = p.account_id
       JOIN hospital_identity.identity_roles r ON r.id = ar.role_id AND r.code = 'super_admin'
       WHERE p.phone_fingerprint = UNHEX('${phone_patient_1}')) = 0
  AND (SELECT COUNT(*) FROM hospital_appointment.appointment_examination_items) >= 21
  AND (SELECT COUNT(DISTINCT status) FROM hospital_appointment.appointment_bookings) = 8
  AND (SELECT COUNT(*) FROM hospital_appointment.appointment_examination_reports report
       JOIN hospital_identity.identity_account_phones p ON p.account_id = report.patient_account_id
       WHERE p.phone_fingerprint = UNHEX('${phone_patient_1}') AND report.status = 'published') >= 4
  AND (SELECT COUNT(*) FROM hospital_guidance.guidance_item_configurations) >= 21
  AND (SELECT COUNT(*) FROM hospital_guidance.guidance_precedence_rules WHERE predecessor_department_id <> successor_department_id) >= 1,
  'ready', 'incomplete');\"" | grep -qx ready || {
    echo "Experience seed verification failed." >&2
    exit 1
  }

"${compose[@]}" exec -T mysql sh -c 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql --protocol=socket -uroot -N -e "
SELECT CONCAT('\''super_admins='\'', COUNT(*)) FROM hospital_identity.identity_account_roles ar JOIN hospital_identity.identity_roles r ON r.id = ar.role_id WHERE r.code = '\''super_admin'\'';
SELECT CONCAT('\''organization_units='\'', COUNT(*)) FROM hospital_identity.identity_organization_units;
SELECT CONCAT('\''rooms='\'', COUNT(*)) FROM hospital_appointment.appointment_rooms;
SELECT CONCAT('\''items='\'', COUNT(*)) FROM hospital_appointment.appointment_examination_items;
SELECT CONCAT('\''bookings='\'', COUNT(*)) FROM hospital_appointment.appointment_bookings;
SELECT CONCAT('\''reports='\'', COUNT(*)) FROM hospital_appointment.appointment_examination_reports;
SELECT CONCAT('\''guidance_configurations='\'', COUNT(*)) FROM hospital_guidance.guidance_item_configurations;"'

echo "Experience test data loaded. Two real super administrators and one ordinary real doctor are ready; the doctor also owns patient-side demonstration records."
