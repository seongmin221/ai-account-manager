# CLI 계약과 셸 연동

## 프로파일과 범위

프로파일 선택은 `--work`, `--personal`을 지원하고 내부적으로는
`--profile <id>`로 정규화한다.

```sh
account-manager add --work
account-manager add --work --codex
account-manager add --work --github --host oss.navercorp.com
account-manager add --work --codex --login

account-manager change --work
account-manager change --work --codex
account-manager change --work --github
```

범위가 없으면 대상 프로파일에 등록된 모든 provider를 선택한다.

확장을 위해 다음 형태도 지원한다.

```sh
account-manager change --work --only codex
account-manager change --work --only github,codex
```

`--codex`, `--github`는 `--only`의 편의용 별칭이다.

## `add`

```sh
account-manager add --work [scope]
```

- 선택된 provider만 등록하거나 갱신한다.
- 선택되지 않은 provider 설정은 그대로 둔다.
- 대상 프로파일이 없으면 생성한다.
- 기존 credential을 덮어쓸 때는 확인을 요구한다.
- 모든 provider 등록이 성공한 뒤에만 설정 파일을 저장한다.

## `change`

```sh
account-manager change --work [scope]
```

선택된 provider만 변경한다. `--codex`로 실행하면 GitHub에는 어떤 변경도
발생하지 않는다.

## 셸 연동

실행 파일은 부모 셸의 환경변수를 직접 변경할 수 없다. 현재 셸에 전환 결과를
적용하려면 zsh wrapper가 필요하다.

```sh
account-manager() {
  if [[ "$1" == "change" ]]; then
    eval "$(command account-manager "$@" --shell zsh)" || return
  else
    command account-manager "$@"
  fi
}
```

wrapper가 설치된 상태에서 사용자가 입력하는 명령은 다음과 같다.

```sh
account-manager change --work --codex
```

wrapper가 없는 상태에서 직접 실행하면 provider의 영속 상태와 설정 파일은
변경될 수 있지만 현재 셸의 환경변수는 변경되지 않는다. CLI는 이 경우
현재 셸에 적용되지 않았다는 경고와 다음 조치를 출력한다.

```text
현재 셸에 환경변수가 적용되지 않았습니다.
→ account-manager init zsh를 실행하거나 새 셸을 시작하세요.
```

CLI가 생성하는 셸 코드는 토큰 값을 직접 포함하지 않고 Keychain 또는 외부
credential store를 참조한다. `change --work --codex`는 `CODEX_HOME` 관련
코드만 생성한다.

새 터미널에서도 저장된 활성 상태를 복원할 수 있도록 다음 초기화 명령을
제공한다.

```sh
eval "$(account-manager env --shell zsh)"
```
