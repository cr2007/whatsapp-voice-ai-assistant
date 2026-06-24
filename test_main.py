from unittest.mock import patch

import pytest
from fastapi.testclient import TestClient

from main import app

client = TestClient(app)


def _mock_result(text: str = "hello world", language: str = "en") -> dict:
    return {"text": text, "language": language}


def test_transcribe_returns_transcription_and_language():
    with patch("main.transcribe_audio", return_value=_mock_result()):
        resp = client.post("/transcribe", content=b"audio data")
    assert resp.status_code == 200
    assert resp.json() == {"transcription": "hello world", "language": "en"}


def test_transcribe_empty_audio_returns_empty_transcription():
    with patch("main.transcribe_audio", return_value=_mock_result(text="")):
        resp = client.post("/transcribe", content=b"")
    assert resp.status_code == 200
    assert resp.json()["transcription"] == ""


def test_transcribe_non_english_language():
    with patch("main.transcribe_audio", return_value=_mock_result(text="bonjour", language="fr")):
        resp = client.post("/transcribe", content=b"audio")
    assert resp.json()["language"] == "fr"


def test_transcribe_forwards_audio_bytes():
    with patch("main.transcribe_audio", return_value=_mock_result()) as mock:
        client.post("/transcribe", content=b"raw audio bytes")
    args, _ = mock.call_args
    assert args[0] == b"raw audio bytes"
