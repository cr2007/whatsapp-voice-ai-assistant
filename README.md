<!-- omit from toc -->
# WhatsApp Voice Message Transcription and AI Assistant

<div align="center">
    <a alt="Open in GitHub Codespaces" href="https://codespaces.new/cr2007/whatsapp-voice-ai-assistant">
        <img src="https://github.com/codespaces/badge.svg" />
    </a>
    <br>
    <a href="https://groq.com" target="_blank" rel="noopener noreferrer">
      <img
        src="https://console.groq.com/powered-by-groq-dark.svg"
        alt="Powered by Groq for fast inference."
        width=10%
      />
    </a>
    <br>
    <a href="https://go.dev">
        <img alt="Go" title="Go Programming Language" src="https://img.shields.io/badge/Go-informational?style=flat&logo=go&logoColor=white&color=00add8">
    </a>
    <a href="https://python.org/">
        <img alt="Python" title="Python Programming Language" src="https://img.shields.io/badge/Python-informational?style=flat&logo=python&logoColor=white&color=3776ab">
    </a>
    <a href="https://www.whatsapp.com/">
        <img alt="WhatsApp" title="WhatsApp" src="https://img.shields.io/badge/WhatsApp-informational?style=flat&logo=WhatsApp&logoColor=white&color=25D366">
    </a>
    <br>
    <a href="https://deepwiki.com/cr2007/whatsapp-voice-ai-assistant">
        <img src="https://deepwiki.com/badge.svg" alt="Ask DeepWiki">
    </a>
</div>
<br>

