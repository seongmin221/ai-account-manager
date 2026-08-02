# CLI 사용 흐름과 최초 설정

## 목표

사용자는 인증 토큰을 직접 입력하거나 설정 파일에 복사하지 않고, 현재 로그인
상태를 프로파일에 등록한 뒤 필요한 provider만 전환할 수 있어야 한다.

```text
설치 → init → provider별 add → change → current/doctor
```

## 1. 최초 초기화

```sh
account-manager init
```

`init`은 다음을 수행한다.

1. 운영체제와 셸을 확인한다.
2. `gh`와 `codex` 실행 파일의 존재를 확인한다.
3. `~/.config/account-manager` 디렉터리를 생성한다.
4. 초기 `config.toml`이 없으면 생성한다.
5. macOS Keychain 접근이 가능한지 확인한다.
6. zsh wrapper를 설치할지 사용자에게 묻는다.

`init`은 계정을 자동으로 등록하거나 로그인하지 않는다.

셸 연동은 별도 명령으로도 실행할 수 있다.

```sh
account-manager init zsh
```

기존 `.zshrc`를 덮어쓰지 않고, 중복 wrapper를 추가하지 않는다.

## 2. GitHub 계정 등록

GitHub 인증은 `gh`에 위임한다. 따라서 먼저 각 호스트의 계정이 `gh`에 등록되어
있어야 한다.

```sh
gh auth login --hostname github.com
gh auth login --hostname oss.navercorp.com
```

그 후 프로파일에 등록한다.

```sh
account-manager add --work --github --host oss.navercorp.com
```

등록 흐름:

1. 대상 프로파일과 provider를 확인한다.
2. 대상 host를 결정한다.
3. `gh auth status`로 해당 host의 계정을 확인한다.
4. 계정명과 host를 사용자에게 보여준다.
5. 확인 후 프로파일 메타데이터를 저장한다.

저장하는 것은 다음 정보뿐이다.

```text
host: oss.navercorp.com
account: work-user
auth_source: gh
```

GitHub 토큰, `.env` 값, `gh auth token` 출력값은 account-manager가 저장하지
않는다.

### Host 결정 규칙

1. `--host`
2. 대상 프로파일에 이미 등록된 host
3. 현재 `GH_HOST`
4. 위 값이 모두 없으면 `AM017` 오류를 반환한다.

profile에 GitHub host가 처음 등록되는 경우에는 `--host`를 반드시 지정한다.

현재 로그인된 host와 대상 host가 다르면 실수 가능성이 있으므로 확인을 다시
요구한다.

## 3. Codex 계정 등록

### 파일 기반 인증

현재 Codex가 기본 `~/.codex/auth.json` 또는 지정된 `CODEX_HOME`의 auth cache를
사용하는 경우:

```sh
account-manager add --work --codex
```

등록 흐름:

1. 현재 Codex home을 확인한다.
2. `auth.json`의 존재와 권한을 확인한다.
3. `codex login status`로 로그인 상태를 확인한다.
4. 현재 상태와 대상 프로파일을 사용자에게 보여준다.
5. auth cache를 해석하지 않은 opaque 데이터로 Keychain에 저장한다.
6. 대상 프로파일의 Codex home 경로를 기록한다.

### Keyring 기반 인증

현재 auth cache 파일이 없으면 account-manager가 Codex 내부 Keychain을 추출하지
않는다. 대신 다음과 같은 안내를 출력한다.

```text
현재 Codex 인증 캐시를 파일에서 찾을 수 없습니다.
work 프로파일에서 Codex 로그인을 시작하려면 다음 명령을 실행하세요:

CODEX_HOME="$HOME/.local/share/account-manager/codex/work" codex login
```

로그인 후 다시 `account-manager add --work --codex`를 실행하거나,
`--login` 옵션을 통해 이 흐름을 한 명령으로 시작할 수 있다.

## 4. `add`의 active 상태 처리

`add`는 기본적으로 등록만 수행하고 다른 provider를 변경하지 않는다.

예외적으로 해당 provider의 active 상태가 아직 없으면 현재 인증이 대상
프로파일에 해당한다고 보고 초기 active 상태를 설정할 수 있다.

