# 설정과 자격 증명 저장

## 설정 파일

기본 경로:

```text
~/.config/account-manager/config.toml
```

설정 파일에는 메타데이터와 OS 자격 증명 저장소의 참조만 기록한다.

```toml
version = 1

[active]
github = "personal"
codex = "work"

[profiles.work.providers.github]
host = "oss.navercorp.com"
account = "work-user"

[profiles.work.providers.codex]
credential_ref = "codex/work"
codex_home = "~/.local/share/account-manager/codex/work"

[profiles.personal.providers.github]
host = "github.com"
account = "personal-user"

[profiles.personal.providers.codex]
credential_ref = "codex/personal"
codex_home = "~/.local/share/account-manager/codex/personal"
```

### 프로파일과 provider

- 프로파일 ID는 `work`, `personal`처럼 소문자 영숫자와 `-`, `_`를 사용한다.
- provider ID도 동일한 규칙을 사용한다.
- 프로파일은 일부 provider만 포함할 수 있다.
- `active`에는 실제로 등록된 provider/profile 조합만 기록한다.
- 같은 profile 안의 provider들은 서로 다른 계정이어도 된다.

## 자격 증명 저장소

credential을 직접 파일에 저장해야 하는 provider는 `CredentialStore`를 사용한다.
GitHub처럼 자체 credential store를 가진 provider는 기본적으로 해당 provider에
저장을 위임할 수 있다.

```text
CredentialStore
├── put(ref, secret)
├── get(ref)
├── exists(ref)
└── delete(ref)
```

v1의 macOS 구현은 Keychain을 사용한다. 저장 항목은 다음처럼 구분한다.

```text
service: account-manager
account: codex/work
account: codex/personal
```

설정 파일의 `credential_ref`는 Keychain 항목의 논리적 참조일 뿐이며 실제
토큰이나 인증 JSON을 포함하지 않는다.

GitHub provider는 `gh` credential store에 위임한다. `account-manager`는 GitHub
토큰을 저장하거나 `.env`를 읽지 않고, host와 계정 선택만 관리한다.
