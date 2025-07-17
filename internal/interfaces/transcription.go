package interfaces

// TranscriptionProvider defines methods for audio transcription
type TranscriptionProvider interface {
	TranscribeAudio(audioPath string) (string /*transcriptFilePath*/, error)
	GetSupportedLanguages() []string
}
