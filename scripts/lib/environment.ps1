Set-StrictMode -Version Latest

function Import-ProjectEnvironment {
    param(
        [Parameter(Mandatory = $true)]
        [string]$RepositoryRoot,

        [switch]$Override
    )

    $environmentFile = Join-Path $RepositoryRoot ".env"
    if (-not (Test-Path -LiteralPath $environmentFile)) {
        return
    }

    foreach ($rawLine in Get-Content -LiteralPath $environmentFile -Encoding UTF8) {
        $line = $rawLine.Trim()
        if ($line.Length -eq 0 -or $line.StartsWith("#")) {
            continue
        }

        $separator = $line.IndexOf("=")
        if ($separator -lt 1) {
            throw "Invalid .env entry: $rawLine"
        }

        $name = $line.Substring(0, $separator).Trim()
        $value = $line.Substring($separator + 1).Trim()
        if ($name -notmatch "^[A-Za-z_][A-Za-z0-9_]*$") {
            throw "Invalid environment variable name in .env: $name"
        }

        if ($value.Length -ge 2) {
            $first = $value[0]
            $last = $value[$value.Length - 1]
            if (($first -eq '"' -and $last -eq '"') -or ($first -eq "'" -and $last -eq "'")) {
                $value = $value.Substring(1, $value.Length - 2)
            }
        }

        if ($Override -or [string]::IsNullOrEmpty([Environment]::GetEnvironmentVariable($name, "Process"))) {
            [Environment]::SetEnvironmentVariable($name, $value, "Process")
        }
    }
}

function Get-RequiredEnvironmentValue {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    $value = [Environment]::GetEnvironmentVariable($Name, "Process")
    if ([string]::IsNullOrWhiteSpace($value)) {
        throw "Required environment variable $Name is missing. Configure it in .env or the current process."
    }
    return $value
}
