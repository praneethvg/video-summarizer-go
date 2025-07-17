package output

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"video-summarizer-go/internal/config"
)

type GDriveOutputProvider struct {
	driveService *drive.Service
	folderID     string
}

func NewGDriveOutputProvider(cfg *config.AppConfig) (*GDriveOutputProvider, error) {
	ctx := context.Background()

	var service *drive.Service
	var err error

	switch cfg.GDriveAuthMethod {
	case "oauth":
		// Use OAuth client + user token
		creds, err := os.ReadFile(cfg.GDriveCredentials)
		if err != nil {
			return nil, fmt.Errorf("failed to read OAuth credentials file: %w", err)
		}
		config, err := google.ConfigFromJSON(creds, drive.DriveFileScope)
		if err != nil {
			return nil, fmt.Errorf("failed to parse OAuth credentials: %w", err)
		}
		tok, err := tokenFromFile(cfg.GDriveToken)
		if err != nil {
			return nil, fmt.Errorf("failed to read OAuth token file: %w", err)
		}
		client := config.Client(ctx, tok)
		service, err = drive.NewService(ctx, option.WithHTTPClient(client))
		if err != nil {
			return nil, fmt.Errorf("failed to create Google Drive service (oauth): %w", err)
		}
	case "service_account":
		fallthrough
	default:
		// Use service account (default)
		service, err = drive.NewService(ctx, option.WithCredentialsFile(cfg.GDriveCredentials))
		if err != nil {
			return nil, fmt.Errorf("failed to create Google Drive service (service_account): %w", err)
		}
	}

	return &GDriveOutputProvider{
		driveService: service,
		folderID:     cfg.GDriveFolderID,
	}, nil
}

func (g *GDriveOutputProvider) UploadSummary(requestID string, videoInfo map[string]interface{}, summaryPath string) error {
	title := ""
	if t, ok := videoInfo["title"].(string); ok {
		title = t
	}
	return g.uploadFileAndCleanup(requestID, title, summaryPath, "summary.txt")
}

func (g *GDriveOutputProvider) UploadTranscript(requestID string, videoInfo map[string]interface{}, transcriptPath string) error {
	title := ""
	if t, ok := videoInfo["title"].(string); ok {
		title = t
	}
	return g.uploadFileAndCleanup(requestID, title, transcriptPath, "transcript.txt")
}

// uploadFileAndCleanup uploads a file to Google Drive and deletes it after upload
func (g *GDriveOutputProvider) uploadFileAndCleanup(requestID, title, filePath, suffix string) error {
	filename := buildOutputFilename(title, requestID, suffix)
	file := &drive.File{
		Name:     filename,
		Parents:  []string{g.folderID},
		MimeType: "text/plain",
	}
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	start := time.Now()
	fmt.Printf("[GDrive] Uploading %s for request %s...\n", filename, requestID)
	_, err = g.driveService.Files.Create(file).Media(f).Do()
	elapsed := time.Since(start)
	if err != nil {
		fmt.Printf("[GDrive] ERROR uploading %s for request %s: %v (%.2fs)\n", filename, requestID, err, elapsed.Seconds())
	} else {
		fmt.Printf("[GDrive] Uploaded %s for request %s in %.2fs\n", filename, requestID, elapsed.Seconds())
	}
	// Cleanup file after upload
	if rmErr := os.Remove(filePath); rmErr != nil {
		fmt.Printf("[GDrive] WARNING: failed to remove temp file %s: %v\n", filePath, rmErr)
	}
	if err != nil {
		return fmt.Errorf("failed to upload %s to Google Drive: %w", filename, err)
	}
	return nil
}

// getTitleForRequest is a placeholder; in real use, fetch from state store or pass as arg
func getTitleForRequest(requestID string) string {
	// TODO: Fetch video title from state store or pass as argument
	return ""
}

// buildOutputFilename builds a sanitized filename
func buildOutputFilename(title, requestID, suffix string) string {
	if title != "" {
		title = sanitizeFilename(title)
		return fmt.Sprintf("%s_%s_%s", title, requestID, suffix)
	}
	return fmt.Sprintf("%s_%s", requestID, suffix)
}

// sanitizeFilename removes/escapes problematic characters
func sanitizeFilename(name string) string {
	// Remove non-alphanumeric, replace spaces with _
	name = strings.ReplaceAll(name, " ", "_")
	reg := regexp.MustCompile(`[^a-zA-Z0-9_\-]`)
	return reg.ReplaceAllString(name, "")
}

// tokenFromFile loads an OAuth2 token from a file
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var token oauth2.Token
	if err := json.NewDecoder(f).Decode(&token); err != nil {
		return nil, err
	}
	return &token, nil
}
