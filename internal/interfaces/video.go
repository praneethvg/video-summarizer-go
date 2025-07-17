package interfaces

// VideoProvider defines methods for video information and audio extraction
type VideoProvider interface {
	GetVideoInfo(url string) (map[string]interface{}, error)
	DownloadAudio(url string) (string, error)
	SupportsURL(url string) bool
}
