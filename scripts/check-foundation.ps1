$ErrorActionPreference = "Stop"

$repositoryRoot = Split-Path -Parent $PSScriptRoot
Push-Location $repositoryRoot

try {
    Write-Host "[1/4] Validating go-zero API contract..."
    & goctl api validate -api contracts/api/app.api
    if ($LASTEXITCODE -ne 0) {
        throw "goctl API validation failed with exit code $LASTEXITCODE"
    }

    Write-Host "[2/4] Validating event envelope JSON..."
    Get-Content -Raw -Encoding UTF8 contracts/events/event-envelope.schema.json |
        ConvertFrom-Json |
        Out-Null

    Write-Host "[3/4] Validating Docker Compose configuration..."
    & docker compose `
        --env-file .env.example `
        -f deploy/compose/docker-compose.yml `
        --profile messaging `
        config --quiet
    if ($LASTEXITCODE -ne 0) {
        throw "Docker Compose validation failed with exit code $LASTEXITCODE"
    }

    Write-Host "[4/4] Running current Go tests..."
    & go test ./...
    if ($LASTEXITCODE -ne 0) {
        throw "Go tests failed with exit code $LASTEXITCODE"
    }

    Write-Host "Foundation checks passed."
}
finally {
    Pop-Location
}
