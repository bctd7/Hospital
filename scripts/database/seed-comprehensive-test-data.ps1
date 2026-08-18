param(
    [switch]$Reset
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$repositoryRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
. (Join-Path (Split-Path -Parent $PSScriptRoot) "lib/environment.ps1")
Import-ProjectEnvironment -RepositoryRoot $repositoryRoot -Override

if ((Get-RequiredEnvironmentValue -Name "APP_ENV") -ne "local") {
    throw "Comprehensive test data may only be loaded in APP_ENV=local."
}

$composeFile = Join-Path $repositoryRoot "deploy/compose/docker-compose.yml"
$environmentFile = Join-Path $repositoryRoot ".env"

if ($Reset) {
    & docker compose --env-file $environmentFile -f $composeFile up -d mysql redis
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to start MySQL before resetting comprehensive test data."
    }
    $ready = $false
    for ($attempt = 1; $attempt -le 30; $attempt++) {
        & docker compose --env-file $environmentFile -f $composeFile exec -T mysql sh -c 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysqladmin ping --protocol=socket -uroot --silent' *> $null
        if ($LASTEXITCODE -eq 0) {
            $ready = $true
            break
        }
        Start-Sleep -Seconds 2
    }
    if (-not $ready) {
        throw "MySQL did not become ready before resetting comprehensive test data."
    }
    $redisReady = $false
    for ($attempt = 1; $attempt -le 30; $attempt++) {
        & docker compose --env-file $environmentFile -f $composeFile exec -T redis sh -c 'redis-cli -a "$REDIS_PASSWORD" ping 2>/dev/null' *> $null
        if ($LASTEXITCODE -eq 0) {
            $redisReady = $true
            break
        }
        Start-Sleep -Seconds 1
    }
    if (-not $redisReady) {
        throw "Redis did not become ready before resetting comprehensive test data."
    }
    & docker compose --env-file $environmentFile -f $composeFile exec -T redis sh -c 'redis-cli -a "$REDIS_PASSWORD" FLUSHDB 2>/dev/null' | Out-Null
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to clear local sessions, authorization versions, and Appointment caches."
    }
    $dropSQL = @"
DROP DATABASE IF EXISTS hospital_appointment;
DROP DATABASE IF EXISTS hospital_identity;
DROP DATABASE IF EXISTS hospital_guidance;
"@
    $dropSQL | & docker compose --env-file $environmentFile -f $composeFile exec -T mysql sh -c 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql --protocol=socket -uroot'
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to reset local Identity, Appointment, and Guidance databases."
    }
    & (Join-Path $PSScriptRoot "bootstrap-local.ps1")
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to rebuild the latest local database schema."
    }
}

$lookupKey = [Convert]::FromBase64String((Get-RequiredEnvironmentValue -Name "IDENTITY_PHONE_LOOKUP_KEY_BASE64"))
$phones = [ordered]@{
    phone_guidance_admin = "+8613482154556"
    phone_doctor_1 = "+8613800000001"
    phone_doctor_2 = "+8613800000002"
    phone_patient_1 = "+8615363658538"
    phone_patient_2 = "+8613900000002"
    phone_patient_3 = "+8613900000003"
    phone_patient_4 = "+8613900000004"
}

# 综合业务数据复用部署初始化生成的医院根节点和超级管理员。
# 本地空库没有部署任务，因此先用同一个正式工具创建一个测试管理员；服务器则直接复用 Secret 中的两个号码。
$previousHospitalCode = [Environment]::GetEnvironmentVariable("IDENTITY_BOOTSTRAP_HOSPITAL_CODE", "Process")
$previousHospitalName = [Environment]::GetEnvironmentVariable("IDENTITY_BOOTSTRAP_HOSPITAL_NAME", "Process")
$previousAdminPhones = [Environment]::GetEnvironmentVariable("IDENTITY_BOOTSTRAP_ADMIN_PHONES", "Process")
try {
    [Environment]::SetEnvironmentVariable("IDENTITY_BOOTSTRAP_HOSPITAL_CODE", "SH-SECOND-PEOPLES-HOSPITAL", "Process")
    [Environment]::SetEnvironmentVariable("IDENTITY_BOOTSTRAP_HOSPITAL_NAME", "上海市第二人民医院", "Process")
    [Environment]::SetEnvironmentVariable("IDENTITY_BOOTSTRAP_ADMIN_PHONES", "13482154556,15363658538", "Process")
    Push-Location $repositoryRoot
    try {
        & go run ./tools/identity/bootstrap-admin
        if ($LASTEXITCODE -ne 0) {
            throw "Failed to bootstrap the local comprehensive-test administrator."
        }
    }
    finally {
        Pop-Location
    }
}
finally {
    [Environment]::SetEnvironmentVariable("IDENTITY_BOOTSTRAP_HOSPITAL_CODE", $previousHospitalCode, "Process")
    [Environment]::SetEnvironmentVariable("IDENTITY_BOOTSTRAP_HOSPITAL_NAME", $previousHospitalName, "Process")
    [Environment]::SetEnvironmentVariable("IDENTITY_BOOTSTRAP_ADMIN_PHONES", $previousAdminPhones, "Process")
}

