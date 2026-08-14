$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$repositoryRoot = Split-Path -Parent $PSScriptRoot
. (Join-Path $PSScriptRoot "lib/environment.ps1")
Import-ProjectEnvironment -RepositoryRoot $repositoryRoot -Override

if ((Get-RequiredEnvironmentValue -Name "APP_ENV") -ne "local") {
    throw "Comprehensive test data may only be loaded in APP_ENV=local."
}

$lookupKey = [Convert]::FromBase64String((Get-RequiredEnvironmentValue -Name "IDENTITY_PHONE_LOOKUP_KEY_BASE64"))
$phones = [ordered]@{
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
    [Environment]::SetEnvironmentVariable("IDENTITY_BOOTSTRAP_HOSPITAL_CODE", "RH-HOSPITAL", "Process")
    [Environment]::SetEnvironmentVariable("IDENTITY_BOOTSTRAP_HOSPITAL_NAME", "仁和医院", "Process")
    [Environment]::SetEnvironmentVariable("IDENTITY_BOOTSTRAP_ADMIN_PHONES", "13482154556", "Process")
    Push-Location $repositoryRoot
    try {
        & go run ./tools/identity-bootstrap-admin
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

$variables = foreach ($entry in $phones.GetEnumerator()) {
    $hmac = [Security.Cryptography.HMACSHA256]::new($lookupKey)
    try {
        $digest = $hmac.ComputeHash([Text.Encoding]::UTF8.GetBytes($entry.Value))
        $hex = [Convert]::ToHexString($digest)
        "SET @$($entry.Key) = UNHEX('$hex');"
    }
    finally {
        $hmac.Dispose()
    }
}

$seedFile = Join-Path $PSScriptRoot "seed-comprehensive-test-data.sql"
$sql = ($variables -join "`n") + "`n" + (Get-Content -Raw -LiteralPath $seedFile -Encoding UTF8)
$composeFile = Join-Path $repositoryRoot "deploy/compose/docker-compose.yml"
$environmentFile = Join-Path $repositoryRoot ".env"

$sql | & docker compose --env-file $environmentFile -f $composeFile exec -T mysql sh -c 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql --protocol=socket -uroot --default-character-set=utf8mb4'
if ($LASTEXITCODE -ne 0) {
    throw "Failed to load comprehensive test data."
}

Write-Output "Comprehensive local test data loaded."
Write-Output "Local administrator: 13482154556"
Write-Output "Radiology doctor: 13800000001"
Write-Output "Ultrasound doctor: 13800000002"
Write-Output "Report patient: 15363658538"
Write-Output "Other patients: 13900000002 through 13900000004"
