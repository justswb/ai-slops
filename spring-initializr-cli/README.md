# spring-initializr-cli

Interactive terminal tool that builds a `spring init` command from [Spring Initializr](https://start.spring.io) options and runs it.

Pick project type, language, Boot/Java versions, coordinates and dependencies with arrow keys; `Esc` goes back a step.

> **Note:** Only Windows is currently supported.

## Tech stack

- **Go 1.27+** (with `golang.org/x/term`)
- **Spring Boot CLI** (`spring`) **2.6 or later**, needed at runtime unless you pass `--dry-run`
- **GitHub Actions** for build, test and release

## Supported Spring Boot CLI versions

| Spring Boot CLI | Supported |
|---|---|
| 4.x | ✅ |
| 3.x | ✅ |
| 2.6 – 2.7 | ✅ |
| 2.5 and older | ❌ `spring init` there uses `--groupId` / `--artifactId` instead of `--group-id` / `--artifact-id` |

Install: `sdk install springboot`, or see the [installation guide](https://docs.spring.io/spring-boot/installing.html).

## Usage

Download and run `spring-init-setup.exe` from [Releases](https://github.com/justswb/ai-slops/releases). Run it again to update. It needs no admin rights: it puts `spring-init.exe` in `%LOCALAPPDATA%\Programs\spring-init` and adds that folder to your user `PATH`.

Alternatively, install using PowerShell:

```powershell
irm https://raw.githubusercontent.com/justswb/spring-initializr-cli/master/install.ps1 | iex
```

Uninstall the setup installation from Windows **Settings > Apps > Installed apps**. This removes the binary and its user PATH entry. Open a new terminal after installing to use the command.

Uninstall a PowerShell installation:

```powershell
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/justswb/spring-initializr-cli/master/install.ps1))) -Uninstall
```

Pass `-Version v2.0.3` the same way to install a specific version. You can read [install.ps1](install.ps1) first, or download `spring-init.exe` from [Releases](https://github.com/justswb/spring-initializr-cli/releases) and put it on `PATH` yourself.

Then, in a new terminal:

```
spring-init                # choose options, then run `spring init ...`
spring-init --dry-run      # only print the command
spring-init --target URL   # use another Initializr service
spring-init --version      # print the version (also -v)
spring-init --help         # print usage (also -h)
```

## Build

```
go build -trimpath -ldflags="-s -w" -o dist/spring-init.exe .
```

To build the Windows installer, install [Inno Setup 6](https://jrsoftware.org/isinfo.php), then run:

```powershell
./build-installer.ps1 -Version v2.0.3
```

Output: `dist/spring-init-setup.exe` and `dist/spring-init.exe`.

## Test

```
go test ./...
```

## Release

- Push to `main`: tags the next patch version and publishes `spring-init-setup.exe` and the standalone exe.
- CI tests the Windows installer with `installer_test.ps1` on a disposable Windows runner (install, update, uninstall, version and user PATH preservation).
- CI also tests `install.ps1` with `install_test.ps1` (install, update, uninstall, user `PATH`). That script really edits your user `PATH`, so don't run it on your own machine.
- **Actions → Release → Run workflow**: choose a patch, minor or major bump. Tick *Dry run* to check the version without releasing.
- Push to any other branch: dry run only.