$fingerprints = @{}
$variables = foreach ($entry in $phones.GetEnumerator()) {
    $hmac = [Security.Cryptography.HMACSHA256]::new($lookupKey)
    try {
        $digest = $hmac.ComputeHash([Text.Encoding]::UTF8.GetBytes($entry.Value))
        $hex = [Convert]::ToHexString($digest)
        $fingerprints[$entry.Key] = $hex
        "SET @$($entry.Key) = UNHEX('$hex');"
    }
    finally {
        $hmac.Dispose()
    }
}

$seedFile = Join-Path $PSScriptRoot "seed-comprehensive-test-data.sql"
$patientOneSnapshotVariables = @(
    "SET @phone_guidance_admin_masked = '134****4556';"
    "SET @phone_guidance_admin_last4 = '4556';"
    "SET @phone_patient_1_masked = '153****8538';"
    "SET @phone_patient_1_last4 = '8538';"
)
$sql = (($variables + $patientOneSnapshotVariables) -join "`n") + "`n" + (Get-Content -Raw -LiteralPath $seedFile -Encoding UTF8)
$sql | & docker compose --env-file $environmentFile -f $composeFile exec -T mysql sh -c 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql --protocol=socket -uroot --default-character-set=utf8mb4'
if ($LASTEXITCODE -ne 0) {
    throw "Failed to load comprehensive test data."
}

