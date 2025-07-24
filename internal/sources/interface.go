package sources

import (
	"context"
)

// ArtifactSource defines the interface for background video sources
type ArtifactSource interface {
	// Start begins the video source processing
	Start(ctx context.Context) error

	// Stop gracefully stops the video source
	Stop() error

	// GetName returns the name of this video source
	GetName() string

	// IsRunning returns true if the source is currently running
	IsRunning() bool
}
