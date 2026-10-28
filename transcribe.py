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
    """Transcribe audio bytes on the CPU with Faster-Whisper.

    Returns the joined segment text and the detected language code.
    """
    # Step 1: Load the model and transcribe the audio.
    model = WhisperModel(model_name, device="cpu", compute_type="int8")
    segments, info = model.transcribe(io.BytesIO(audio_data), word_timestamps=word_timestamps)

    # Step 2: Join the segments into one string.
    result: TranscriptionResult = {
        "text": " ".join(segment.text for segment in segments).strip(),
        "language": info.language,
    }

    print(f"Transcription: {result['text']}")
    if verbose:
        print(f"Transcription info: {info}")

    return result
