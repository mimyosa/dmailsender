# dMailSender

**SMTP 메일 대량 발송 데스크톱 애플리케이션**

> 레거시 Java/Swing 기반 dmailer를 Go + Wails + React로 재구축한 모던 크로스 플랫폼 앱입니다.

## Latest Version

**v0.5** — [Releases](https://github.com/mimyosa/dmailsender/releases)

## Features

- **SMTP 메일 발송** — TLS (STARTTLS) / SSL (Implicit) / 평문 지원, TLS 버전 선택 (1.0~1.3)
- **다중 스레드 전송** — 최대 50개 동시 전송, 전송 간격(ms) 설정
- **넘버링** — From/To/Subject 자동 증분 (예: `user001@example.com` → `user002@...`)
- **타임스탬프** — Subject에 발송 시간 자동 추가
- **EML 모드** — `.eml` 파일 로드, MIME 디코딩 프리뷰, 대량 전송
- **파일 첨부** — Input 모드에서 다중 파일 첨부 지원
- **커스텀 헤더** — Key-Value 형태로 추가 헤더 설정 (Input/EML 모두 지원)
- **실시간 SMTP 로그** — 클라이언트(→)/서버(←) 세션 라인 표시
- **연결 테스트** — TCP + TLS 핸드셰이크 검증
- **인증서 검증 건너뛰기** — 자체 서명 인증서 환경 지원
- **다크/라이트 테마** — Catppuccin 기반, 설정 저장
- **설정 저장/불러오기** — JSON 기반 설정, 비밀번호는 OS 키체인 저장
- **버전 확인** — GitHub Releases API 연동
- **키보드 단축키** — Ctrl+Enter(전송), Ctrl+S(저장), F5(전송), Ctrl+L(로그 초기화)

## Screenshots

<!-- 스크린샷 이미지를 추가하세요 -->
<!-- ![dMailSender Screenshot](docs/screenshot.png) -->

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.22 |
| Desktop Framework | [Wails v2](https://wails.io/) |
| Frontend | React 18 + TypeScript (Vite) |
| SMTP | [go-mail](https://github.com/wneessen/go-mail) |
| Keychain | [go-keyring](https://github.com/zalando/go-keyring) |

## Prerequisites

- **Go** 1.22+
- **Node.js** 18+ & npm
- **Wails CLI** v2

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### Platform-specific

| OS | Requirement |
|----|------------|
| Windows | WebView2 (Windows 10/11 기본 포함) |
| macOS | Xcode Command Line Tools |
| Linux | `gtk3`, `webkit2gtk` 개발 패키지 |

## Build

### Production Build

```bash
cd dmailsender
wails build
```

빌드 완료 후 실행 파일이 생성됩니다:

```
build/bin/dMailSender.exe      # Windows
build/bin/dMailSender          # macOS / Linux
```

### Development Mode

```bash
wails dev
```

핫 리로드가 지원되며, 브라우저 DevTools로 디버깅할 수 있습니다.

### Clean Build

```bash
wails build -clean
```

## Project Structure

```
dmailsender/
├── main.go                    # Wails 앱 진입점, 윈도우 설정
├── app.go                     # AppService: 프론트엔드 바인딩 브릿지
├── core/
│   ├── models.go              # 공유 데이터 구조체
│   ├── config.go              # JSON 설정 로드/저장 + 키체인
│   ├── sender.go              # SMTP 전송, EML 파싱, 연결 테스트
│   ├── worker.go              # 고루틴 풀, 동시 전송 제어
│   ├── validate.go            # 입력 검증 (필수값 + 범위)
│   └── version.go             # GitHub Releases API 버전 확인
├── frontend/
│   ├── src/
│   │   ├── App.tsx            # 메인 앱: 툴바, 레이아웃, 전송 로직
│   │   ├── components/
│   │   │   ├── SettingsPanel.tsx   # 접이식 사이드바 설정
│   │   │   ├── EditorHeader.tsx    # 봉투(From/To) + 콘텐츠(Subject/Body)
│   │   │   └── BottomPanel.tsx     # 하단 패널: Results + SMTP Log
│   │   └── style.css          # 다크/라이트 테마 CSS
│   └── package.json
└── wails.json
```

## macOS Note

macOS에서 다운로드한 앱 실행 시 "손상되었습니다" 메시지가 나타나면, 터미널에서 다음 명령을 실행하세요:

```bash
xattr -cr dMailSender.app
```

이후 우클릭 → 열기로 실행할 수 있습니다.

## Configuration

설정 파일 위치:

| OS | Path |
|----|------|
| Windows | `%APPDATA%\dMailSender\config.json` |
| macOS | `~/Library/Application Support/dMailSender/config.json` |
| Linux | `~/.config/dMailSender/config.json` |

비밀번호는 OS 키체인에 저장됩니다 (Windows DPAPI / macOS Keychain / Linux libsecret).

## License

MIT License

## Author

**mimyosa** — [GitHub](https://github.com/mimyosa)
