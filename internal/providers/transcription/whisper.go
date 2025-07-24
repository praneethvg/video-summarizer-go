package transcription

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	log "github.com/sirupsen/logrus"
)

// WhisperCppTranscriptionProvider implements interfaces.TranscriptionProvider using whisper.cpp CLI
type WhisperCppTranscriptionProvider struct {
	WhisperPath string // path to whisper.cpp binary (e.g., ./tools/whisper)
	ModelPath   string // path to model file (e.g., ./models/ggml-base.en.bin)
}

func NewWhisperCppTranscriptionProvider(whisperPath, modelPath string) *WhisperCppTranscriptionProvider {
	return &WhisperCppTranscriptionProvider{
		WhisperPath: whisperPath,
		ModelPath:   modelPath,
	}
}

// TranscribeAudio runs whisper.cpp CLI and returns the path to the transcript file
func (p *WhisperCppTranscriptionProvider) TranscribeAudio(audioPath string) (string, error) {
	// Create a temp file for the transcript base (no .txt extension)
	tmpFile, err := ioutil.TempFile("", "transcript-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp transcript file: %v", err)
	}
	tmpBasePath := tmpFile.Name()
	tmpFile.Close()

	cmdArgs := []string{"-m", p.ModelPath, "-f", audioPath, "-otxt", "-of", tmpBasePath}
	log.Infof("Running command: %s %v", p.WhisperPath, cmdArgs)
	cmd := exec.Command(p.WhisperPath, cmdArgs...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		os.Remove(tmpBasePath + ".txt")
		log.Errorf("%v, output: %s", err, out.String())
		return "", fmt.Errorf("whisper.cpp error: %v, output: %s", err, out.String())
	}

	transcriptPath := tmpBasePath + ".txt"
	// Check file size and log output if empty
	info, statErr := os.Stat(transcriptPath)
	if statErr != nil {
		log.Errorf("could not stat transcript file: %v", statErr)
	} else {
		log.Debugf("Transcript file size: %d bytes", info.Size())
		if info.Size() == 0 {
			log.Warnf("transcript file is empty! Command output: %s", out.String())
		}
	}

	return transcriptPath, nil
}

// GetSupportedLanguages returns supported languages (for demo, just English)
func (p *WhisperCppTranscriptionProvider) GetSupportedLanguages() []string {
	return []string{"en"}
}
