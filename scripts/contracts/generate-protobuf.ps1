param(
    [switch]$Check
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$repositoryRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$protoRoot = Join-Path $repositoryRoot "contracts/proto"
$generatedRoot = Join-Path $repositoryRoot "contracts/gen"

function Require-Command {
    param([Parameter(Mandatory = $true)][string]$Name)

    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        throw "Required command '$Name' was not found in PATH."
    }
}

Require-Command "protoc"
$protoFiles = @(Get-ChildItem -LiteralPath $protoRoot -Recurse -Filter "*.proto" | ForEach-Object {
    $_.FullName.Substring($protoRoot.Length + 1).Replace("\", "/")
})
if ($protoFiles.Count -eq 0) {
    throw "No protobuf sources were found under $protoRoot."
}

Push-Location $repositoryRoot
try {
    if ($Check) {
        $descriptorFile = [IO.Path]::GetTempFileName()
        try {
            & protoc -I $protoRoot --include_imports --descriptor_set_out=$descriptorFile @protoFiles
            if ($LASTEXITCODE -ne 0) {
                throw "Protobuf validation failed with exit code $LASTEXITCODE."
            }
            Write-Output "Protobuf contracts are valid."
        }
        finally {
            Remove-Item -LiteralPath $descriptorFile -Force -ErrorAction SilentlyContinue
        }
        return
    }

    Require-Command "protoc-gen-go"
    Require-Command "protoc-gen-go-grpc"
    & protoc -I $protoRoot `
        --go_out=$generatedRoot --go_opt=paths=source_relative `
        --go-grpc_out=$generatedRoot --go-grpc_opt=paths=source_relative `
        @protoFiles
    if ($LASTEXITCODE -ne 0) {
        throw "Protobuf generation failed with exit code $LASTEXITCODE."
    }
    & gofmt -w $generatedRoot
    if ($LASTEXITCODE -ne 0) {
        throw "Formatting generated Go files failed with exit code $LASTEXITCODE."
    }
    Write-Output "Generated protobuf Go contracts under contracts/gen."
}
finally {
    Pop-Location
}
