# 구현 기술과 테스트 전략

## 확정 결정

- Keychain은 Go에서 직접 연결하는 얇은 macOS native adapter로 구현한다.
- 최종 배포물은 Go 단일 바이너리다.
- Codex는 file-backed auth cache를 자동 import하고, keyring-backed 상태는
  프로파일별 재로그인을 안내한다.
- GitHub 인증은 `gh` credential store에 위임한다.
- `account-manager`는 `.env`를 읽지 않으며 GitHub 토큰을 저장하지 않는다.
- `GH_ENTERPRISE_URL`은 `gh`나 `account-manager`의 표준 인증 변수로 사용하지
  않는다. 다른 도구가 필요로 하는 경우에만 별도로 유지할 수 있다.
- Keychain 접근이 실패하거나 잠겨 있으면 plaintext 파일로 fallback하지 않고
  실패한다.

## 단일 바이너리 결정

사용자는 Node.js나 별도 런타임 없이 다음처럼 실행할 수 있어야 한다.

```sh
account-manager change --work --codex
```

따라서 최종 배포물은 운영체제와 CPU에 맞는 단일 native binary로 만든다.
개발 중에는 패키지 매니저를 사용할 수 있지만, 사용자가 실행하는 최종
인터페이스는 `account-manager` 하나로 유지한다.

### 가능한 구현 언어

| 선택지 | 장점 | 단점 |
| --- | --- | --- |
| Go | 단일 바이너리, 빠른 빌드, CLI와 외부 프로세스 연동이 단순함 | macOS Keychain은 native adapter가 필요함 |
| Rust | 강한 타입·메모리 안전성, 단일 바이너리 | 학습·빌드 복잡도가 상대적으로 높음 |
| Swift | macOS Keychain과 가장 자연스럽게 통합됨 | macOS 중심이 되고 향후 다른 OS 확장이 어려움 |
| Node.js 패키징 | 빠른 초기 개발 | native 모듈과 런타임 포함 배포가 복잡함 |

v1의 기준안은 Go다. Go 자체가 설치되어 있지 않아도 개발자가 빌드한
바이너리를 배포할 수 있고, 향후 Linux나 다른 credential store를 추가하기도
쉽다.

## v1 구현 기준

v1은 현재 사용 환경을 기준으로 다음 범위를 우선 지원한다.

- 운영체제: macOS
- 셸: zsh
- 외부 도구: `gh`, `codex`
- 자격 증명 저장소: macOS Keychain
- 설정 형식: TOML
- 배포 형태: native single binary

향후 Linux, 다른 셸, 다른 OS의 credential store를 추가할 수 있도록 외부
의존성은 adapter로 격리한다.

## 구현 경계

언어가 바뀌더라도 다음 경계는 유지한다.

- 설정, 프로파일, provider 데이터는 정적 타입으로 표현한다.
- `gh`와 `codex`는 외부 프로세스로 실행한다.
- CLI 파싱, TOML 처리, Keychain 연동은 각각 별도 모듈로 감싼다.
- 실행 시 shell을 거치지 않고 argv 배열로 외부 명령을 호출한다.
- core 로직과 OS·credential store adapter를 분리한다.

### CredentialStore adapter

애플리케이션 코드는 OS API나 `/usr/bin/security`를 직접 호출하지 않는다.

```text
CredentialStore
├── MacOSKeychainCredentialStore
├── MemoryCredentialStore  # 테스트용
└── FileCredentialStore    # 개발용, 기본 비활성화
```

macOS 구현은 비밀번호를 명령행 인자로 넘기지 않는 네이티브 Keychain 연동을
사용한다. `/usr/bin/security`를 사용해야 하는 경우에도 secret을 argv나 로그에
노출하지 않는 별도 adapter로 제한한다.

Keychain 연동 방식은 세 가지가 있다.

1. **언어용 native Keychain library**
   - 애플리케이션에서 바로 Keychain API를 호출한다.
   - 단일 바이너리와 가장 잘 맞는다.
   - library의 유지보수 상태와 macOS 버전을 확인해야 한다.
