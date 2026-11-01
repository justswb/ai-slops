# spring-deps-mgr

Gradle 프로젝트에서 바이너리를 실행하면 의존성 체크리스트가 열립니다. 패키지 이름이나 `add`/`remove` 명령을 입력할 필요 없이 화살표와 스페이스바로 선택하고 Enter로 반영합니다. 기존 `spring-initializr-cli`의 Initializr 카탈로그와 터미널 조작 방식을 참고했습니다.

## 실행

프로젝트 디렉터리의 터미널에서 실행하세요. 별도 창이나 전체 화면으로 전환하지 않고 기존 출력 아래에 선택 목록을 표시합니다. 종료 후에도 목록과 결과가 터미널에 남습니다.

```sh
spring-deps-mgr
```

```text
Dependencies | build.gradle | Boot 4.1.2
Up/Down: move  Space: toggle  Enter: apply  Ctrl-C: cancel
Filter:

> [x] Spring Web  (implementation)
  [ ] H2 Database  (runtimeOnly)
  [ ] Lombok  (annotationProcessor)
```

| 키 | 동작 |
|---|---|
| ↑ / ↓ | 항목 이동, 화면 끝에서 스크롤 |
| Space | 체크/해제 |
| Enter | 변경 사항을 한 번에 반영하고 종료 |
| Ctrl+C | 원본 파일을 변경하지 않고 취소 |
| 문자 입력 | 이름·좌표·configuration 검색 필터 |
| Backspace | 검색어 한 글자 삭제 |

현재 파일에 있는 의존성은 미리 체크됩니다. **체크하면 추가, 체크를 해제하면 제거**합니다. 필터로 숨겨진 항목의 선택도 유지됩니다. Enter로 적용할 때 파일 전체의 들여쓰기 탭을 탭 하나당 스페이스 4칸으로 변환합니다. 선택 변경이 없어도 이 변환은 적용하며 문자열 안의 탭과 기존 의존성의 명시적 버전은 유지합니다. Ctrl+C로 취소하면 포맷도 변경하지 않습니다.

기본 화면은 **개별 의존성/configuration 단위**입니다. 좌표 경로 대신 라이브러리 이름을 표시하고 카탈로그에 없는 기존 의존성은 artifact 이름을 표시합니다. 같은 Lombok 좌표라도 `compileOnly`, `annotationProcessor` 등을 각각 선택할 수 있습니다. 표시에서 숨긴 원래 좌표로도 검색할 수 있습니다. 추가 후보는 선택한 Boot 버전의 Initializr 카탈로그이고, 카탈로그 밖의 기존 문자열 의존성도 표시하여 제거할 수 있습니다. Maven Central 전체 검색이나 임의 좌표 입력 화면은 제공하지 않습니다.

## 라이브러리 묶음 선택

```sh
spring-deps-mgr --lib
```

이 화면에서는 `Spring Web`, `Lombok`, `Validation` 같은 Initializr 라이브러리 이름을 선택합니다. 화면을 처음 열 때만 체크된 항목을 맨 위로 모읍니다. 이후 체크·해제할 때는 다시 정렬하지 않고 항목 순서와 커서 위치를 유지합니다. 키 조작은 기본 화면과 같습니다. 개별 의존성이 아니라 Initializr가 생성하는 관련 의존성을 함께 추가·제거합니다. 예를 들어 Lombok은 compileOnly와 annotationProcessor, 해당 Boot 버전의 테스트용 의존성도 함께 처리합니다.

기존 파일에서 라이브러리의 대표 모듈이 하나라도 발견되면 체크합니다. 일부 의존성만 있는 라이브러리도 체크될 수 있으며, 라이브러리 선택을 변경하여 적용하면 체크된 라이브러리 묶음의 누락된 의존성도 채웁니다. 체크된 라이브러리끼리 공유하는 의존성은 유지합니다. `spring-boot-starter`, `spring-boot-starter-test`, `junit-platform-launcher`는 Initializr의 기본 의존성으로 취급하여 묶음에 포함하지 않습니다.

