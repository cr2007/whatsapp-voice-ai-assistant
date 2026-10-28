from unittest.mock import patch

from fastapi.testclient import TestClient

from main import app

client = TestClient(app)


def test_transcribe_forwards_audio_and_returns_result():
    result = {"text": "bonjour", "language": "fr"}
    with patch("main.transcribe_audio", return_value=result) as mock:
        resp = client.post("/transcribe", content=b"raw audio bytes")

    assert resp.status_code == 200
    assert resp.json() == {"transcription": "bonjour", "language": "fr"}
    mock.assert_called_once_with(b"raw audio bytes", model_name="medium.en", word_timestamps=True, verbose=False)
