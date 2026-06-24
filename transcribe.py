import io
from typing import TypedDict

from faster_whisper import WhisperModel


class TranscriptionResult(TypedDict):
    text: str
    language: str


def transcribe_audio(
    audio_data: bytes,
    model_name: str = "base",
    word_timestamps: bool = True,
    verbose: bool = True,
) -> TranscriptionResult:
    audio_stream = io.BytesIO(audio_data)

    model = WhisperModel(model_name, device="cpu", compute_type="int8")
    segments, info = model.transcribe(audio_stream, word_timestamps=word_timestamps)

    result: TranscriptionResult = {
        "text": " ".join([segment.text for segment in segments]).strip(),
        "language": info.language,
    }

    print(f"Transcription: {result['text']}")

    if verbose:
        print(f"Transcription info: {info}")

    return result
