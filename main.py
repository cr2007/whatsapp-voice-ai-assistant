from flask import Flask, request, jsonify
from transcribe import transcribe_audio  # Import the transcribe_audio function

app = Flask(__name__)

@app.route('/transcribe', methods=['POST'])
def transcribe():
	# Get the binary audio data from the request
	audio_data = request.data

	# Call the transcribe_audio function
	result = transcribe_audio(
		audio_data,
		model_name="medium.en",
		word_timestamps=True,
		verbose=False
	)

	# Return the transcription as a JSON response
	return jsonify({
		"transcription": result["text"],
		"language": result["language"],
		"segments": result["segments"]
	})

if __name__ == '__main__':
	app.run(host="0.0.0.0", port=5000)
