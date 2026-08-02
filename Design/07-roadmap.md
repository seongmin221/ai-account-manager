# 구현 로드맵

## 초기 구현 범위

1. provider registry와 공통 데이터 타입 구현
2. TOML 설정 로드·검증·원자적 저장 구현
3. macOS Keychain 기반 `CredentialStore` 구현
4. `current`와 `list` 구현
5. GitHub Provider 구현
6. Codex Provider 구현
7. zsh shell output 및 wrapper 설치 구현
8. 부분 전환과 롤백 테스트 작성
9. 배포 방식과 사용자 설치 흐름 확정

## 이후 확장

- `--only <provider>` 기반 provider 범위 확장
- Slack, AWS, npm 등 추가 Provider
- OS별 credential store 구현
- Codex keyring-backed 인증의 직접 등록 지원
- `doctor`의 provider별 진단 확장