## 옵션

```sh
# 하위 모듈 / Kotlin DSL
spring-deps-mgr --file app/build.gradle.kts

# 선택 화면에서 Enter를 누르면 파일을 쓰는 대신 결과를 출력
spring-deps-mgr --dry-run

# Boot 버전 지정 / 다른 Initializr 서버
spring-deps-mgr --lib --boot-version 4.1.2
spring-deps-mgr --target https://start.spring.io

spring-deps-mgr --help
spring-deps-mgr --version
```

Boot 버전은 `org.springframework.boot` 플러그인의 리터럴 버전에서 감지합니다. 버전이 변수라서 감지할 수 없으면 `--boot-version`을 사용하세요. Boot 플러그인이 없으면 Initializr 기본 버전의 카탈로그를 사용합니다. 버전 없는 좌표를 해석할 BOM/의존성 관리와 configuration은 프로젝트에 준비되어 있어야 합니다.

카탈로그 조회에는 네트워크가 필요하고, 서비스에서 지원하지 않는 Boot 버전은 거부될 수 있습니다. 터미널 입력과 ANSI 출력이 가능한 실제 터미널에서 실행해야 합니다. Java, Gradle, Spring Boot CLI는 파일 편집에 필요하지 않습니다.

## 지원 범위

- Gradle Groovy `build.gradle`, Kotlin `build.gradle.kts`를 지원합니다. Maven은 지원하지 않습니다.
- 현재 디렉터리에서 파일을 찾습니다. 두 파일이 모두 있으면 `--file`이 필요합니다. 하위 모듈은 자동 탐색하지 않습니다.
- 최상위 `dependencies {}` 블록 하나의 문자열 좌표 선언을 편집합니다. 블록이 없으면 추가합니다. 여러 블록 또는 중첩된 블록만 있으면 오류로 종료합니다.
- 작은따옴표/큰따옴표, 괄호 유무, CRLF/LF를 처리하고 다른 설정을 보존합니다.
- 추가·수정 시 단순 의존성 선언의 들여쓰기는 스페이스 4칸으로 맞추고, 제거 시 독립된 선언의 줄 전체를 지웁니다. 실제 변경 시 블록 경계와 선언 사이에 이미 쌓인 공백 줄도 정리합니다. 주석·문자열 내용과 블록 밖의 공백은 보존합니다.
- version catalog, map notation, 변수/보간식, project/files, platform 등 동적 선언은 체크리스트 편집 대상이 아닙니다. 이 선언과 카탈로그 후보 간 중복은 자동 감지하지 않습니다.
- 후속 closure/exclude가 있는 기존 선언은 읽기 전용으로 표시합니다.
- BOM·추가 저장소·Gradle 플러그인 설정이 필요한 후보는 잠금 표시합니다. 라이브러리 생성 결과에서 추가 BOM/저장소나 복잡한 선언이 발견되면 파일을 변경하지 않고 오류로 종료합니다.
- 변경 배치 전체를 검증한 뒤 임시 파일로 교체합니다. 선택 중 원본이 바뀌면 덮어쓰기를 거부합니다. 심볼릭 링크는 지원하지 않습니다.
- Gradle 빌드/의존성 해석은 실행하지 않습니다. 일반적인 Gradle 프로그램 전체를 해석하는 도구는 아닙니다.

## 설치와 운영체제

Go나 JVM 없이 사용하는 Windows amd64 CLI이며, **`spring-deps-mgr-setup.exe` 설치 프로그램**으로 배포합니다. 배포 버전은 CI가 프로젝트의 기존 태그를 기준으로 산정하여 설치 프로그램과 CLI에 주입합니다.

| 운영체제 | 아키텍처 | 파일 |
|---|---|---|
| Windows | x64 (amd64) | `spring-deps-mgr-setup.exe` |

배포 대상은 Windows x64만 지원합니다. Linux/macOS 및 ARM64 설치 프로그램은 배포하지 않습니다.

