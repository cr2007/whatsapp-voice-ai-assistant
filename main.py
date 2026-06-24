import socket
from collections.abc import AsyncIterator
from contextlib import asynccontextmanager

from fastapi import FastAPI, Request
from pydantic import BaseModel, Field

from transcribe import transcribe_audio


class TranscriptionResponse(BaseModel):
    transcription: str = Field(description="The transcribed text from the audio.")
    language: str = Field(description="BCP-47 language code detected by Whisper (e.g. 'en').")


def _local_ip() -> str:
    with socket.socket(socket.AF_INET, socket.SOCK_DGRAM) as s:
        s.connect(("8.8.8.8", 80))
        return s.getsockname()[0]


@asynccontextmanager
async def lifespan(app: FastAPI) -> AsyncIterator[None]:
    print(f" * LAN address: {_local_ip()}")
    yield


app = FastAPI(
    title="WhatsApp Voice Transcription API",
    description=(
        "Local transcription service for the WhatsApp Voice AI Assistant. "
        "Accepts raw audio bytes and returns text and detected language via "
        "[Faster-Whisper](https://github.com/SYSTRAN/faster-whisper) on CPU."
    ),
    version="1.0.0",
    lifespan=lifespan,
)


@app.post(
    "/transcribe",
    summary="Transcribe audio",
    description=(
        "POST raw audio bytes (`application/octet-stream`).\n\n"
        "Supports any format Faster-Whisper accepts: WAV, MP3, OGG, Opus, M4A, and more. "
        "Runs locally on CPU using `medium.en`."
    ),
    response_description="Transcribed text and BCP-47 language code.",
    tags=["Transcription"],
)
async def transcribe(request: Request) -> TranscriptionResponse:
    audio_data = await request.body()

    result = transcribe_audio(
        audio_data,
        model_name="medium.en",
        word_timestamps=True,
        verbose=False,
    )

    return TranscriptionResponse(
        transcription=result["text"],
        language=result["language"],
    )


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=5000)
