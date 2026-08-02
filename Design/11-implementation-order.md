# 구현 순서

설계된 기능을 한 번에 연결하지 않고, 외부 시스템에 영향을 주지 않는 핵심
계약부터 구현한다. 각 단계가 독립적으로 검증된 뒤 다음 단계로 넘어가야
한다.

## 순서 요약

| 단계 | 구현 범위 | 완료 기준 |
| --- | --- | --- |
| 0 | Go 프로젝트 골격 | 단일 바이너리 빌드와 기본 실행이 가능하다 |
| 1 | 도메인 타입과 공통 계약 | provider, registry, scope, 오류, adapter 인터페이스가 테스트된다 |
| 2 | TOML 설정 저장소 | 스키마 검증, 권한, 원자적 저장, round-trip이 테스트된다 |
| 3 | 자격 증명 저장소 | 메모리 fake와 macOS Keychain adapter가 같은 계약을 만족한다 |
| 4 | 외부 명령과 전환 엔진 | fake provider로 사전 검증·적용·롤백이 검증된다 |
| 5 | 읽기 전용 CLI | `init`, `current`, `list`, `config validate`, 기본 `doctor`가 동작한다 |
| 6 | GitHub Provider | `gh` 계정 전환, host 필수 규칙, 롤백이 동작한다 |
| 7 | Codex Provider | 프로파일별 `CODEX_HOME`, auth cache 저장·복원, 재로그인 안내가 동작한다 |
| 8 | 쓰기 명령과 부분 전환 | `add`, `change`, `--only`, 실패 시 복구가 동작한다 |
| 9 | 셸 연동과 배포 | zsh wrapper, 설치 흐름, macOS 수동 검증이 완료된다 |

## 0. 프로젝트 골격

완료했다.

- Go module과 `cmd/account-manager` 진입점을 만들었다.
- `internal/app`, `config`, `credentials`, `execution`, `providers` 경계를
  만들었다.
- Go 1.26.5 환경에서 `go test ./...`와 단일 바이너리 빌드를 통과했다.

## 1. 도메인 타입과 공통 계약

### 현재 구현 상태

1단계부터 9단계의 핵심 계약과 테스트가 구현되어 있다. 현재 바이너리는 읽기·쓰기
CLI, GitHub/Codex provider, Keychain adapter, provider transaction, zsh patch를
포함한다. zsh 설정 파일에 wrapper를 자동 설치하는 동작과 실제 계정/Keychain을
사용하는 macOS 수동 통합 검증은 별도 작업으로 남아 있다.

### 구현 세부

먼저 실제 파일, Keychain, `gh`, `codex`에 접근하지 않는 순수 코드부터
구현한다.

- 프로파일 ID, provider ID, provider 범위, 활성 상태 타입을 정의한다.
- 설계 문서의 오류 코드와 종료 코드 타입을 정의한다.
- `Provider`, `ProviderRegistry`, `CommandRunner`, `CredentialStore` 인터페이스를
  정의한다.
- 메모리 기반 registry, command runner, credential store fake를 만든다.
- 잘못된 프로파일·provider·범위가 공통 규칙으로 거절되는지 테스트한다.

이 단계의 목적은 실제 provider를 붙이기 전에 애플리케이션의 의존 방향과
테스트 방법을 고정하는 것이다. 첫 구현 작업은 이 단계에서 시작한다.

## 2. TOML 설정 저장소

공통 타입을 바탕으로 설정의 읽기와 쓰기를 구현한다.

- `~/.config/account-manager/config.toml` 경로를 OS 경로 규칙으로 계산한다.
- TOML decode 후 schema와 active 상태를 검증한다.
- 디렉터리 `0700`, 파일 `0600` 정책을 적용한다.
- 임시 파일 작성, flush/sync, rename 순서로 원자적 저장을 구현한다.
- provider가 추가되어도 알 수 없는 provider 데이터가 보존되는지 확인한다.
- 정상 round-trip, 손상된 TOML, 잘못된 버전, 중복 활성 상태를 테스트한다.

이 단계가 끝나면 아직 계정을 전환하지 않아도 `init`, `config validate`,
설정 상태 표시의 기반을 사용할 수 있다.

## 3. 자격 증명 저장소

보안 경계는 테스트 구현부터 만든 뒤 운영체제 adapter로 확장한다.

1. `MemoryCredentialStore`로 Set/Get/Delete/Exists 계약과 오류 처리를
   테스트한다.
2. macOS 전용 파일에서 Security.framework의 SecItem API를 감싼다.
3. service, account, 접근 가능 시점, Data Protection Keychain 설정을 설계
   문서의 정책대로 적용한다.
4. Keychain 잠금·거부·손상 시 plaintext fallback이 발생하지 않는지 확인한다.

Keychain 테스트는 비밀값을 출력하지 않으며, 실제 macOS Keychain을 사용하는
수동 통합 테스트와 메모리 fake를 사용하는 자동 테스트를 분리한다.

## 4. 외부 명령과 전환 엔진

provider 구현 전에 실행과 트랜잭션을 분리한다.

- `CommandRunner`는 shell 문자열이 아니라 명령어와 argv 배열로 실행한다.
- stdout, stderr, 종료 코드를 구분하고 secret을 로그에 넣지 않는다.
- 전환 엔진은 선택된 provider만 대상으로 preflight → apply → commit을
  수행한다.
