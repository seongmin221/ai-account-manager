# 목적과 핵심 모델

## 목적

`account-manager`는 하나의 명령으로 여러 도메인의 계정 프로파일을 등록하고
전환한다. v1에서 지원하는 provider는 `github`와 `codex`이며, 이후 Slack,
AWS, npm 등 provider를 추가할 수 있어야 한다.

`work`와 `personal`은 계정 자체가 아니라 여러 provider 설정을 담는 프로파일이다.
각 provider는 서로 다른 프로파일을 활성화할 수 있다.

```text
github = personal
codex  = work
```

이 상태의 전체 모드는 `mixed`로 표시한다.

## 핵심 불변조건

- 활성 프로파일은 provider별로 독립적으로 관리한다.
- 선택하지 않은 provider의 환경과 상태는 변경하지 않는다.
- 대상 provider 설정이 없거나 유효하지 않으면 전환하지 않는다.
- 전체 전환은 모든 대상 provider의 사전 검증이 성공한 뒤에만 적용한다.
- 전환 중 실패하면 이미 적용한 provider를 역순으로 복구한다.
- 토큰과 Codex 인증 캐시는 일반 설정 파일에 저장하지 않는다.
- 토큰은 로그, 오류 메시지, 프로세스 인자에 노출하지 않는다.
- 활성 상태 변경은 원자적으로 저장한다.

## 상태 계산

`account-manager current`는 provider별 활성 상태를 표시한다.

- 모든 활성 provider가 같은 프로파일이면 그 프로파일명을 전체 모드로 표시한다.
- 둘 이상의 프로파일이 활성화되어 있으면 `mixed`로 표시한다.
- 등록되지 않은 provider는 `unconfigured`로 표시한다.
