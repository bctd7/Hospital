$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$repositoryRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$allSchemaTables = [System.Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)

foreach ($service in @("identity", "appointment", "guidance")) {
    $migrationDirectory = Join-Path $repositoryRoot "migrations/$service"
    $upFiles = @(Get-ChildItem -LiteralPath $migrationDirectory -Filter "*.up.sql")
    $downFiles = @(Get-ChildItem -LiteralPath $migrationDirectory -Filter "*.down.sql")
    if ($upFiles.Count -ne 1 -or $downFiles.Count -ne 1) {
        throw "$service must contain exactly one flattened up/down migration pair."
    }
    if (-not $upFiles[0].Name.StartsWith("000001_") -or -not $downFiles[0].Name.StartsWith("000001_")) {
        throw "$service flattened migration must use version 000001."
    }

    $upText = Get-Content -Raw -Encoding UTF8 $upFiles[0].FullName
    $downText = Get-Content -Raw -Encoding UTF8 $downFiles[0].FullName
    $createdTables = @([regex]::Matches($upText, '(?im)^CREATE TABLE(?: IF NOT EXISTS)?\s+`?([a-z0-9_]+)`?') |
        ForEach-Object { $_.Groups[1].Value })
    $droppedTables = @([regex]::Matches($downText, '(?im)^DROP TABLE(?: IF EXISTS)?\s+`?([a-z0-9_]+)`?') |
        ForEach-Object { $_.Groups[1].Value })
    if ($createdTables.Count -eq 0) {
        throw "$service initial migration does not create any tables."
    }

    $missingDown = @($createdTables | Where-Object { $_ -notin $droppedTables })
    $extraDown = @($droppedTables | Where-Object { $_ -notin $createdTables })
    if ($missingDown.Count -gt 0 -or $extraDown.Count -gt 0) {
        throw "$service up/down tables are not symmetric. Missing down: $($missingDown -join ', '); extra down: $($extraDown -join ', ')."
    }
    foreach ($table in $createdTables) {
        [void]$allSchemaTables.Add($table)
    }
    Write-Output "${service}: version=000001 tables=$($createdTables.Count)"
}

$seedFile = Join-Path $PSScriptRoot "seed-comprehensive-test-data.sql"
$seedText = Get-Content -Raw -Encoding UTF8 $seedFile
$seedTables = [System.Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
foreach ($pattern in @(
    '(?im)\bINSERT\s+INTO\s+`?([a-z0-9_]+)`?',
    '(?im)\bDELETE\s+FROM\s+`?([a-z0-9_]+)`?',
    '(?im)\bUPDATE\s+`?([a-z0-9_]+)`?'
)) {
    foreach ($match in [regex]::Matches($seedText, $pattern)) {
        [void]$seedTables.Add($match.Groups[1].Value)
    }
}
$unknownSeedTables = @($seedTables | Where-Object { -not $allSchemaTables.Contains($_) })
if ($unknownSeedTables.Count -gt 0) {
    throw "Comprehensive seed references tables absent from flattened migrations: $($unknownSeedTables -join ', ')."
}
Write-Output "Comprehensive seed table references are valid."
