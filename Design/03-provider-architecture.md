# Provider 아키텍처와 트랜잭션

## Provider 인터페이스

핵심 명령은 provider의 구체적인 구현을 알지 않고 provider registry를 순회한다.

```text
Provider
├── id()
├── inspectCurrent()
├── capture(targetProfile)
├── validate(profileProviderConfig)
├── planActivate(profileProviderConfig)
├── apply(plan)
└── rollback(plan)
```

`inspectCurrent`는 현재 셸과 로컬 상태에서 provider의 현재 계정을 확인한다.

```text
identity = {
  provider: "github",
  host: "github.com",
  account: "personal-user"
}
```

`capture`는 provider의 인증 저장 방식에 따라 현재 인증 상태를 저장하거나,
외부 credential store의 계정 메타데이터만 반환한다. GitHub처럼 자체 credential
store를 사용하는 provider는 credential reference를 만들지 않는다.

`planActivate`는 사전 검증과 변경 계획을 만들고, `apply`는 계획을 적용하며
롤백에 필요한 이전 상태를 반환한다.

환경변수 변경은 provider별 patch로 표현한다.

```text
EnvPatch {
  set: { name: value-or-secret-reference }
  unset: [name]
}
```

provider가 담당하지 않는 환경변수는 변경하지 않는다.

## Provider 범위 해석

```text
target_profile = --profile 또는 --work/--personal
scope = --only, --codex, --github 또는 대상 프로파일의 전체 provider
```

범위가 명시된 경우 선택된 provider만 처리한다.

## `add` 트랜잭션

```text
add --work --codex
```

1. 대상 프로파일과 provider 범위를 확인한다.
2. 현재 provider 상태를 확인한다.
3. 현재 인증 캐시의 존재와 형식을 검증한다.
4. 현재 계정 상태와 저장 대상 프로파일을 보여주고 확인한다.
5. provider 방식에 따라 인증 데이터를 임시 credential reference에 저장하거나
   외부 credential store 계정 메타데이터를 준비한다.
6. provider 메타데이터를 구성한다.
7. 설정 파일을 검증하고 원자적으로 저장한다.
8. 임시 credential reference가 있는 경우 최종 reference로 확정한다.

실패하면 설정 파일과 최종 credential reference를 변경하지 않고 임시
credential도 삭제한다.

대상 provider가 아직 active로 등록되어 있지 않다면 `add`는 현재 인증이
대상 프로파일에 해당한다고 간주해 active 상태를 초기화할 수 있다. 이미
active 상태가 있으면 `add`만으로 활성 프로파일을 변경하지 않는다.

## `change` 트랜잭션

### 사전 검증

```text
change --work --codex
```

적용 전에 다음을 모두 검증한다.

- 대상 프로파일과 선택된 provider 설정의 존재
- provider가 요구하는 credential reference 또는 외부 credential store 계정의 존재
- provider별 설정값의 유효성
- 현재 active provider를 저장해야 할 경우 현재 credential의 접근성
- provider가 생성할 경로와 파일에 대한 접근성

하나라도 실패하면 어떤 provider도 적용하지 않는다.

### 적용 순서

```text
1. 현재 provider 상태 snapshot
2. 모든 provider의 activation plan 생성
3. provider plan을 순서대로 적용
4. active 상태를 임시 설정 파일에 기록
5. 설정 파일을 원자적으로 교체
6. 셸 환경 patch 출력
```

각 provider의 `apply`는 이전 상태를 포함한 rollback 정보를 반환한다. 적용
중 실패하면 이미 적용된 provider를 역순으로 rollback한다. GitHub의 경우
`gh auth switch` 이전의 host/account를 rollback 정보에 포함하고, Codex의
경우 대상 auth cache 파일과 이전 `CODEX_HOME` 상태를 포함한다.

설정 파일 교체가 실패한 경우에도 provider rollback을 수행한다.
