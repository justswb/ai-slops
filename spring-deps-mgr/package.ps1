param(
    [string]$Version = '0.0.0',
    [switch]$BinaryOnly,
    [string]$ISCCPath
)
$ErrorActionPreference = 'Stop'
if ($Version -notmatch '^[A-Za-z0-9._+-]+$') { throw 'Invalid version' }
Push-Location $PSScriptRoot
$previousOS = $env:GOOS
$previousArch = $env:GOARCH
$previousCGO = $env:CGO_ENABLED
try {
    $env:CGO_ENABLED = '0'
    New-Item -ItemType Directory -Force -Path dist | Out-Null
    $env:GOOS = 'windows'
    $env:GOARCH = 'amd64'
    $outputPath = 'dist/spring-deps-mgr.exe'
    go build -trimpath -ldflags "-s -w -X main.version=$Version" -o $outputPath .
    if ($LASTEXITCODE -ne 0) { throw 'Build failed: windows/amd64' }
    # Remove only the previous packaging outputs, leaving other dist contents alone.
    foreach ($obsoleteName in @(
        'spring-deps-mgr-windows-amd64.exe', 'spring-deps-mgr-windows-arm64.exe',
        'spring-deps-mgr-linux-amd64', 'spring-deps-mgr-linux-arm64',
        'spring-deps-mgr-darwin-amd64', 'spring-deps-mgr-darwin-arm64'
    )) {
        $obsoletePath = Join-Path $PSScriptRoot "dist/$obsoleteName"
        if (Test-Path -LiteralPath $obsoletePath) { Remove-Item -LiteralPath $obsoletePath }
    }
    $packagePath = $outputPath
    if (-not $BinaryOnly) {
        $numericVersion = $Version -replace '^v', ''
        if ($numericVersion -notmatch '^\d+\.\d+\.\d+$') { throw 'Installer version must be X.Y.Z or vX.Y.Z' }
        if (-not $ISCCPath) {
            $compiler = Get-Command ISCC.exe -ErrorAction SilentlyContinue
            if ($compiler) { $ISCCPath = $compiler.Source }
            foreach ($candidate in @(
                "${env:ProgramFiles(x86)}\Inno Setup 6\ISCC.exe",
                "$env:ProgramFiles\Inno Setup 6\ISCC.exe",
                "$env:ProgramFiles\Inno Setup 7\ISCC.exe"
            )) {
                if (-not $ISCCPath -and (Test-Path -LiteralPath $candidate)) { $ISCCPath = $candidate }
            }
        }
        if (-not $ISCCPath) { throw 'Install Inno Setup or pass -ISCCPath. Use -BinaryOnly to build just the CLI.' }
        & $ISCCPath "/DAppVersion=$numericVersion" (Join-Path $PSScriptRoot 'installer.iss')
        if ($LASTEXITCODE -ne 0) { throw 'Installer compilation failed' }
        $packagePath = 'dist/spring-deps-mgr-setup.exe'
    }
    $digest = (Get-FileHash -LiteralPath $packagePath -Algorithm SHA256).Hash.ToLowerInvariant()
    $hashes = @("$digest  $([IO.Path]::GetFileName($packagePath))")
    [IO.File]::WriteAllLines((Join-Path $PSScriptRoot 'dist/SHA256SUMS'), [string[]]$hashes, [Text.UTF8Encoding]::new($false))
} finally {
    $env:GOOS = $previousOS
    $env:GOARCH = $previousArch
    $env:CGO_ENABLED = $previousCGO
    Pop-Location
}
