package interfaces

// OutputProvider defines methods for uploading summary and transcript
// Implementations may upload to Google Drive, S3, webhooks, etc.
type OutputProvider interface {
	UploadSummary(requestID string, videoInfo map[string]interface{}, summaryPath string) error
	UploadTranscript(requestID string, videoInfo map[string]interface{}, transcriptPath string) error
}
