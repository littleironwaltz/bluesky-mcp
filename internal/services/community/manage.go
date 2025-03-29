package community

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/littleironwaltz/bluesky-mcp/internal/auth"
	"github.com/littleironwaltz/bluesky-mcp/internal/cache"
	"github.com/littleironwaltz/bluesky-mcp/pkg/config"
)

// Cache for user feed results
var (
	userFeedCache = cache.New()
)

func ManageCommunity(cfg config.Config, params map[string]interface{}) (interface{}, error) {
	// Proper type assertions with validation
	userHandle, ok := params["userHandle"].(string)
	if !ok || userHandle == "" {
		return nil, fmt.Errorf("missing or invalid user handle")
	}

	limit, ok := params["limit"].(float64)
	if !ok || limit <= 0 || limit > 50 {
		// Default with reasonable upper bound
		limit = 5
	}

	// Validate and sanitize userHandle to prevent injection
	userHandle = strings.TrimSpace(userHandle)
	if !strings.HasPrefix(userHandle, "did:") && !strings.Contains(userHandle, ".") {
		return nil, fmt.Errorf("invalid user handle format")
	}


	// Generate cache key based on params
	cacheKey := generateCacheKey(userHandle, limit)

	// Check cache first
	if cachedResult, found := userFeedCache.Get(cacheKey); found {
		return cachedResult, nil
	}

	// Get auth token from Bluesky API
	token, err := auth.GetToken(cfg)
	if err != nil {
		return nil, fmt.Errorf("authentication error")
	}

	// Get the shared authentication token manager's client
	client := auth.GetTokenManager(cfg).GetClient()
	
	// Make sure the client has the auth token set
	client.SetAuthToken(token)

	// Prepare parameters
	query := url.Values{}
	query.Set("actor", userHandle)
	query.Set("limit", fmt.Sprintf("%d", int(limit)))

	// Make API request
	responseBody, err := client.Get("app.bsky.feed.getAuthorFeed", query)
	if err != nil {
		return nil, fmt.Errorf("API request error")
	}

	var feed struct {
		Feed []struct {
			Post struct {
				Record struct {
					Text      string    `json:"text"`
					CreatedAt time.Time `json:"createdAt"`
				} `json:"record"`
			} `json:"post"`
		} `json:"feed"`
	}

	if err := json.Unmarshal(responseBody, &feed); err != nil {
		return nil, fmt.Errorf("response parsing error")
	}

	// Pre-allocate slice with capacity equal to limit for better performance
	recentPosts := make([]string, 0, int(limit))
	weekAgo := time.Now().Add(-7 * 24 * time.Hour)

	for _, item := range feed.Feed {
		if item.Post.Record.CreatedAt.After(weekAgo) {
			recentPosts = append(recentPosts, item.Post.Record.Text)
		}
		if len(recentPosts) >= int(limit) {
			break
		}
	}

	// Prepare result
	result := map[string]interface{}{
		"user":        userHandle,
		"recentPosts": recentPosts,
		"count":       len(recentPosts),
	}

	// Cache the result for 3 minutes
	userFeedCache.Set(cacheKey, result, 3*time.Minute)

	return result, nil
}

// generateCacheKey creates a unique key for caching based on parameters
func generateCacheKey(userHandle string, limit float64) string {
	hash := sha256.New()
	hash.Write([]byte(fmt.Sprintf("user:%s:%f", userHandle, limit)))
	return hex.EncodeToString(hash.Sum(nil))
}