1. GitHub Release에서 `spring-deps-mgr-setup.exe`를 다운로드하여 실행합니다. main 반영 전에는 CI의 **spring-deps-mgr-vX.Y.Z-dry-run** artifact로 설치를 시험할 수 있습니다.
2. 관리자 권한 없이 `%LOCALAPPDATA%\Programs\spring-deps-mgr`에 설치되고 **사용자 PATH**에 자동 등록됩니다. 시스템 PATH는 수정하지 않습니다.
3. 터미널을 새로 열고 프로젝트 디렉터리에서 `spring-deps-mgr`를 실행합니다. 기존 터미널 앱이 이전 PATH를 유지하면 앱 자체를 종료한 뒤 다시 여세요.

업데이트는 새 버전의 설치 프로그램을 다시 실행하면 됩니다. PATH 항목은 중복 추가하지 않습니다. 제거는 **Windows 설정 → 앱 → 설치된 앱 → spring-deps-mgr → 제거**에서 할 수 있으며, 설치한 CLI와 해당 PATH 항목만 제거하고 사용자 파일은 보존합니다.

무인 설치도 지원합니다.

```powershell
./spring-deps-mgr-setup.exe /VERYSILENT /SUPPRESSMSGBOXES /NORESTART /SP-
```

설치 프로그램은 CLI를 내장하므로 설치할 때 추가 다운로드는 필요하지 않습니다. 함께 배포하는 `SHA256SUMS`는 설치 프로그램의 SHA-256 체크섬입니다.

## 빌드·테스트·패키징

Go 1.24 이상이 필요합니다. 터미널 raw mode에는 기존 프로젝트와 같은 `golang.org/x/term`을 사용합니다.

```sh
cd spring-deps-mgr
go test ./...
go vet ./...
go build -trimpath -ldflags="-s -w -X main.version=0.0.0" -o dist/spring-deps-mgr.exe .
```

Windows amd64 패키징은 PowerShell 5.1 또는 PowerShell 7과 [Inno Setup](https://jrsoftware.org/isinfo.php)이 필요합니다. `package.ps1`은 CLI를 빌드한 뒤 설치 프로그램과 체크섬을 생성합니다. CLI만 필요하면 `-BinaryOnly`를 지정하세요.

```powershell
./package.ps1 -Version 0.0.0
./package.ps1 -Version 0.0.0 -ISCCPath 'C:\Program Files (x86)\Inno Setup 6\ISCC.exe'
./package.ps1 -Version 0.0.0 -BinaryOnly
# Windows 실행 정책으로 차단될 때 해당 프로세스에만 적용
powershell -NoProfile -ExecutionPolicy Bypass -File ./package.ps1 -Version 0.0.0
```

릴리스 정책은 `spring-initializr-cli`와 같습니다.

- 관련 코드가 `main`에 반영되면 최신 `spring-deps-mgr/vX.Y.Z` 태그에서 patch를 올려 자동 태깅·배포합니다.
- 다른 브랜치의 push는 버전 계산·빌드·테스트만 수행하는 dry run이며 설치 프로그램과 체크섬을 artifact로 보관합니다. 릴리스나 태그를 만들지 않습니다.
- **Run workflow**에서 patch/minor/major를 선택할 수 있습니다. `dry_run`을 켜면 배포하지 않으며, main 외 브랜치에서는 항상 dry run입니다.
- 명시적으로 `spring-deps-mgr/vX.Y.Z` 태그를 push하는 경우에는 기존 도구와 동일하게 해당 버전으로 배포합니다. 이 수동 태그 경로는 main 반영 없이도 배포하므로 사전 버전 표시 목적으로 태그를 push하지 마세요.
- 같은 커밋의 재실행은 기존 태그를 재사용합니다. 이전 태그가 전혀 없으면 첫 버전은 v0.1.0(major 선택 시 v1.0.0)입니다.

CI는 Windows에서 CLI 테스트·vet·버전 검증과 실제 무인 설치·재설치·제거 테스트를 수행합니다. `installer_test.ps1`은 CI 사용자 PATH를 변경하므로 일회용 GitHub Actions runner에서만 실행하도록 제한했습니다.