```text
active.github가 없음
add --work --github 실행
→ active.github = work
```

이미 active 상태가 있으면 `add`만으로 활성 프로파일을 바꾸지 않는다. 등록과
전환을 한 번에 하고 싶다면 명시적으로 다음을 사용한다.

```sh
account-manager add --work --codex --activate
```

`--activate`는 등록 성공 후 선택한 provider에 대해 `change`를 수행한다.

## 5. 전체 전환

```sh
account-manager change --work
```

흐름:

1. work에 등록된 provider 목록을 찾는다.
2. 모든 provider와 credential을 사전 검증한다.
3. GitHub는 `GH_HOST`와 `gh auth switch`를 준비한다.
4. Codex는 work `CODEX_HOME`과 auth cache를 준비한다.
5. 모든 준비가 성공하면 적용한다.
6. active 상태를 원자적으로 저장한다.
7. zsh 환경 patch를 출력한다.

하나라도 준비에 실패하면 어떤 provider도 변경하지 않는다.

## 6. 부분 전환

```sh
account-manager change --work --codex
```

Codex만 work로 전환한다.

```text
github = personal  # 유지
codex  = work      # 변경
mode   = mixed
```

GitHub 관련 `GH_HOST`, GitHub 계정 선택, GitHub credential에는 접근하지 않는다.

반대로 다음 명령은 GitHub만 변경한다.

```sh
account-manager change --work --github
```

## 7. 상태 확인

```sh
account-manager current
```

예상 출력:

```text
Overall mode: mixed

github
  profile: personal
  host: github.com
  account: personal-user
  source: gh

codex
  profile: work
  home: ~/.local/share/account-manager/codex/work
  auth: available
```

credential 값이나 auth cache 내용은 출력하지 않는다.

등록된 모든 프로파일은 다음으로 확인한다.

```sh
account-manager list
```

## 8. 진단

```sh
account-manager doctor
```

진단 항목:

- `gh`와 `codex` 실행 가능 여부
- 현재 셸 wrapper 설치 여부
- 설정 파일 형식과 권한
- provider별 프로파일 등록 여부
- GitHub host별 `gh` 로그인 상태
- Codex credential reference 존재 여부
- Codex home과 auth cache 권한
- 현재 셸의 `GH_TOKEN`, `GH_ENTERPRISE_TOKEN` 등 충돌 변수 존재 여부

진단은 secret의 값이 아니라 존재 여부와 상태만 출력한다.

## 9. 기존 `.env` 사용자 전환

account-manager는 `.env`를 읽지 않는다. 기존 환경에서는 다음 순서로 전환한다.

1. `gh auth login --hostname ...`으로 각 GitHub 계정을 `gh`에 등록한다.
2. `account-manager add --work --github --host oss.navercorp.com`과
   `add --personal --github --host github.com`을 실행한다.
3. Codex 계정을 provider별로 등록한다.
4. 셸 설정에서 `GH_TOKEN`, `GITHUB_TOKEN`, `GH_ENTERPRISE_TOKEN`,
   `GITHUB_ENTERPRISE_TOKEN`, `GH_HOST`를 제거한다.
5. `GH_ENTERPRISE_URL`이 다른 도구에서 필요하면 유지한다.
6. 새 셸을 열고 `account-manager current`로 확인한다.

`GH_ENTERPRISE_URL`은 account-manager가 사용하지 않는 외부 도구용 변수로
취급한다.

## 10. 주요 오류 메시지

사용자가 다음 행동을 바로 알 수 있도록 오류에 해결 방법을 포함한다.

```text
gh가 설치되어 있지 않습니다.
→ GitHub CLI를 설치한 뒤 다시 실행하세요.

github.com에 로그인된 계정이 없습니다.
→ gh auth login --hostname github.com 을 실행하세요.

Codex auth cache를 찾을 수 없습니다.
→ 지정된 CODEX_HOME에서 codex login을 실행하세요.

work 프로파일에 codex provider가 등록되지 않았습니다.
→ account-manager add --work --codex 를 실행하세요.
```

오류 메시지에는 토큰, auth JSON, Keychain secret을 포함하지 않는다.
