// Package fallbacks provides cached responses for when API calls fail
package fallbacks

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/littleironwaltz/bluesky-mcp/pkg/apiclient"
)

var (
	fallbacksPath = "./configs/fallbacks"
	loaderOnce    sync.Once
	initialized   bool
)

// InitializeFallbacks loads fallback responses from disk and registers them
func InitializeFallbacks(client *apiclient.BlueskyClient) error {
	var initErr error
	
	loaderOnce.Do(func() {
		// Load timeline fallback
		timelineData, err := loadFallbackFile("timeline.json")
		if err != nil {
			initErr = fmt.Errorf("failed to load timeline fallback: %w", err)
			return
		}
		
		// Register fallback responses
		client.RegisterFallbackResponse("app.bsky.feed.getTimeline", timelineData)
		
		// Add more fallbacks as needed
		
		initialized = true
		log.Println("Fallback responses initialized")
	})
	
	return initErr
}

// loadFallbackFile loads a fallback JSON file from the fallbacks directory
func loadFallbackFile(filename string) ([]byte, error) {
	filePath, err := filepath.Abs(filepath.Join(fallbacksPath, filename))
	if err != nil {
		return nil, err
	}
	
	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	
	// Validate JSON
	var testJSON interface{}
	if err := json.Unmarshal(data, &testJSON); err != nil {
		return nil, fmt.Errorf("invalid JSON in fallback file %s: %w", filename, err)
	}
	
	return data, nil
}

// IsInitialized returns whether fallbacks have been successfully initialized
func IsInitialized() bool {
	return initialized
}