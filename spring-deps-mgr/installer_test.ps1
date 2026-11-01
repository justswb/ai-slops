# This integration test intentionally changes the CI user's PATH. Never run on a developer account.
param(
    [Parameter(Mandatory)][string]$Setup,
    [Parameter(Mandatory)][string]$ExpectedVersion
)
$ErrorActionPreference = 'Stop'
if ($env:GITHUB_ACTIONS -ne 'true') { throw 'Run this PATH integration test only on a disposable GitHub Actions runner.' }
$Setup = (Resolve-Path -LiteralPath $Setup).Path
$testRoot = Join-Path $env:LOCALAPPDATA ('Programs\spring-deps-mgr-test-' + [guid]::NewGuid().ToString('N'))
$installDir = Join-Path $testRoot 'Install With Spaces'
$beforePath = (Get-Item 'HKCU:\Environment').GetValue('Path', '', 'DoNotExpandEnvironmentNames')
$beforeProcessPath = $env:Path
$beforeMachinePath = [Environment]::GetEnvironmentVariable('Path', 'Machine')
$uninstallKey = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\justswb.spring-deps-mgr_is1'

function UserPath { (Get-Item 'HKCU:\Environment').GetValue('Path', '', 'DoNotExpandEnvironmentNames') }
function Normalize([string]$Value) { $Value.Trim().Trim('"').TrimEnd('\').ToLowerInvariant() }
function PathCount { @((UserPath) -split ';' | Where-Object { (Normalize $_) -eq (Normalize $installDir) }).Count }
function RunInstaller([string]$File, [string[]]$Arguments) {
    $process = Start-Process -FilePath $File -ArgumentList $Arguments -PassThru -Wait -WindowStyle Hidden
    if ($process.ExitCode -ne 0) { throw "Installer failed with exit code $($process.ExitCode): $File" }
}
function AssertInstalled {
    if ((PathCount) -ne 1) { throw 'Expected exactly one user PATH entry.' }
    $env:Path = [Environment]::GetEnvironmentVariable('Path', 'Machine') + ';' + [Environment]::GetEnvironmentVariable('Path', 'User')
    $command = Get-Command spring-deps-mgr -CommandType Application -ErrorAction Stop | Select-Object -First 1
    if ((Normalize $command.Source) -ne (Normalize (Join-Path $installDir 'spring-deps-mgr.exe'))) { throw 'PATH resolved the wrong executable.' }
    $actual = & $command.Source --version
    if ($LASTEXITCODE -ne 0 -or $actual -ne $ExpectedVersion) { throw "Wrong installed version: $actual" }
    $registration = Get-ItemProperty -LiteralPath $uninstallKey -ErrorAction Stop
    if ($registration.DisplayVersion -ne ($ExpectedVersion -replace '^v', '')) { throw 'Wrong per-user uninstall registration version.' }
    if ([Environment]::GetEnvironmentVariable('Path', 'Machine') -cne $beforeMachinePath) { throw 'Machine PATH was changed.' }
}

try {
    $arguments = @('/VERYSILENT', '/SUPPRESSMSGBOXES', '/NORESTART', '/SP-', "/DIR=`"$installDir`"")
    RunInstaller $Setup $arguments
    AssertInstalled
    # Reinstall replaces a damaged binary and does not duplicate the PATH entry.
    [IO.File]::WriteAllText((Join-Path $installDir 'spring-deps-mgr.exe'), 'damaged')
    RunInstaller $Setup $arguments
    AssertInstalled
    $unrelated = Join-Path $installDir 'user-file.txt'
    [IO.File]::WriteAllText($unrelated, 'keep this file')
    RunInstaller (Join-Path $installDir 'unins000.exe') @('/VERYSILENT', '/SUPPRESSMSGBOXES', '/NORESTART')
    if (Test-Path -LiteralPath (Join-Path $installDir 'spring-deps-mgr.exe')) { throw 'Executable remains after uninstall.' }
    if ((PathCount) -ne 0) { throw 'PATH entry remains after uninstall.' }
    if ((UserPath) -cne $beforePath) { throw 'Uninstall changed unrelated user PATH content.' }
    if (Test-Path -LiteralPath $uninstallKey) { throw 'Per-user uninstall registration remains.' }
    if (-not (Test-Path -LiteralPath $unrelated)) { throw 'Uninstall deleted an unrelated user file.' }
    Write-Host 'PASS: per-user install, PATH resolution, reinstall, uninstall, and PATH preservation.'
} finally {
    $uninstaller = Join-Path $installDir 'unins000.exe'
    if (Test-Path -LiteralPath $uninstaller) {
        RunInstaller $uninstaller @('/VERYSILENT', '/SUPPRESSMSGBOXES', '/NORESTART')
    }
    $env:Path = $beforeProcessPath
}
