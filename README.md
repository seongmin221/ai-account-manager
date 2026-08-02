# ai-account-manager
여러 개의 codex 계정을 빠르게 전환할 수 있도록 돕는 cli

## 개발

Go 단일 바이너리로 빌드합니다.

```sh
go build -o account-manager ./cmd/account-manager
./account-manager --help
```

구체적인 기능과 구현 순서는 [Design/index.md](Design/index.md)를 참고하세요.
