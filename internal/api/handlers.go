package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"video-summarizer-go/internal/config"
	"video-summarizer-go/internal/interfaces"
	"video-summarizer-go/internal/services"
	"video-summarizer-go/internal/sources"
)

// APIHandler handles HTTP requests for the video summarizer API
type APIHandler struct {
	submissionService *services.VideoSubmissionService
	promptManager     *config.PromptManager
	sourceManager     *sources.VideoSourceManager
}

// NewAPIHandler creates a new API handler
func NewAPIHandler(submissionService *services.VideoSubmissionService, promptManager *config.PromptManager, sourceManager *sources.VideoSourceManager) *APIHandler {
	return &APIHandler{
		submissionService: submissionService,
		promptManager:     promptManager,
		sourceManager:     sourceManager,
	}
}

// SubmitVideoRequest represents a request to submit a video for processing
type SubmitVideoRequest struct {
	URL      string            `json:"url"`
	Prompt   interfaces.Prompt `json:"prompt"`             // Unified prompt struct
	Category string            `json:"category,omitempty"` // Category for folder organization (default: "general")
	// No metadata field
}

// SubmitVideoResponse represents the response from submitting a video
type SubmitVideoResponse struct {
	RequestID   string    `json:"request_id"`
	Status      string    `json:"status"`
	SubmittedAt time.Time `json:"submitted_at"`
}

// StatusResponse represents the response from checking a request status
type StatusResponse struct {
	RequestID   string                 `json:"request_id"`
	Status      string                 `json:"status"`
	Progress    float64                `json:"progress"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Error       string                 `json:"error,omitempty"`
	VideoInfo   map[string]interface{} `json:"video_info,omitempty"`
	Transcript  string                 `json:"transcript_path,omitempty"`
	Summary     string                 `json:"summary_path,omitempty"`
	OutputPath  string                 `json:"output_path,omitempty"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status         string         `json:"status"`
	Timestamp      time.Time      `json:"timestamp"`
	RequestCounts  map[string]int `json:"request_counts"`
	EnabledSources []string       `json:"enabled_sources"`
}

// SubmitVideo handles POST /api/submit
func (h *APIHandler) SubmitVideo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SubmitVideoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	// Add API source metadata
	// In the SubmitVideo handler, set SourceType and URL directly
	// (Assume all current requests are for videos)
	sourceType := "video"
	url := req.URL
	category := req.Category
	if category == "" {
		category = "general"
	}
	prompt := req.Prompt
	maxTokens := 10000 // Default value, can be made configurable
	requestID, err := h.submissionService.SubmitVideo(url, prompt, sourceType, category, maxTokens)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to submit video: %v", err), http.StatusInternalServerError)
		return
	}

	response := SubmitVideoResponse{
		RequestID:   requestID,
		Status:      "submitted",
		SubmittedAt: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// GetStatus handles GET /api/status/{requestID}
func (h *APIHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract request ID from URL path
	// Assuming URL pattern: /api/status/{requestID}
	requestID := r.URL.Query().Get("request_id")
	if requestID == "" {
		http.Error(w, "Request ID is required", http.StatusBadRequest)
		return
	}

	state, err := h.submissionService.GetRequestStatus(requestID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get status: %v", err), http.StatusInternalServerError)
		return
	}

	if state == nil {
		http.Error(w, "Request not found", http.StatusNotFound)
		return
	}

	response := StatusResponse{
		RequestID:   state.RequestID,
		Status:      string(state.Status),
		Progress:    state.Progress,
		CreatedAt:   state.CreatedAt,
		UpdatedAt:   state.UpdatedAt,
		CompletedAt: state.CompletedAt,
		Error:       state.Error,
		VideoInfo:   state.VideoInfo,
		Transcript:  state.Transcript,
		Summary:     state.Summary,
		OutputPath:  state.OutputPath,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CancelRequest handles POST /api/cancel/{requestID}
func (h *APIHandler) CancelRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	requestID := r.URL.Query().Get("request_id")
	if requestID == "" {
		http.Error(w, "Request ID is required", http.StatusBadRequest)
		return
	}

	err := h.submissionService.CancelRequest(requestID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to cancel request: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "cancelled"})
}

// Health handles GET /api/health
func (h *APIHandler) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get request counts from the submission service
	requestCounts := h.submissionService.GetRequestCountsByStatus()

	// Get enabled sources
	enabledSources := h.sourceManager.GetEnabledSourceNames()

	response := HealthResponse{
		Status:         "healthy",
		Timestamp:      time.Now(),
		RequestCounts:  requestCounts,
		EnabledSources: enabledSources,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ListPrompts handles GET /api/prompts
func (h *APIHandler) ListPrompts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	prompts := h.promptManager.GetAllPrompts()

	type PromptInfo struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Category    string `json:"category"`
	}

	promptInfos := make([]PromptInfo, len(prompts))
	for i, prompt := range prompts {
		promptInfos[i] = PromptInfo{
			ID:          prompt.ID,
			Name:        prompt.Name,
			Description: prompt.Description,
			Category:    prompt.Category,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"prompts": promptInfos,
		"count":   len(promptInfos),
	})
}
