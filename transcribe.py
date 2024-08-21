import io
from faster_whisper import WhisperModel

def transcribe_audio(audio_data, model_name="base", word_timestamps=True, verbose=True):
	audio_stream = io.BytesIO(audio_data)

	model = WhisperModel(model_name, device="cpu", compute_type="int8")
	segments, info = model.transcribe(audio_stream, word_timestamps=word_timestamps)

	result = {
		"text": " ".join([segment.text for segment in segments]).strip(),
		"language": info.language,  # Add detected language to the result
	}

	print(f"Transcription: {result["text"]}")

	if verbose:
		print(f"Transcription info: {info}")

	return result