$verificationSQL = @"
SET @phone_patient_1 = UNHEX('$($fingerprints.phone_patient_1)');
SET @phone_guidance_admin = UNHEX('$($fingerprints.phone_guidance_admin)');
SET @hospital_local_today = DATE(CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+08:00'));
SELECT IF(
  (SELECT COUNT(*) FROM hospital_identity.identity_organization_units WHERE unit_type = 'campus' AND status = 'active') >= 2
  AND (SELECT COUNT(*) FROM hospital_identity.identity_organization_units WHERE unit_type = 'department' AND status = 'active') >= 4
  AND (SELECT COUNT(*) FROM hospital_identity.identity_staff_profiles WHERE staff_status = 'active') >= 4
  -- 153 管理员以真实超级管理员账号兼任报告患者，因此 account_type 不伪装成 patient。
  AND (SELECT COUNT(*) FROM hospital_identity.identity_accounts WHERE account_type = 'patient' AND status = 'active') >= 10
  AND (SELECT COUNT(*) FROM hospital_appointment.appointment_bookings) = 34
  AND (SELECT COUNT(DISTINCT status) FROM hospital_appointment.appointment_bookings) = 8
  AND (SELECT COUNT(*) FROM hospital_appointment.appointment_examination_items
       WHERE estimated_duration_minutes BETWEEN 5 AND 480 AND MOD(estimated_duration_minutes, 5) = 0) = 11
  AND (SELECT COUNT(*) FROM hospital_appointment.appointment_bookings
       WHERE estimated_duration_minutes_snapshot BETWEEN 5 AND 480 AND MOD(estimated_duration_minutes_snapshot, 5) = 0) = 34
  AND (SELECT COUNT(*) FROM hospital_appointment.appointment_bookings WHERE status = 'canceled') >= 1
  AND (SELECT COUNT(*) FROM hospital_appointment.appointment_bookings WHERE status = 'report_pending' AND service_date < @hospital_local_today) >= 1
  AND (SELECT COUNT(DISTINCT department_id) FROM hospital_appointment.appointment_bookings WHERE service_date = @hospital_local_today) = 4
  AND (SELECT COUNT(*) FROM hospital_appointment.appointment_check_queues) >= 5
  AND (SELECT COUNT(*) FROM hospital_appointment.appointment_check_queue_entries) >= 10
  AND (SELECT COUNT(*) FROM hospital_appointment.appointment_check_queue_events WHERE event_type = 'called') >= 4
  AND (SELECT COUNT(*) FROM hospital_appointment.appointment_bookings
       WHERE id = '40000000-0000-4000-8000-000000000007'
         AND started_at = CONVERT_TZ(TIMESTAMP(DATE_SUB(@hospital_local_today, INTERVAL 1 DAY), '09:15:00'), '+08:00', '+00:00')) = 1
  AND (SELECT COUNT(*) FROM hospital_appointment.appointment_examination_report_versions WHERE version_kind = 'correction' AND status = 'published') >= 1
  AND (SELECT COUNT(*) FROM hospital_appointment.appointment_examination_reports) >= 8
  AND (SELECT COUNT(*) FROM hospital_appointment.appointment_examination_reports WHERE status = 'published') >= 6
  AND (SELECT COUNT(*) FROM hospital_appointment.appointment_bookings b
       LEFT JOIN hospital_appointment.appointment_examination_reports r ON r.booking_id = b.id
       WHERE b.status = 'report_pending' AND r.id IS NULL) >= 1
  AND (SELECT COUNT(*)
       FROM hospital_identity.identity_account_phones p
       JOIN hospital_identity.identity_account_roles ar ON ar.account_id = p.account_id
       JOIN hospital_identity.identity_roles r ON r.id = ar.role_id AND r.code = 'super_admin'
       JOIN hospital_appointment.appointment_examination_reports report ON report.patient_account_id = p.account_id
       WHERE p.phone_fingerprint = @phone_patient_1 AND report.status = 'published') >= 4
  AND (SELECT COUNT(*) FROM hospital_appointment.appointment_message_reads) >= 1
  AND (SELECT COUNT(*) FROM hospital_guidance.guidance_item_configurations) = 11
  AND (SELECT COUNT(*) FROM hospital_guidance.guidance_precedence_rules
       WHERE predecessor_department_id <> successor_department_id) >= 1
  AND (SELECT COUNT(*)
       FROM hospital_identity.identity_account_phones p
       JOIN hospital_identity.identity_account_roles ar ON ar.account_id = p.account_id
       JOIN hospital_identity.identity_roles r ON r.id = ar.role_id AND r.code = 'super_admin'
       WHERE p.phone_fingerprint = @phone_guidance_admin) = 1
  AND (SELECT COUNT(*)
       FROM hospital_appointment.appointment_bookings b
       JOIN hospital_identity.identity_account_phones p ON p.account_id = b.patient_account_id
       WHERE p.phone_fingerprint = @phone_guidance_admin
         AND b.service_date = @hospital_local_today
         AND b.status = 'confirmed') = 3,
  'ready', 'incomplete');
"@
$verification = $verificationSQL | & docker compose --env-file $environmentFile -f $composeFile exec -T mysql sh -c 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql --protocol=socket -uroot -N'
if ($LASTEXITCODE -ne 0 -or ($verification | Select-Object -Last 1).Trim() -ne "ready") {
    throw "Comprehensive test data verification failed."
}

Write-Output "Comprehensive local test data loaded."
Write-Output "Reset mode: $Reset"
Write-Output "Local administrators: 13482154556 and 15363658538"
Write-Output "Radiology doctor: 13800000001"
Write-Output "Ultrasound doctor: 13800000002"
Write-Output "Report patient (also a super administrator): 15363658538"
Write-Output "Login-capable patients: 13900000002 through 13900000004"
Write-Output "Display-only patients: 张伟、刘洋、陈静、孙磊、周婷、吴昊、林悦"
Write-Output "Guidance patient: 13482154556 (today: 血常规 -> 冠状动脉CTA -> 泌尿系彩超)"
Write-Output "Coverage: 11 examination items across 1/2/3号楼, 34 bookings, 8 statuses, 5 room queues, 8 reports"
