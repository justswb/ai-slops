param(
    [ValidatePattern('^v?\d+\.\d+\.\d+$')]
    [string] $Version = '0.0.0',
    [string] $Compiler
)
$ErrorActionPreference = 'Stop'
if (-not $Compiler) {
    $command = Get-Command ISCC.exe -ErrorAction SilentlyContinue
    if ($command) { $Compiler = $command.Source }
    else { $Compiler = Join-Path ${env:ProgramFiles(x86)} 'Inno Setup 6\ISCC.exe' }
}
if (-not (Test-Path -LiteralPath $Compiler)) { throw 'Install Inno Setup 6 or pass -Compiler with the path to ISCC.exe.' }
Push-Location $PSScriptRoot
try {
    $numericVersion = $Version.TrimStart('v')
    & go build -trimpath "-ldflags=-s -w -X main.version=v$numericVersion" -o dist/spring-init.exe .
    if ($LASTEXITCODE -ne 0) { throw 'Go build failed.' }
    & $Compiler "/DAppVersion=$numericVersion" installer/spring-init.iss
    if ($LASTEXITCODE -ne 0) { throw 'Installer compilation failed.' }
} finally { Pop-Location }
