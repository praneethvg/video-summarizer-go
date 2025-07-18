package video

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// YtDlpVideoProvider implements interfaces.VideoProvider using yt-dlp binary
type YtDlpVideoProvider struct {
	YtDlpPath string // path to yt-dlp binary
	TmpDir    string // where to save temp audio files
}

func NewYtDlpVideoProvider(ytDlpPath, tmpDir string) *YtDlpVideoProvider {
	return &YtDlpVideoProvider{
		YtDlpPath: ytDlpPath,
		TmpDir:    tmpDir,
	}
}

// GetVideoInfo fetches video info as a map using yt-dlp --dump-json
func (p *YtDlpVideoProvider) GetVideoInfo(url string) (map[string]interface{}, error) {
	cmd := exec.Command(p.YtDlpPath, "--dump-json", url)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("yt-dlp error: %v, output: %s", err, out.String())
	}
	var info map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &info); err != nil {
		return nil, fmt.Errorf("failed to parse yt-dlp output: %v", err)
	}
	return info, nil
}

// DownloadAudio downloads audio as mp3 using yt-dlp and returns the file path
func (p *YtDlpVideoProvider) DownloadAudio(url string) (string, error) {
	filename := fmt.Sprintf("audio-%d.mp3", time.Now().UnixNano())
	outPath := filepath.Join(p.TmpDir, filename)
	cmd := exec.Command(p.YtDlpPath, "-x", "--audio-format", "mp3", "-o", outPath, url)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("yt-dlp audio error: %v, output: %s", err, out.String())
	}
	return outPath, nil
}

// SupportsURL returns true if yt-dlp can handle the URL
func (p *YtDlpVideoProvider) SupportsURL(url string) bool {
	return strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be")
}
