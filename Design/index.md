# Account Manager Design

`account-manager` 설계 문서의 목차입니다.

## 문서

1. [설계 배경과 논의 과정](00-context.md)
2. [목적과 핵심 모델](01-overview.md)
3. [설정과 자격 증명 저장](02-storage.md)
4. [Provider 아키텍처와 트랜잭션](03-provider-architecture.md)
5. [CLI 계약과 셸 연동](04-cli.md)
6. [GitHub·Codex Provider 흐름](05-providers.md)
7. [실패 복구와 상태 점검](06-reliability.md)
8. [구현 로드맵](07-roadmap.md)
9. [구현 기술과 테스트 전략](08-implementation.md)
10. [CLI 사용 흐름과 최초 설정](09-cli-workflows.md)
11. [설정 스키마와 오류 코드](10-config-schema.md)
12. [구현 순서](11-implementation-order.md)

## 설계 원칙 요약

- `work`, `personal`은 여러 provider 설정을 담는 프로파일이다.
- 활성 프로파일은 provider별로 독립적으로 관리한다.
- 선택하지 않은 provider는 부분 전환에서 변경하지 않는다.
- 인증 정보는 일반 설정 파일이 아닌 OS 자격 증명 저장소에 보관한다.
- provider 추가가 핵심 로직 변경으로 이어지지 않도록 registry와 공통 인터페이스를 사용한다.
- 전체 전환은 사전 검증 후 원자적으로 적용하고, 실패 시 롤백한다.

## 현재 상태

현재 저장소에는 설계 문서와 단일 바이너리의 초기 구현 골격이 있다. 실제 기능
구현은 [구현 순서](11-implementation-order.md)의 단계별 완료 조건을 따른다.