- 중간 실패 시 이미 적용한 provider를 역순으로 rollback한다.
- GitHub처럼 외부 도구의 persistent state를 바꾸는 provider도 rollback
  계획에 포함할 수 있도록 한다.

fake provider로 성공, preflight 실패, apply 실패, rollback 실패, 부분 범위
전환을 먼저 검증한다. 이 단계가 있어야 GitHub와 Codex의 실제 부작용을
테스트에서 통제할 수 있다.

## 5. 읽기 전용 CLI

데이터를 변경하지 않는 명령부터 사용자 인터페이스를 연결한다.

- 인자 파싱과 공통 출력 형식을 구현한다.
- `init`, `current`, `list`, `config validate`를 설정 저장소에 연결한다.
- `doctor`는 명령 존재 여부, 설정 유효성, provider 등록 상태부터 점검한다.
- 오류 코드에 대응하는 종료 코드와 사람이 읽을 수 있는 메시지를 고정한다.

이 시점에는 `gh` 계정이나 Codex 인증을 변경하지 않는다. 사용자는 설정의
현재 상태와 문제를 확인할 수 있어야 한다.

## 6. GitHub Provider

Codex보다 외부 상태와 인증 경계가 단순한 GitHub를 먼저 연결한다.

- 등록 시 `--host`를 받고, 최초 등록에서 host가 없으면 실패시킨다.
- `gh auth status`와 필요한 확인 명령을 `CommandRunner`로 실행한다.
- 계정 선택은 `gh auth switch --hostname <host> --user <account>`에 위임한다.
- 토큰, `.env`, `GH_TOKEN`, `GH_ENTERPRISE_TOKEN`은 읽거나 저장하지 않는다.
- 적용 전 host/account를 저장하고, 실패 시 이전 `gh` 계정으로 복원한다.
- 실제 `gh`가 없는 환경과 비로그인 환경의 오류를 명확히 구분한다.

GitHub Provider 단독 contract 테스트를 fake runner로 작성한 뒤, 실제 `gh`
계정을 사용하는 macOS 수동 테스트를 추가한다.

## 7. Codex Provider

Codex는 인증 저장 위치가 둘일 수 있으므로 파일 기반 경로와 재로그인 경로를
분리하여 구현한다.

- 프로파일별 `CODEX_HOME` 경로를 계산하고 권한을 적용한다.
- file-backed `auth.json`은 opaque bytes로 읽어 Keychain에 저장한다.
- 활성화 시 해당 프로파일의 cache를 복원하고 이전 상태를 rollback 정보로
  보관한다.
- auth cache가 없으면 `CODEX_HOME=... codex login` 안내를 반환한다.
- Codex 내부 keyring 형식을 직접 해석하지 않는다.
- 비밀값과 auth cache를 로그나 오류 메시지에 포함하지 않는다.

먼저 임시 디렉터리와 `MemoryCredentialStore`로 경로·복원 테스트를 하고,
실제 Codex CLI와 Keychain을 사용하는 테스트는 별도의 macOS 수동 단계로
둔다.

## 8. 쓰기 명령과 부분 전환

provider가 독립적으로 동작한 뒤 애플리케이션 명령에 연결한다.

- `add --<profile> [--github] [--codex]`를 provider별 등록 흐름에 연결한다.
- `change --<profile>`은 등록된 모든 provider를 선택한다.
- `--github`, `--codex`, `--only <provider>`는 선택한 provider만 변경한다.
- preflight에서 모든 대상을 확인한 뒤 apply하고, 실패하면 역순 rollback한다.
- 일부 provider만 성공한 상태를 출력하고, rollback 실패는 별도 오류로
  표시한다.

`add --work --github --host ...`처럼 최초 GitHub host를 명시하는 흐름과,
Codex만 등록하거나 전환하는 흐름을 각각 end-to-end 테스트한다.

## 9. 셸 연동과 배포

핵심 전환 기능이 검증된 마지막에 현재 셸과 설치 동작을 연결한다.

- `change --shell zsh`가 필요한 환경 변수 export/unset만 출력하도록 한다.
- `account-manager init zsh`가 wrapper를 설치하거나 갱신한다.
- 직접 실행할 때 부모 셸이 바뀌지 않는다는 점을 경고한다.
- zsh subprocess 테스트에서 wrapper가 올바른 바이너리를 재호출하는지
  검증한다.
- macOS arm64 빌드와 설치 문서를 확인한다.

## 단계별 공통 완료 조건

각 단계는 다음 조건을 모두 만족해야 다음 단계로 이동한다.

- 해당 단계의 자동 테스트가 통과한다.
- 오류 경로와 secret 비노출 경로가 테스트된다.
- 이전 단계의 contract를 깨지 않는다.
- 설계 문서와 실제 동작의 차이가 있으면 문서를 먼저 갱신한다.
- `go test ./...`, `go vet ./...`, `git diff --check`가 통과한다.

## 구현하지 않는 순서

다음 항목은 초기 구현에서 앞당기지 않는다.

- GitHub 토큰을 account-manager Keychain에 중복 저장하는 기능
- Codex 내부 keyring 형식 직접 해석
- 여러 OS와 셸을 동시에 지원하는 추상화
- provider 하나의 성공만으로 전체 전환을 완료하는 비원자적 흐름
- 실제 Keychain을 대체하는 평문 파일 fallback
