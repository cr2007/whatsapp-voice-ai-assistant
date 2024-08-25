<!-- omit from toc -->
# WhatsApp Voice Message Transcription and AI Assistant


<div align="center">
    <a alt="Open in GitHub Codespaces" href="https://codespaces.new/cr2007/whatsapp-voice-ai-assistant">
        <img src="https://github.com/codespaces/badge.svg" />
    </a>
    <br>
    <a href="https://go.dev">
        <img alt="Go" title="Go Programming Language" src="https://img.shields.io/badge/Go-informational?style=flat&logo=go&logoColor=white&color=00add8">
    </a>
    <a href="https://python.org/">
        <img alt="Python" title="Python Programming Language" src="https://img.shields.io/badge/Python-informational?style=flat&logo=python&logoColor=white&color=3776ab">
    </a>
    <a href="https://go.dev">
        <img alt="WhatsApp" title="WhatsApp" src="https://img.shields.io/badge/WhatsApp-informational?style=flat&logo=WhatsApp&logoColor=white&color=25D366">
    </a>
    <a href="https://groq.com">
        <img alt="Groq" title="Groq" src="https://img.shields.io/badge/Groq-informational?style=flat&logo=groq&logoColor=white&color=F55036">
    </a>
</div>
<br>

A microservices-based WhatsApp bot that automatically transcribes voice messages and provides AI-powered responses.

<!-- omit from toc -->
## Table of Contents
- [Features](#features)
- [Technologies Used](#technologies-used)
- [Setup](#setup)
  - [Prerequisites](#prerequisites)
  - [Installation](#installation)
    - [Go](#go)
    - [Python](#python)
- [Usage](#usage)
- [Configuration](#configuration)
- [API Endpoint](#api-endpoint)
- [Contributing](#contributing)
- [Acknowledgements](#acknowledgements)

## Features
- Real-time voice message transcription
- AI-powered responses (powered by Groq API)
- Integration with WhatsApp
- Microservices architecture for scalability

## Technologies Used
- Go
- Python
- Flask
- SQLite
- WhatsApp API (via whatsmeow)
- Faster-Whisper (for speech recognition)
- Groq API

## Setup

### Prerequisites

To run this application, make sure you have the following installed:

- [Go](https://go.dev)
- [Python](https://python.org)
- Groq API Key (Get it from [groqcloud](https://console.groq.com/keys))

> [!IMPORTANT]
> In order to use the Whatsmeow library on Windows, ensure that you have GCC installed.
>
> Alternatively, you can use WSL for running the Go code. The Python code can run normally on any machine.

### Installation

#### Go

```go
go mod tidy
```

#### Python
1. Create a virtual environment

```python
python -m venv .venv
```

2. Activate the virtual environment

```shell
# For Linux
source .venv/bin/activate

# For Windows
.venv\Scripts\activate
```

## Usage

> [!NOTE]
> Before you start, make sure to configure your IP Address in [`main.go`](./main.go) for sending the audio data to the Flask server.

To start, you need to start the Go application as well as the Flask server.

1. Start the Flask server

```
python main.py
```

Note down the IP Address mentioned in the terminal, as you would need it to configure the Go application for sending the audio data.

2. Start the Go application

```go
go run main.go
```

3. Scan the QR Code displayed in the terminal to log into WhatsApp

4. Once logged in, any audio message sent to you will be transcribed and you will receive the response from the AI model sent back as a WhatsApp message.

## Configuration

1. You would need to change your IP Address in `main.go` that sends the POST request to the Flask server.
2. The transcription model can be changed in the `transcribe.py`file by modifying the `model_name` parameter.
3. To use a different AI model, update the `Model` field in the `RequestPayload` struct within the [`groq/groq.go`](./groq/groq.go) file.

## API Endpoint
The transcription service exposes a single endpoint:

- POST `/transcribe`

Accepts binary audio data in the request body.<br>
Returns a JSON object with transcription and language fields

## Contributing
1. Create a [new issue](https://github.com/cr2007/whatsapp-voice-ai-assistant/issues/new/choose)
1. [Fork the repository](https://github.com/cr2007/whatsapp-voice-ai-assistant/fork)
2. Create a new branch
4. Commit your changes
5. Push to the branch
6. Create a new Pull Request

## Acknowledgements

1. [YASSERMD/whatsmeow-groq](https://github.com/YASSERRMD/whatsmeow-groq)
2. [hoehermann/whatsmeow-transcribe](https://github.com/hoehermann/whatsmeow-transcribe)
