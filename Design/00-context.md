# 설계 배경과 논의 과정

<details>
<summary>1. 저장소 접근 가능 여부 확인</summary>

처음에는 `seongmin221/ai-account-manager` 저장소에 현재 환경에서 접근할 수
있는지 확인했다.

- `git`과 `gh`가 설치되어 있는지 확인했다.
- `git ls-remote`로 GitHub 저장소의 원격 refs를 조회했다.
- GitHub 저장소의 `main` 브랜치와 최신 커밋을 확인했다.
- `gh`는 기본 호스트가 사내 GitHub Enterprise로 설정되어 있었고, GitHub.com을
  명시했을 때 별도 인증이 필요하다는 것을 확인했다.

결과적으로 저장소는 `git` 기준으로 접근 가능했고, 프로젝트는 README만 있는
초기 상태였다.

</details>

<details>
<summary>2. GitHub 인증 변수 분리</summary>

GitHub.com에서 `gh`를 사용할 때 401이 발생하지 않도록 GitHub 호스트별 인증을
분리했다.

- GitHub.com은 `GH_TOKEN`을 사용한다.
- GitHub Enterprise는 `GH_ENTERPRISE_TOKEN`을 사용한다.
- `GH_HOST`로 현재 대상 호스트를 명시한다.
- `~/.env`는 자동으로 읽히지 않으므로 셸에서 source해야 한다.

이 단계에서 업무용 GitHub Enterprise와 개인용 GitHub.com을 모드로 나누는
기본 방향을 정했다.

</details>

<details>
<summary>3. 업무 모드와 개인 모드에 Codex 계정 추가</summary>

단순히 GitHub 환경변수만 바꾸는 것을 넘어, 모드 전환 시 Codex 계정도 함께
전환하는 요구사항을 정의했다.

```sh
account-manager add --work
account-manager change --work
```

`add`는 현재 로그인된 GitHub와 Codex 계정을 해당 프로파일에 등록하고,
`change`는 저장된 계정을 활성화한다.

계정 정보는 일반 설정 파일이나 `.env`에 직접 저장하지 않고 기기의 안전한
자격 증명 저장소를 사용하기로 했다. Codex는 계정별 `CODEX_HOME`을 분리해
인증 상태가 섞이지 않도록 하는 방향을 잡았다.

</details>

<details>
<summary>4. 여러 도메인으로 확장 가능한 구조</summary>

GitHub와 Codex 외에도 향후 Slack, AWS, npm 등 다른 계정 도메인이 추가될 수
있다는 요구사항을 반영했다.

`work`와 `personal`을 특정 서비스의 계정으로 취급하지 않고, 여러 provider의
설정을 담는 프로파일로 정의했다.

```text
work
├── github
├── codex
└── aws (future)
```

각 도메인은 공통적인 `inspect`, `capture`, `validate`, `activate`, `rollback`
동작을 구현하는 provider가 된다.

</details>

<details>
<summary>5. 도메인별 부분 전환</summary>

GitHub는 업무 계정이지만 Codex는 개인 계정일 수 있으므로 provider 하나만
전환할 수 있도록 확장했다.

```sh
account-manager change --work --codex
```

이 명령은 Codex만 업무 계정으로 바꾸고 GitHub 환경은 변경하지 않는다.

이 요구사항 때문에 활성 상태를 단일 `active_profile` 값으로 저장하지 않고,
provider별로 저장하도록 변경했다.

```text
github = personal
codex  = work
```

이처럼 provider별 활성 프로파일이 다르면 전체 상태는 `mixed`로 표시한다.

</details>

<details>
<summary>6. 현재 설계로 확정된 방향</summary>

- 프로파일: `work`, `personal` 및 향후 사용자 정의 프로파일
- provider: `github`, `codex` 및 향후 확장 provider
- 활성 상태: provider별 프로파일 매핑
- 인증 정보: OS Keychain 등 안전한 자격 증명 저장소
- Codex 상태: 프로파일별 `CODEX_HOME`
- 전환 범위: 전체 provider 또는 특정 provider
- 셸 반영: zsh wrapper와 shell environment patch
- 안정성: 사전 검증, 원자적 상태 저장, 실패 시 롤백

</details>
