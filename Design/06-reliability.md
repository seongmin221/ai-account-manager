# 실패 복구와 상태 점검

## 전환 실패

Provider는 실패 유형을 구분해 반환한다.

```text
NotRegistered
CredentialMissing
InvalidConfiguration
AuthenticationUnavailable
PermissionDenied
ActivationFailed
RollbackFailed
```

`RollbackFailed`가 발생하면 원래 active 상태를 설정 파일에 유지하고, 어떤
provider가 복구되지 않았는지 명시한다. credential이나 auth cache의 내용은
오류 메시지에 포함하지 않는다.

## 부분 전환 규칙

현재 상태가 다음과 같을 때:

```text
github = personal
codex  = work
```

`account-manager change --personal --github`는 GitHub만 personal로 바꾸고
Codex는 work로 유지한다.

provider별 변경이므로 대상 provider의 rollback만 수행하면 된다.

## 상태 점검

```sh
account-manager current
account-manager doctor
```

`doctor`는 credential 값 자체를 출력하지 않고 provider별 상태만 표시한다.

```text
github: registered / credential available / host reachable
codex: registered / credential available / auth valid
```
