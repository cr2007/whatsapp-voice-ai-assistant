from unittest.mock import MagicMock, patch

from transcribe import transcribe_audio


def _make_model(texts: list[str], language: str = "en") -> MagicMock:
    model = MagicMock()
    model.transcribe.return_value = ([MagicMock(text=t) for t in texts], MagicMock(language=language))
    return model


def test_segments_are_joined_and_stripped():
    with patch("transcribe.WhisperModel", return_value=_make_model(["  hello", "world  "])):
        result = transcribe_audio(b"audio", verbose=False)
    assert result["text"] == "hello world"


def test_no_segments_return_empty_text():
    with patch("transcribe.WhisperModel", return_value=_make_model([])):
        result = transcribe_audio(b"audio", verbose=False)
    assert result["text"] == ""


def test_model_name_and_language_are_passed_through():
    with patch("transcribe.WhisperModel", return_value=_make_model(["bonjour"], language="fr")) as mock_cls:
        result = transcribe_audio(b"audio", model_name="tiny", verbose=False)
    mock_cls.assert_called_once_with("tiny", device="cpu", compute_type="int8")
    assert result["language"] == "fr"
