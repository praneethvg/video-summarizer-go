package output

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"video-summarizer-go/internal/config"

	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
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

func (g *GDriveOutputProvider) UploadSummary(requestID string, videoInfo map[string]interface{}, summaryPath string, category string, user string) error {
	title := ""
	if t, ok := videoInfo["title"].(string); ok {
		title = t
	}
	return g.uploadFileAndCleanup(requestID, title, summaryPath, "summary.txt", category, user)
}

func (g *GDriveOutputProvider) UploadTranscript(requestID string, videoInfo map[string]interface{}, transcriptPath string, category string, user string) error {
	title := ""
	if t, ok := videoInfo["title"].(string); ok {
		title = t
	}
	return g.uploadFileAndCleanup(requestID, title, transcriptPath, "transcript.txt", category, user)
}

// uploadFileAndCleanup uploads a file to Google Drive and deletes it after upload
func (g *GDriveOutputProvider) uploadFileAndCleanup(requestID, title, filePath, suffix, category, user string) error {
	// Normalize user (default to "admin" if empty)
	if user == "" {
		user = "admin"
	}
	// Normalize category (default to "general" if empty)
	if category == "" {
		category = "general"
	}
	// Create user folder if it doesn't exist
	userFolderID, err := g.getOrCreateUserFolder(user)
	if err != nil {
		return fmt.Errorf("failed to get/create user folder: %w", err)
	}
	// Create category folder under user
	categoryFolderID, err := g.getOrCreateCategoryFolder(category, userFolderID)
	if err != nil {
		return fmt.Errorf("failed to get/create category folder: %w", err)
	}
	// Create video-specific folder under category
	videoFolderID, err := g.getOrCreateVideoFolder(requestID, title, categoryFolderID)
	if err != nil {
		return fmt.Errorf("failed to get/create video folder: %w", err)
	}
	filename := buildOutputFilename(title, requestID, suffix)
	file := &drive.File{
		Name:     filename,
		Parents:  []string{videoFolderID}, // Upload to video-specific folder
		MimeType: "text/plain",
	}
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	start := time.Now()
	log.Infof("Uploading %s for request %s to user: %s, category: %s...", filename, requestID, user, category)
	_, err = g.driveService.Files.Create(file).Media(f).Do()
	elapsed := time.Since(start)
	if err != nil {
		log.Errorf("ERROR uploading %s for request %s: %v (%.2fs)", filename, requestID, err, elapsed.Seconds())
	} else {
		log.Infof("Uploaded %s for request %s in %.2fs", filename, requestID, elapsed.Seconds())
	}
	// Cleanup file after upload
	if rmErr := os.Remove(filePath); rmErr != nil {
		log.Warnf("WARNING: failed to remove temp file %s: %v", filePath, rmErr)
	}
	if err != nil {
		return fmt.Errorf("failed to upload %s to Google Drive: %w", filename, err)
	}
	return nil
}

// getOrCreateUserFolder creates a user folder if it doesn't exist, returns existing if it does
func (g *GDriveOutputProvider) getOrCreateUserFolder(user string) (string, error) {
	query := fmt.Sprintf("name='%s' and mimeType='application/vnd.google-apps.folder' and '%s' in parents and trashed=false", user, g.folderID)
	files, err := g.driveService.Files.List().Q(query).Do()
	if err != nil {
		return "", fmt.Errorf("failed to search for user folder: %w", err)
	}
	if len(files.Files) > 0 {
		log.Infof("Found existing user folder: %s (ID: %s)", user, files.Files[0].Id)
		return files.Files[0].Id, nil
	}
	folder := &drive.File{
		Name:     user,
		MimeType: "application/vnd.google-apps.folder",
		Parents:  []string{g.folderID},
	}
	createdFolder, err := g.driveService.Files.Create(folder).Do()
	if err != nil {
		return "", fmt.Errorf("failed to create user folder: %w", err)
	}
	log.Infof("Created new user folder: %s (ID: %s)", user, createdFolder.Id)
	return createdFolder.Id, nil
}

// getOrCreateCategoryFolder creates a category folder under the user folder
func (g *GDriveOutputProvider) getOrCreateCategoryFolder(category string, userFolderID string) (string, error) {
	query := fmt.Sprintf("name='%s' and mimeType='application/vnd.google-apps.folder' and '%s' in parents and trashed=false", category, userFolderID)
	files, err := g.driveService.Files.List().Q(query).Do()
	if err != nil {
		return "", fmt.Errorf("failed to search for category folder: %w", err)
	}
	if len(files.Files) > 0 {
		log.Infof("Found existing category folder: %s (ID: %s)", category, files.Files[0].Id)
		return files.Files[0].Id, nil
	}
	folder := &drive.File{
		Name:     category,
		MimeType: "application/vnd.google-apps.folder",
		Parents:  []string{userFolderID},
	}
	createdFolder, err := g.driveService.Files.Create(folder).Do()
	if err != nil {
		return "", fmt.Errorf("failed to create category folder: %w", err)
	}
	log.Infof("Created new category folder: %s (ID: %s)", category, createdFolder.Id)
	return createdFolder.Id, nil
}

// getOrCreateVideoFolder creates a video-specific folder under the category folder
func (g *GDriveOutputProvider) getOrCreateVideoFolder(requestID, title, categoryFolderID string) (string, error) {
	// Create folder name from title and request ID
	folderName := buildVideoFolderName(title, requestID)

	// First, try to find existing video folder
	query := fmt.Sprintf("name='%s' and mimeType='application/vnd.google-apps.folder' and '%s' in parents and trashed=false", folderName, categoryFolderID)
	files, err := g.driveService.Files.List().Q(query).Do()
	if err != nil {
		return "", fmt.Errorf("failed to search for video folder: %w", err)
	}

	// If folder exists, return its ID
	if len(files.Files) > 0 {
		log.Infof("Found existing video folder: %s (ID: %s)", folderName, files.Files[0].Id)
		return files.Files[0].Id, nil
	}

	// Create new video folder
	folder := &drive.File{
		Name:     folderName,
		MimeType: "application/vnd.google-apps.folder",
		Parents:  []string{categoryFolderID},
	}

	createdFolder, err := g.driveService.Files.Create(folder).Do()
	if err != nil {
		return "", fmt.Errorf("failed to create video folder: %w", err)
	}

	log.Infof("Created new video folder: %s (ID: %s)", folderName, createdFolder.Id)
	return createdFolder.Id, nil
}

// buildVideoFolderName creates a sanitized folder name for the video
func buildVideoFolderName(title, requestID string) string {
	if title != "" {
		title = sanitizeFilename(title)
		return fmt.Sprintf("%s_%s", title, requestID)
	}
	return fmt.Sprintf("video_%s", requestID)
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
