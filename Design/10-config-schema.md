# 설정 스키마와 오류 코드

## 목적

설정 파일은 프로파일, provider 메타데이터, active 상태만 저장한다.
토큰과 Codex 인증 캐시는 저장하지 않는다.

```text
config.toml
├── version
├── active
└── profiles
    └── <profile>
        └── providers
            └── <provider>
```

## v1 스키마

```toml
version = 1

[active]
github = "personal"
codex = "work"

[profiles.work]
display_name = "Work"

[profiles.work.providers.github]
host = "oss.navercorp.com"
account = "work-user"
auth_source = "gh"

[profiles.work.providers.codex]
credential_ref = "codex/work"
codex_home = "~/.local/share/account-manager/codex/work"

[profiles.personal]
display_name = "Personal"

[profiles.personal.providers.github]
host = "github.com"
account = "personal-user"
auth_source = "gh"

[profiles.personal.providers.codex]
credential_ref = "codex/personal"
codex_home = "~/.local/share/account-manager/codex/personal"
```

## 필드 규칙

### Root

| 필드 | 필수 | 규칙 |
| --- | --- | --- |
| `version` | 예 | 현재 값은 `1` |
| `active` | 아니오 | provider ID를 profile ID에 매핑 |
| `profiles` | 아니오 | profile ID를 profile 설정에 매핑 |

파일이 없으면 `init`이 기본 파일을 생성한다. 파일이 존재하지만 TOML이
잘못되었으면 자동으로 덮어쓰지 않는다.

### Profile

| 필드 | 필수 | 규칙 |
| --- | --- | --- |
| `display_name` | 아니오 | 사용자에게 보여줄 이름 |
| `providers` | 아니오 | provider별 설정 테이블 |

profile ID는 다음 정규식을 사용한다.

```text
^[a-z][a-z0-9_-]{0,31}$
```

`work`, `personal`은 기본 프로파일일 뿐이며 사용자 정의 profile도 허용한다.

### Provider 공통 필드

provider 설정은 공통 필드와 provider 전용 필드로 구성한다.

| 필드 | 필수 | 규칙 |
| --- | --- | --- |
| `credential_ref` | provider별 | Keychain 참조. GitHub `gh` 위임 provider에는 없음 |

provider ID도 profile ID와 같은 형식으로 검증한다.

등록되지 않은 provider 테이블은 설정에서 보존할 수 있지만, 해당 provider가
현재 바이너리에 없으면 활성화할 수 없다. `doctor`는 이를 `unknown provider`로
표시한다.

## Provider별 스키마

### GitHub

```toml
[profiles.work.providers.github]
host = "oss.navercorp.com"
account = "work-user"
auth_source = "gh"
```

- `host`: scheme과 path가 없는 hostname
- `account`: `gh auth status`에서 확인한 계정명
- `auth_source`: v1에서는 반드시 `gh`
- 토큰 필드와 `.env` 경로 필드는 허용하지 않음

`GH_ENTERPRISE_URL`은 provider 설정에 저장하지 않는다. Enterprise host는
`host` 필드에 hostname만 저장한다.

### Codex

```toml
[profiles.work.providers.codex]
credential_ref = "codex/work"
codex_home = "~/.local/share/account-manager/codex/work"
```

- `credential_ref`: Keychain에 저장된 opaque auth cache 참조
- `codex_home`: 활성화할 때 사용할 관리 대상 Codex home
- `codex_home`에는 토큰을 설정 파일로 저장하지 않음

v1에서는 `codex_home`을 account-manager가 관리하는 디렉터리 아래로 제한한다.
현재 로그인된 기본 Codex home은 등록 시 읽기 source로만 사용한다.

## Active 상태 규칙

```toml
[active]
github = "personal"
codex = "work"
```

각 active 항목에 대해 다음을 검증한다.

1. provider ID가 유효한가
2. profile ID가 존재하는가
3. 해당 profile에 provider 설정이 있는가
4. provider 설정이 유효한가

active 항목이 없다는 것은 아직 해당 provider가 선택되지 않았다는 뜻이다.
모든 provider가 같은 profile을 가리킬 필요는 없다.

## 설정 로드와 저장

### 로드

1. 파일 존재 여부 확인
2. TOML 파싱
3. version 확인
4. ID와 필드 형식 검증
5. provider별 검증
6. active 참조 검증

지원하지 않는 미래 version은 읽기 전용 명령에서도 임의로 해석하지 않고
명확한 오류를 반환한다.

### 저장

설정 변경은 다음 순서로 원자적으로 저장한다.

1. 기존 설정을 메모리에 로드
2. 변경 적용
3. 전체 스키마 검증
4. 같은 디렉터리에 임시 파일 생성
5. 임시 파일 권한을 `0600`으로 설정
6. 내용을 기록하고 flush/sync
7. 기존 파일을 임시 파일로 atomic rename

설정 디렉터리는 `0700`으로 생성한다. 검증에 실패하면 기존 파일을 변경하지
않는다.

## 명령 계약

설정 자체를 점검할 수 있는 명령을 제공한다.

```sh
account-manager config validate
```

성공 시에는 다음과 같이 출력한다.

```text
Configuration is valid.
Profiles: 2
Providers: github, codex
Active: github=personal, codex=work
```

`--json` 옵션을 사용하면 자동화 가능한 구조화된 결과를 반환한다. secret이나
credential 데이터는 포함하지 않는다.

## 오류 코드

오류 메시지는 사람이 읽을 수 있어야 하며, 자동화에서는 안정적인 오류 코드를
사용한다.

| 코드 | 의미 |
| --- | --- |
| `AM001` | 설정 파일을 찾을 수 없음 |
| `AM002` | TOML 문법 오류 |
| `AM003` | 지원하지 않는 설정 version |
| `AM004` | 잘못된 profile ID |
| `AM005` | 잘못된 provider ID |
| `AM006` | 존재하지 않는 active profile 참조 |
| `AM007` | provider 설정 오류 |
| `AM008` | 알 수 없는 provider |
| `AM009` | credential reference 누락 |
| `AM010` | credential을 Keychain에서 찾을 수 없음 |
| `AM011` | 외부 도구가 설치되지 않음 |
| `AM012` | 외부 인증 상태가 유효하지 않음 |
| `AM013` | provider 활성화 실패 |
| `AM014` | 롤백 실패 |
| `AM015` | 셸 환경 patch 적용 필요 |
| `AM016` | 파일 또는 Keychain 권한 오류 |
| `AM017` | 최초 GitHub 등록에 host가 필요함 |

## 프로세스 종료 코드

```text
0  성공
2  사용법 또는 인자 오류
3  설정 오류
4  credential 또는 인증 오류
5  provider 활성화 오류
6  롤백 실패
7  외부 도구/시스템 오류
```

오류 출력 예:

```text
error[AM010]: Codex credential 'codex/work' was not found.
hint: run account-manager add --work --codex
```

토큰, auth JSON, Keychain secret은 오류 출력에 포함하지 않는다.

최초 GitHub 등록 오류 예:

```text
error[AM017]: GitHub host is required for the first registration.
hint: account-manager add --work --github --host oss.navercorp.com
```