> [!WARNING]
> **For educational purposes only.** This project uses the unofficial WhatsApp multi-device protocol via [whatsmeow](https://github.com/tulir/whatsmeow).
> Running automated bots on a personal WhatsApp account violates [WhatsApp's Terms of Service](https://www.whatsapp.com/legal/terms-of-service)
> and risks having your account permanently banned.
>
> **Do not use this in production or at any scale.**
>
> If you need a production-grade WhatsApp integration, use the official
> [WhatsApp Business Platform (Cloud API)](https://developers.facebook.com/docs/whatsapp/cloud-api/)
> provided by Meta.

A microservices-based WhatsApp bot that transcribes voice messages and audio files, with optional AI-powered responses via Groq.

<!-- omit from toc -->
## Table of Contents
- [Features](#features)
- [Technologies Used](#technologies-used)
- [Setup](#setup)
  - [Prerequisites](#prerequisites)
  - [Environment Variables](#environment-variables)
  - [Installation](#installation)
    - [Go](#go)
    - [Python](#python)
- [Usage](#usage)
  - [Trigger Commands](#trigger-commands)
  - [Optional Flag](#optional-flag)
- [Configuration](#configuration)
- [Testing](#testing)
- [API Endpoint](#api-endpoint)
- [Contributing](#contributing)
- [Acknowledgements](#acknowledgements)

## Features

- Transcribes WhatsApp voice notes **and** regular audio files
- Two trigger modes — transcribe-only, or transcribe and send to Groq for an AI response
- Early validation: replies with a usage hint if the quoted message is not audio
- Configurable transcription server URL and message prefix
- No GCC required — uses a pure-Go SQLite driver (`modernc.org/sqlite`)

## Technologies Used

- Go
- Python
- Flask
- SQLite (via `modernc.org/sqlite` — no CGO)
- [whatsmeow](https://github.com/tulir/whatsmeow) — unofficial WhatsApp multi-device library
- [Faster-Whisper](https://github.com/SYSTRAN/faster-whisper) — local speech recognition
- [Groq API](https://console.groq.com) — fast LLM inference

| ![Overall App Logic Flow](./images/Logic-Flow.png) | ![System Architecture](./images/Overall-System-Architecture.png) |
| -------------------------------------------------- | ---------------------------------------------------------------- |

## Setup

### Prerequisites

- [Go 1.25+](https://go.dev)
- [Python 3.x](https://python.org) with [uv](https://docs.astral.sh/uv)
- A Groq API key from [console.groq.com/keys](https://console.groq.com/keys)

> [!NOTE]
> GCC is no longer required. The project switched from `mattn/go-sqlite3`
> (CGO) to `modernc.org/sqlite` (pure Go), so it builds on any platform
> without a C compiler.

### Environment Variables

Copy `sample.env` to `.env` and fill in your values:

```env
GROQ_API_KEY=your_key_here
TRANSCRIBE_URL=http://<flask-server-ip>:5000/transcribe
```

| Variable | Required | Description |
|----------|----------|-------------|
| `GROQ_API_KEY` | For `1> transcribe` | Groq API key for AI responses |
| `TRANSCRIBE_URL` | No | Flask server URL. Defaults to `http://127.0.0.1:5000/transcribe` |

### Installation

#### Go

Install dependencies:

```shell
go mod tidy
```

Build the binary:

```shell
# Linux / macOS
go build -o whatsapp-bot .

# Windows
go build -o whatsapp-bot.exe .
```

> [!NOTE]
> Use `go build` rather than `go run`. On some platforms (e.g. Windows with
> Application Control policies), `go run` compiles to a temp directory that
> may be blocked from executing.

#### Python

Install dependencies via [uv](https://docs.astral.sh/uv):

```shell
uv sync
```

## Usage

1. Start the Flask transcription server:

   ```shell
   uv run main.py
   ```

   If the bot runs on a different machine, note the server's IP address and
   set it as `TRANSCRIBE_URL` in your `.env`.

2. Start the bot:

   ```shell
   # Linux / macOS
   ./whatsapp-bot

   # Windows
   .\whatsapp-bot.exe
   ```

3. On first run, scan the QR code displayed in the terminal to log into WhatsApp.

4. Reply to a voice note or audio file with one of the trigger commands below.

### Trigger Commands

| Command | Behaviour |
|---------|-----------|
| `1> transcribe` | Transcribe the audio, then send the transcript to Groq for an AI response |
| `2> transcribe` | Transcribe the audio only. No AI response sent |

The trigger can appear anywhere in the message (e.g. `can you 1> transcribe this?`).

If you reply to a message that is not a voice note or audio file, the bot will send a hint explaining what to do instead.

### Optional Flag

```shell
./whatsapp-bot --message-head "Transcript: "
```

`--message-head` sets the text prepended to every transcription reply.
Defaults to `*Transcript:*\n> ` (bold label with a block-quote indent).

## Configuration

| Setting | How to change |
|---------|---------------|
| Transcription server URL | Set `TRANSCRIBE_URL` in `.env` |
| Groq AI model | Edit the `Model` field in `groq/groq.go` |
| Whisper model size | Edit `model_name` in `main.py` (default: `medium.en`) |
| Transcription reply prefix | Pass `--message-head` flag at startup |

## Testing

Run the full test suite:

```shell
go test ./...
```

For verbose output:

```shell
go test ./... -v
```

18 tests are included across two packages:

| Test | Package | Cases |
|------|---------|-------|
| `TestParseTrigger` | `main` | 9 - trigger command parsing edge cases |
| `TestGetTranscription` | `main` | 4 - HTTP client against a local test server |
| `TestHtmlToWhatsAppFormat` | `groq` | 10 - HTML-to-WhatsApp text conversion |
| `TestSendPostRequestGroq` | `groq` | 4 - Groq API client against a local test server |

All tests run without network access or external services.

## API Endpoint

The transcription service exposes a single endpoint:

```
POST /transcribe
```

- **Request body:** raw binary audio data (`application/octet-stream`)
- **Response:** JSON object

```json
{
  "transcription": "the transcribed text",
  "language": "en"
}
```

## Contributing

1. Create a [new issue](https://github.com/cr2007/whatsapp-voice-ai-assistant/issues/new/choose)
2. [Fork the repository](https://github.com/cr2007/whatsapp-voice-ai-assistant/fork)
3. Create a new branch
4. Commit your changes
5. Push to the branch
6. Create a new Pull Request

## Acknowledgements

1. [YASSERMD/whatsmeow-groq](https://github.com/YASSERRMD/whatsmeow-groq)
2. [hoehermann/whatsmeow-transcribe](https://github.com/hoehermann/whatsmeow-transcribe)
