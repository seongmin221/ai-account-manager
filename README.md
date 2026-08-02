# ai-account-manager
GitHub와 Codex의 업무용·개인용 계정 프로파일을 안전하게 등록하고 전환하는
macOS용 CLI입니다. v1은 Go 단일 바이너리, zsh, `gh`, `codex`, macOS Keychain을
기준으로 합니다.

## 개발

Go 단일 바이너리로 빌드합니다.

```sh
go build -o account-manager ./cmd/account-manager
./account-manager --help
```

주요 명령은 `init`, `add`, `change`, `current`, `list`, `doctor`,
`config validate`, `env --shell zsh`입니다. `change --work --codex`처럼
provider를 선택해 부분 전환할 수 있으며, 인증 secret은 설정 파일에 저장하지
않습니다.

구체적인 기능과 구현 순서는 [Design/index.md](Design/index.md)를 참고하세요.
