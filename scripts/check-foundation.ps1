$ErrorActionPreference = "Stop"

# Compatibility entry point. Keep one source of truth for project checks.
& (Join-Path $PSScriptRoot "check.ps1")
