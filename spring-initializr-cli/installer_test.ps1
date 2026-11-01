# Runs only on a disposable Windows CI runner: installs into the real user profile and edits PATH.
param([Parameter(Mandatory)] [string] $ExpectedVersion)
$ErrorActionPreference = 'Stop'
$installDir = Join-Path $env:LOCALAPPDATA 'Programs\spring-init'
$setup = Join-Path $PSScriptRoot 'dist\spring-init-setup.exe'
$originalPath = (Get-Item 'HKCU:\Environment').GetValue('Path', $null, 'DoNotExpandEnvironmentNames')
if (Test-Path -LiteralPath $installDir) { throw "Test requires a clean runner: $installDir already exists." }

function RunInstaller($file, $arguments) {
    $process = Start-Process -FilePath $file -ArgumentList $arguments -WindowStyle Hidden -Wait -PassThru
    if ($process.ExitCode -ne 0) { throw "Installer exited with $($process.ExitCode)." }
}
function AssertInstalled {
    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    $entries = @($userPath -split ';' | Where-Object { $_.Trim().TrimEnd('\') -ieq $installDir })
    if ($entries.Count -ne 1) { throw 'Expected exactly one install directory in user PATH.' }
    $env:Path = "$installDir;" + [Environment]::GetEnvironmentVariable('Path', 'Machine')
    if ((spring-init --version) -ne "spring-init $ExpectedVersion") { throw 'Installed binary version mismatch.' }
}
try {
    # Existing expandable entries and empty entries must survive install/update/uninstall unchanged.
    $baseline = '%USERPROFILE%\spring-init-test-other;;' + $originalPath
    Set-ItemProperty 'HKCU:\Environment' -Name Path -Value $baseline -Type ExpandString
    RunInstaller $setup '/VERYSILENT /SUPPRESSMSGBOXES /NORESTART'
    AssertInstalled
    RunInstaller $setup '/VERYSILENT /SUPPRESSMSGBOXES /NORESTART'
    AssertInstalled
    RunInstaller (Join-Path $installDir 'unins000.exe') '/VERYSILENT /SUPPRESSMSGBOXES /NORESTART'
    if (Test-Path (Join-Path $installDir 'spring-init.exe')) { throw 'Binary remains after uninstall.' }
    $after = (Get-Item 'HKCU:\Environment').GetValue('Path', '', 'DoNotExpandEnvironmentNames')
    if ($after -cne $baseline) { throw 'Uninstall did not restore the other PATH entries.' }
    Write-Host 'PASS: installer install, update, version, PATH preservation and uninstall'
} finally {
    if ($null -eq $originalPath) { Remove-ItemProperty 'HKCU:\Environment' -Name Path -ErrorAction SilentlyContinue }
    else { Set-ItemProperty 'HKCU:\Environment' -Name Path -Value $originalPath -Type ExpandString }
}
