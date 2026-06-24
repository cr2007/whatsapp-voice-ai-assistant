from unittest.mock import MagicMock, patch

from transcribe import transcribe_audio


def _make_segment(text: str) -> MagicMock:
    seg = MagicMock()
    seg.text = text
    return seg


def _make_model(segments: list, language: str = "en") -> MagicMock:
    info = MagicMock()
    info.language = language
    model = MagicMock()
    model.transcribe.return_value = (segments, info)
    return model


def test_single_segment_is_stripped():
    with patch("transcribe.WhisperModel", return_value=_make_model([_make_segment("  hello  ")])):
        result = transcribe_audio(b"audio", verbose=False)
    assert result["text"] == "hello"


def test_multiple_segments_are_joined():
    segs = [_make_segment("hello"), _make_segment("world")]
    with patch("transcribe.WhisperModel", return_value=_make_model(segs)):
        result = transcribe_audio(b"audio", verbose=False)
    assert result["text"] == "hello world"


def test_empty_segments_return_empty_string():
    with patch("transcribe.WhisperModel", return_value=_make_model([])):
        result = transcribe_audio(b"audio", verbose=False)
    assert result["text"] == ""


def test_detected_language_is_passed_through():
    with patch("transcribe.WhisperModel", return_value=_make_model([_make_segment("bonjour")], language="fr")):
        result = transcribe_audio(b"audio", verbose=False)
    assert result["language"] == "fr"


def test_model_name_is_forwarded():
    model = _make_model([_make_segment("hi")])
    with patch("transcribe.WhisperModel", return_value=model) as mock_cls:
        transcribe_audio(b"audio", model_name="tiny", verbose=False)
    mock_cls.assert_called_once_with("tiny", device="cpu", compute_type="int8")
