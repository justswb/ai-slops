# 구현 결정

- 기존 spring-initializr-cli와 동일한 Go CLI. 표준 라이브러리와 기존 프로젝트에서도 사용하는 x/term, x/sys로 독립 바이너리를 배포한다.
- 바이너리를 실행하면 체크리스트를 연다. 화살표로 이동, Space로 선택/해제, Enter로 변경을 일괄 적용한다. add/remove 하위 명령이나 좌표 입력을 요구하지 않는다.
- 기본 화면은 Initializr 카탈로그의 개별 좌표 및 기존 의존성을 configuration별로 다룬다. `--lib` 화면은 라이브러리 묶음을 다룬다. 기존 항목은 미리 체크하며 체크 해제는 제거다.
- `--lib`는 Initializr `/starter.zip` 생성 결과를 사용한다. ID를 starter 이름으로 추측하지 않아 Boot 버전별 좌표와 annotation processor 구성을 반영한다.
- Gradle은 실행 가능한 언어이므로 전체 해석기를 만들지 않는다. 주석/문자열을 구별하는 스캐너와 제한된 리터럴 선언 편집기를 사용하며 복잡한 선언은 보존한다. 모호한 대상 블록은 오류로 종료한다.
- BOM/저장소 등 의존성 블록 외 설정이 필요한 라이브러리는 자동 적용을 거부한다. 부분 적용으로 빌드를 손상시키지 않는다.
- 배포 바이너리는 Windows amd64 `spring-deps-mgr.exe` 하나로 제한한다. 다른 OS/아키텍처는 배포하지 않는다. 소스의 Windows/Linux/macOS CI 테스트는 유지한다.
- 터미널 raw mode는 기존 프로젝트와 동일한 `golang.org/x/term`을 사용한다. 키 입력/선택 모델을 분리해 파일 변경 없이 조작을 테스트한다. ANSI 화면은 기존 프로젝트 방식을 재사용하며 별도 TUI 프레임워크는 추가하지 않는다.
