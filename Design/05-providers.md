# GitHub·Codex Provider 흐름

## GitHub Provider

### 등록

대상 host는 다음 순서로 결정한다.

1. `--host`
2. 대상 프로파일에 이미 기록된 host
3. 현재 `GH_HOST`
4. 위 값이 모두 없으면 오류를 반환한다.

최초 등록 예:

```sh
account-manager add --work --github --host oss.navercorp.com
account-manager add --personal --github --host github.com
```

현재 계정은 대상 host에 대해 `gh api user`로 확인한다. 현재 host와 대상
host가 다르면 계정이 의도한 것인지 확인을 요구한다.

기본 등록 방식에서는 `gh auth status`로 현재 계정을 확인하고, 계정명과 host만
설정 파일에 기록한다. GitHub credential은 `gh`의 자체 credential store에
위임한다. 토큰은 설정 파일, 출력, 프로세스 인자에 포함하지 않는다.

### 활성화

업무 프로파일:

```text
set:
  GH_HOST=oss.navercorp.com
unset:
  GH_TOKEN
  GITHUB_TOKEN
  GH_ENTERPRISE_TOKEN
  GITHUB_ENTERPRISE_TOKEN
```

개인 프로파일도 `GH_HOST=github.com`을 설정하고 네 가지 토큰 환경변수를 모두
해제한다. 이후 `gh`가 host별로 저장한 credential을 사용한다.

같은 host에 여러 GitHub 계정이 등록된 경우에는 `gh auth switch --hostname`
과 `--user` 계정명을 사용해 대상 계정을 선택한다.

GitHub 전환 계획에는 전환 전의 host/account를 저장한다. `gh auth switch` 또는
host 변경 뒤 다른 provider나 설정 저장이 실패하면 이전 GitHub 계정을 다시
활성화한다.

GitHub Provider는 Codex의 `CODEX_HOME`이나 Codex credential을 변경하지 않는다.

## Codex Provider

### 등록

Codex provider는 현재 `CODEX_HOME`을 다음 순서로 결정한다.

1. `CODEX_HOME` 환경변수
2. 기본값 `~/.codex`

v1에서는 해당 위치의 `auth.json`을 opaque 데이터로 읽어 Keychain에 저장한다.
JSON 내부의 token 필드를 해석하거나 로그에 출력하지 않는다.

`auth.json`이 없는 경우에는 현재 설치가 OS keyring 기반일 수 있으므로 현재
인증을 자동 추출하지 않는다. 대상 Codex home에서 별도 로그인이 필요하다는
오류를 반환한다.

### 활성화

계정별 Codex home:

```text
~/.local/share/account-manager/codex/work
~/.local/share/account-manager/codex/personal
```

1. 현재 active Codex 프로파일의 auth cache를 Keychain에 최신 상태로 저장한다.
2. 대상 credential을 Keychain에서 읽는다.
3. 대상 `CODEX_HOME` 디렉터리를 `0700`으로 생성한다.
4. 임시 파일을 `0600`으로 작성하고 검증한다.
5. `auth.json`을 원자적으로 교체한다.
6. `CODEX_HOME` 환경변수 patch를 생성한다.
7. 새 Codex home으로 `codex login status`를 실행해 확인한다.

뒤 단계가 실패하면 대상 auth cache를 제거하거나 이전 파일로 복구하고,
이전 `CODEX_HOME`을 rollback patch로 반환한다.

Codex 활성화는 GitHub 환경변수에 영향을 주지 않는다.

## 부분 전환 예시

현재 상태:

```text
github = personal
codex  = personal
```

명령:

```sh
account-manager change --work --codex
```

결과:

```text
github = personal   # 변경 없음
codex  = work       # 변경됨
mode   = mixed
```

생성되는 셸 patch에는 `CODEX_HOME`만 포함되고 `GH_HOST`, `GH_TOKEN`,
`GH_ENTERPRISE_TOKEN`은 포함되지 않는다.
