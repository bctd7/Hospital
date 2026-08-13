param(
    [switch]$Restart
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$repositoryRoot = Split-Path -Parent $PSScriptRoot
. (Join-Path $PSScriptRoot "lib/environment.ps1")

# Override inherited values so a stale or incorrectly decoded shell environment
# cannot silently replace the UTF-8 values from the project .env file.
Import-ProjectEnvironment -RepositoryRoot $repositoryRoot -Override

$runtimeRoot = Join-Path ([IO.Path]::GetTempPath()) "hospital-backend"
New-Item -ItemType Directory -Path $runtimeRoot -Force | Out-Null

$services = @(
    @{
        Name = "identity-rpc"
        ProcessName = "identity-rpc"
        Port = 8080
        Package = "./service/identity/rpc"
        Binary = Join-Path $runtimeRoot "identity-rpc.exe"
        Config = Join-Path $repositoryRoot "service/identity/rpc/etc/identity-rpc.yaml"
    },
    @{
        Name = "appointment-rpc"
        ProcessName = "appointment-rpc"
        Port = 8081
        Package = "./service/appointment/rpc"
        Binary = Join-Path $runtimeRoot "appointment-rpc.exe"
        Config = Join-Path $repositoryRoot "service/appointment/rpc/etc/appointment.yaml"
    },
    @{
        Name = "app-api"
        ProcessName = "app-api"
        Port = 8888
        Package = "./service/app/api"
        Binary = Join-Path $runtimeRoot "app-api.exe"
        Config = Join-Path $repositoryRoot "service/app/api/etc/app-api.yaml"
    }
)

foreach ($service in $services) {
    $listeners = @(Get-NetTCPConnection -LocalPort $service.Port -State Listen -ErrorAction SilentlyContinue)
    if ($listeners.Count -eq 0) {
        continue
    }
    if (-not $Restart) {
        throw "$($service.Name) is already listening on port $($service.Port). Use -Restart to replace it."
    }
    foreach ($listener in $listeners) {
        $process = Get-Process -Id $listener.OwningProcess -ErrorAction Stop
        if ($process.ProcessName -notin @($service.ProcessName, "rpc", "api")) {
            throw "Port $($service.Port) belongs to unexpected process $($process.ProcessName); refusing to stop it."
        }
        Stop-Process -Id $process.Id -Force
    }
}

foreach ($service in $services) {
    Push-Location $repositoryRoot
    try {
        & go build -o $service.Binary $service.Package
        if ($LASTEXITCODE -ne 0) {
            throw "Failed to build $($service.Name)."
        }
    }
    finally {
        Pop-Location
    }

    $stdout = Join-Path $runtimeRoot "$($service.Name).stdout.log"
    $stderr = Join-Path $runtimeRoot "$($service.Name).stderr.log"
    $process = Start-Process `
        -FilePath $service.Binary `
        -ArgumentList @("-f", $service.Config) `
        -WorkingDirectory $repositoryRoot `
        -WindowStyle Hidden `
        -RedirectStandardOutput $stdout `
        -RedirectStandardError $stderr `
        -PassThru
    $service.Process = $process
}

$deadline = (Get-Date).AddSeconds(30)
do {
    Start-Sleep -Milliseconds 400
    $ready = $true
    foreach ($service in $services) {
        if (-not (Get-NetTCPConnection -LocalPort $service.Port -State Listen -ErrorAction SilentlyContinue)) {
            $ready = $false
        }
    }
} until ($ready -or (Get-Date) -gt $deadline)

if (-not $ready) {
    foreach ($service in $services) {
        if ($service.Process -and -not $service.Process.HasExited) {
            Stop-Process -Id $service.Process.Id -Force
        }
        $stderr = Join-Path $runtimeRoot "$($service.Name).stderr.log"
        if (Test-Path -LiteralPath $stderr) {
            Get-Content -LiteralPath $stderr -Tail 40
        }
    }
    throw "Backend services did not become ready within 30 seconds."
}

$health = Invoke-RestMethod -Uri "http://127.0.0.1:8888/api/v1/health" -TimeoutSec 5
foreach ($service in $services) {
    Write-Output "$($service.Name) PID=$($service.Process.Id) PORT=$($service.Port)"
}
Write-Output "health=$($health.status) api=http://127.0.0.1:8888"
Write-Output "logs=$runtimeRoot"