2. **Swift helper 또는 FFI**
   - Apple의 Security framework를 직접 사용한다.
   - 보안 동작은 명확하지만 빌드와 배포 구조가 복잡해진다.
3. **`/usr/bin/security` 명령 호출**
   - 별도 library가 필요 없다.
   - secret을 명령행 인자로 넘기기 쉽고 프로세스 목록이나 로그에 노출될
     위험이 있어 기본 구현으로 사용하지 않는다.

Apple은 Keychain Services를 암호화된 데이터베이스에 작은 비밀 데이터를
저장하는 API로 제공하며, macOS에서는 Security framework의 SecItem 계열 API를
사용할 수 있다. [Apple Keychain Services 문서](https://developer.apple.com/documentation/security/keychain-services)

### Go 라이브러리 선택

Go에서 검토할 수 있는 선택지는 다음과 같다.

| 선택지 | 특징 | 판단 |
| --- | --- | --- |
| `zalando/go-keyring` | 간단한 `Set/Get/Delete` API와 여러 OS 지원 | macOS에서 `security` 명령을 사용하고 크기 제한이 있어 기본 선택에서 제외 |
| `keybase/go-keychain` | macOS Security API에 가까운 native Go wrapper | 직접 native 연동에 가깝지만 API가 Apple 스타일이고 플랫폼 범위가 제한적 |
| `lexfrei/keychain` | cgo 없이 native Security.framework를 사용하고 여러 OS 지원 | 유망하지만 비교적 새로운 라이브러리이므로 v1 핵심 경로에 바로 의존하기 전에 검토 필요 |
| 자체 얇은 adapter | 필요한 SecItem 동작만 직접 감싼다 | 코드가 늘지만 동작과 보안 경계를 직접 통제할 수 있음 |

`zalando/go-keyring`은 macOS에서 `/usr/bin/security`를 사용하므로 secret이
명령행에 노출될 가능성이 있고, macOS 저장 데이터 크기 제한도 있다. 반면
`lexfrei/keychain`은 Security.framework를 직접 사용하지만 unsigned binary의
재빌드와 Keychain access partition에 관한 주의점이 있다. [go-keyring 구현 설명](https://github.com/zalando/go-keyring), [lexfrei/keychain 구현 설명](https://github.com/lexfrei/keychain)

v1은 `MacOSKeychainCredentialStore`를 자체 얇은 adapter로 구현한다. 필요한
동작은 `Set`, `Get`, `Delete`, `Exists`뿐이므로 범위를 작게 유지한다.

```text
Go application
    ↓
CredentialStore interface
    ↓
MacOSKeychainCredentialStore
    ↓
Security.framework / SecItem API
```

이렇게 하면 secret이 subprocess argv에 들어가지 않고, 외부 라이브러리의
저장 방식에 애플리케이션 전체가 종속되지 않는다. Linux나 Windows 지원 시에는
동일한 `CredentialStore`에 해당 OS adapter를 추가한다.

### Keychain 보안 정책

v1 adapter는 macOS Security.framework의 SecItem API만 사용한다.
deprecated된 legacy `SecKeychain*` API나 `/usr/bin/security` subprocess는
사용하지 않는다.

Keychain item은 다음 기준으로 만든다.

```text
class: generic password
service: com.seongmin221.ai-account-manager
account: codex/<profile>
accessible: when unlocked
synchronizable: false
```

추가 정책:

- `kSecUseDataProtectionKeychain`을 활성화한다.
- `kSecAttrAccessibleWhenUnlocked`를 사용해 잠긴 기기에서는 읽지 못하게 한다.
- `trust-all` ACL이나 모든 애플리케이션 허용 옵션을 사용하지 않는다.
- Keychain 접근 거부·잠금·손상 시 파일 저장소로 fallback하지 않는다.
- Keychain 오류에는 secret을 포함하지 않고 `AM016`으로 반환한다.
- 개발 빌드와 배포 빌드가 같은 secret namespace를 사용하더라도 macOS의
  사용자 인증과 Keychain 접근 제어를 우회하지 않는다.

이 정책은 “같은 macOS 사용자로 실행 중인 악성 프로세스”까지 막는 설계가
아니다. 대신 파일 평문 저장, 명령행 노출, 임의 애플리케이션 허용을 피하고
OS Keychain의 보호 경계를 사용한다.

## Codex 인증 저장 방식의 맥락

Codex 계정 전환에서 중요한 것은 현재 로그인된 계정이 반드시 일반 파일에
있는 것은 아니라는 점이다.

Codex는 인증 상태를 다음 중 하나에 저장할 수 있다.

```text
파일 기반: ~/.codex/auth.json
OS 저장소: macOS Keychain 등
```

파일 기반이면 `account-manager`가 현재 `auth.json`을 읽어 opaque 데이터로
자신의 Keychain에 보관할 수 있다. 반면 Codex가 OS 저장소에 직접 보관한
credential은 `account-manager`가 공식적인 Codex API를 통해 추출하거나 복제할
수 있다고 가정하면 안 된다.

쉽게 말하면 Codex가 자기 열쇠고리에 보관한 열쇠를 다른 앱이 안전하게 복사할
수 있다는 보장이 없다. 선택지는 다음과 같다.

1. **파일 기반 인증을 자동 import**
   - `add --work --codex`가 현재 auth cache를 저장한다.
   - v1에서 즉시 지원하기 쉽다.
2. **프로파일별 Codex home에서 다시 로그인**
   - `CODEX_HOME=... codex login`을 실행한다.
   - keyring 기반 인증까지 지원할 수 있지만 프로파일마다 한 번씩 로그인해야
     한다.
3. **Codex 내부 keyring 형식을 직접 해석**
   - 자동 import 경험은 좋아질 수 있다.
   - 공식 인터페이스가 아니므로 Codex 업데이트에 깨질 수 있어 사용하지 않는다.

v1은 1번을 자동 지원하고, `auth.json`이 없는 경우 2번을 안내한다.

## GitHub 인증 소유권의 맥락

GitHub에는 이미 `gh` CLI가 자체 인증 저장소를 가지고 있다. `gh auth login`은
가능하면 OS credential store에 인증을 저장하고, `gh auth status`, `gh auth
switch`, `gh auth token`으로 상태와 계정을 관리할 수 있다. [GitHub CLI 인증 문서](https://cli.github.com/manual/gh_auth_login)

따라서 `account-manager`가 GitHub 토큰을 어떻게 다룰지는 두 모델로 나뉜다.

### 모델 A: `gh`에 위임

```text
account-manager → GH_HOST 설정 → gh의 저장된 계정 사용
```

- `account-manager`는 host와 account 이름만 관리한다.
- GitHub 토큰을 중복 저장하지 않는다.
- 토큰 갱신과 로그아웃은 `gh`가 담당한다.

주의할 점은 `GH_TOKEN`과 `GITHUB_TOKEN` 같은 환경변수가 저장된 `gh` 인증보다
우선할 수 있다는 것이다. 이 모델에서는 전환 시 불필요한 GitHub 토큰 환경변수를
해제해야 한다. [GitHub CLI 환경변수 문서](https://cli.github.com/manual/gh_help_environment)

### 모델 B: `account-manager`가 직접 관리

```text
account-manager → Keychain에서 토큰 조회 → GH_TOKEN/GH_ENTERPRISE_TOKEN 설정
```

- 현재 사용 중인 `GH_TOKEN`/`GH_ENTERPRISE_TOKEN` 방식과 잘 맞는다.
- 호스트별 토큰을 명시적으로 전환할 수 있다.
- 대신 `gh`와 별도로 토큰을 저장해 중복 관리한다.
- 토큰 갱신, 폐기, 만료, `gh` 저장소와의 불일치를 직접 처리해야 한다.

모델 A로 확정한다. `.env`, `GH_TOKEN`, `GH_ENTERPRISE_TOKEN`을
`account-manager`의 인증 입력이나 저장 대상으로 지원하지 않는다. 기존 셸
설정에 해당 변수가 남아 있으면 `gh`의 저장 credential보다 우선될 수 있으므로
사용자가 셸 설정에서 직접 제거해야 한다.

### CommandRunner

외부 명령 실행도 추상화한다.

```text
CommandRunner
├── run(command, args, env, cwd)
├── runCapture(command, args, env, cwd)
└── exists(command)
```

실제 구현은 `gh`, `codex`를 실행하고, 테스트 구현은 미리 정의한 결과를
반환한다. 모든 호출은 다음 원칙을 따른다.

- `shell: true` 금지
- 사용자 입력을 명령 문자열로 조합하지 않음
- secret을 stdout, stderr, error message에 포함하지 않음
- 환경변수는 provider가 생성한 최소 범위만 전달

## 권장 모듈 구조

```text
cmd/account-manager/main.go
internal/
├── app/
│   ├── add.go
│   ├── change.go
│   ├── current.go
│   └── doctor.go
├── config/
│   ├── schema.go
│   ├── load.go
│   └── atomic_write.go
├── credentials/
│   ├── store.go
│   └── macos_keychain.go
├── execution/
│   ├── command_runner.go
│   └── shell_patch.go
└── providers/
    ├── provider.go
    ├── registry.go
    ├── github.go
    └── codex.go
```

`app`은 provider의 구체적인 구현 대신 `Provider` 인터페이스와
`CredentialStore`, `CommandRunner`에만 의존한다.

## 테스트 계층

### 1. 순수 단위 테스트

외부 파일, Keychain, `gh`, `codex`를 사용하지 않는다.

- 프로파일 ID와 provider ID 검증
- TOML 모델 변환
- `--work`, `--personal`, `--only` 범위 해석
- provider별 active 상태 계산
- `mixed` 상태 계산
- 환경변수 patch 생성
- secret이 shell output에 포함되지 않는지 검증

### 2. Provider contract 테스트

모든 provider가 같은 공통 계약을 만족하는지 fake credential store와 fake
command runner로 검증한다.

- 등록 성공
- credential 누락
- 잘못된 설정
- activation plan 생성
- apply 성공
- apply 실패 후 rollback
- 선택하지 않은 provider가 변경되지 않음

### 3. 파일 시스템 통합 테스트

임시 `HOME`, `CODEX_HOME`, 설정 디렉터리를 사용한다.

- 설정 파일 원자적 저장
- 파일 권한 `0600`과 디렉터리 권한 `0700`
- Codex auth cache 임시 파일 작성 후 rename
- 부분 전환에서 비대상 provider 상태 유지
- 설정 저장 실패 시 복구

### 4. 셸 통합 테스트

임시 zsh 프로세스에서 shell wrapper와 patch를 평가한다.

```sh
zsh -f -c 'source wrapper.zsh; account-manager change --work --codex; env'
```

검증 항목:

- `--codex`는 `CODEX_HOME`만 변경
- `--github`는 `GH_*`만 변경
- 전체 전환은 두 provider를 모두 변경
- secret 값이 명령 출력에 직접 나타나지 않음

### 5. macOS 수동 통합 테스트

실제 Keychain과 실제 `gh`/`codex`를 사용하는 테스트는 자동화하지 않고
명시적으로 실행한다.

- `add --work --github`
- `add --work --codex`
- `change --work --github`
- `change --work --codex`
- `change --personal`
- `current`와 `doctor`

테스트에 사용하는 계정과 credential은 별도 테스트 계정이어야 하며, 실제
토큰 값은 로그와 테스트 결과에 남기지 않는다.

## 배포와 설치

초기 개발 단계에서는 다음 흐름을 사용한다.

```sh
go test ./...
go build -o dist/account-manager ./cmd/account-manager
go install ./cmd/account-manager
```

설치 명령은 다음 작업을 제공한다.

```sh
account-manager init zsh
```

이 명령은 기존 `.zshrc`를 백업하지 않고 덮어쓰지 않으며, 필요한 wrapper와
초기화 구문을 추가할지 사용자에게 확인한다.

## 구현 전 확인할 사항

- Codex가 keyring-backed 상태일 때 `add --codex`를 재로그인 방식으로 보완할지
- macOS arm64 단일 바이너리부터 시작할지 universal binary까지 동시에 빌드할지
