param(
    [Parameter(Mandatory = $true)]
    [string]$OutputPath
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$scriptDirectory = Split-Path -Parent $MyInvocation.MyCommand.Path
$repositoryRoot = (Resolve-Path -LiteralPath (Join-Path $scriptDirectory "../../..")).Path
$dockerfile = Join-Path $repositoryRoot "deploy/production/Dockerfile"
$outputFullPath = [IO.Path]::GetFullPath($OutputPath)
$rawTarPath = "$outputFullPath.docker.tar"

if (Test-Path -LiteralPath $outputFullPath) {
    throw "Refusing to overwrite existing archive: $outputFullPath"
}
if (Test-Path -LiteralPath $rawTarPath) {
    throw "Refusing to overwrite existing temporary archive: $rawTarPath"
}

$images = @(
    "mysql:8.4.11",
    "redis:7.4.10-alpine",
    "apache/kafka:4.2.0",
    "hospital-production-identity-migrate:latest",
    "hospital-production-appointment-migrate:latest",
    "hospital-production-identity-bootstrap-admin:latest",
    "hospital-production-identity-rpc:latest",
    "hospital-production-appointment-rpc:latest",
    "hospital-production-app-api:latest"
)

function Invoke-Docker {
    param([string[]]$Arguments)
    & docker @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "docker $($Arguments -join ' ') failed with exit code $LASTEXITCODE"
    }
}

Push-Location $repositoryRoot
try {
    Invoke-Docker -Arguments @("pull", "mysql:8.4.11")
    Invoke-Docker -Arguments @("pull", "redis:7.4.10-alpine")
    Invoke-Docker -Arguments @("pull", "apache/kafka:4.2.0")
    Invoke-Docker -Arguments @("build", "--target", "db-migrate", "-t", $images[3], "-f", $dockerfile, ".")
    Invoke-Docker -Arguments @("tag", $images[3], $images[4])
    Invoke-Docker -Arguments @("build", "--target", "identity-bootstrap-admin", "-t", $images[5], "-f", $dockerfile, ".")
    Invoke-Docker -Arguments @("build", "--target", "identity-rpc", "-t", $images[6], "-f", $dockerfile, ".")
    Invoke-Docker -Arguments @("build", "--target", "appointment-rpc", "-t", $images[7], "-f", $dockerfile, ".")
    Invoke-Docker -Arguments @("build", "--target", "app-api", "-t", $images[8], "-f", $dockerfile, ".")
    Invoke-Docker -Arguments (@("save", "-o", $rawTarPath) + $images)

    $input = [IO.File]::OpenRead($rawTarPath)
    try {
        $output = [IO.File]::Create($outputFullPath)
        try {
            $gzip = [IO.Compression.GZipStream]::new($output, [IO.Compression.CompressionLevel]::Optimal)
            try {
                $input.CopyTo($gzip)
            }
            finally {
                $gzip.Dispose()
            }
        }
        finally {
            $output.Dispose()
        }
    }
    finally {
        $input.Dispose()
    }
}
finally {
    Pop-Location
    if (Test-Path -LiteralPath $rawTarPath) {
        Remove-Item -LiteralPath $rawTarPath -Force
    }
}

Write-Host "Image archive created: $outputFullPath"
