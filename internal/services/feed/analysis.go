package feed

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/littleironwaltz/bluesky-mcp/internal/auth"
	"github.com/littleironwaltz/bluesky-mcp/internal/cache"
	"github.com/littleironwaltz/bluesky-mcp/internal/models"
	_ "github.com/littleironwaltz/bluesky-mcp/pkg/apiclient" // We need the BlueskyClient impl
	"github.com/littleironwaltz/bluesky-mcp/pkg/config"
)

// BlueskyAPIClient defines the interface for Bluesky API client
type BlueskyAPIClient interface {
	Get(endpoint string, params url.Values) ([]byte, error)
	Post(endpoint string, body interface{}) ([]byte, error)
	SetAuthToken(token string)
}

// Cache for feed operations
var (
	feedCache = cache.NewWithOptions(cache.CacheOptions{
		MaxItems:         2000,
		DefaultTTL:       5 * time.Minute,
		CleanupInterval:  5 * time.Minute,
		AllowStaleOnFail: true,
		StaleTimeout:     1 * time.Hour,
		PersistOptions: cache.PersistOptions{
			Enabled:       true,
			Directory:     "./cache/feed",
			Filename:      "feed_cache.json",
			SaveInterval:  10 * time.Minute,
			LoadOnStartup: true,
		},
	})
)

// FetchError represents an error during feed fetching
type FetchError struct {
	Message   string
	Cause     error
	Retryable bool
}

func (e FetchError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Note: We're now using the shared client from auth.GetTokenManager().GetClient()
// This ensures we have a consistent authentication state across all services

// AnalyzeFeed processes and analyzes a user's feed
func AnalyzeFeed(cfg config.Config, params map[string]interface{}) (interface{}, error) {
	// Validate and extract parameters
	params, err := validateParams(params)
	if err != nil {
		return nil, err
	}

	hashtag := params["hashtag"].(string)
	limit := int(params["limit"].(float64))

	// Generate cache key
	cacheKey := generateCacheKey(hashtag, limit)

	// Try to get from cache with the loader function
	result, err := feedCache.GetWithLoader(cacheKey, 2*time.Minute, func() (interface{}, error) {
		// This function is called if the item isn't in the cache
		return fetchAndProcessFeed(cfg, hashtag, limit)
	})

	if err != nil {
		// Even with the error, we might have gotten a stale result
		if result != nil {
			// We have a stale result (from fallback cache)
			// Return the stale result with a warning
			if feedResp, ok := result.(models.FeedResponse); ok {
				feedResp.Warning = "Data may be stale due to API errors"
				feedResp.Source = "cache_stale"
				return feedResp, nil
			}
		}
		return nil, fmt.Errorf("feed analysis failed: %w", err)
	}

	// Add source if missing
	if feedResp, ok := result.(models.FeedResponse); ok && feedResp.Source == "" {
		feedResp.Source = "api_fresh"
		return feedResp, nil
	}

	return result, nil
}

// fetchAndProcessFeed fetches and processes the feed data
func fetchAndProcessFeed(cfg config.Config, hashtag string, limit int) (interface{}, error) {
	// Get auth token
	token, err := auth.GetToken(cfg)
	if err != nil {
		return nil, FetchError{
			Message:   "Authentication error",
			Cause:     err,
			Retryable: true,
		}
	}

	// Get the shared authentication token manager's client
	client := auth.GetTokenManager(cfg).GetClient()
	
	// Make sure the client has the auth token set
	client.SetAuthToken(token)

	// Fetch feed data with parallelism and timeout for large feeds
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Use client as BlueskyAPIClient (it implements the interface)
	var apiClient BlueskyAPIClient = client
	feedData, err := fetchFeedWithTimeout(ctx, apiClient, hashtag, limit)
	if err != nil {
		return nil, err
	}

	// Process posts with parallelism for sentiment analysis
	posts := processPostsParallel(feedData, hashtag, limit)

	// Create response
	result := models.FeedResponse{
		Posts:  posts,
		Count:  len(posts),
		Source: "api_fresh",
	}

	return result, nil
}

// validateParams validates and normalizes the request parameters
func validateParams(params map[string]interface{}) (map[string]interface{}, error) {
	// Set defaults for missing params
	if _, ok := params["hashtag"]; !ok {
		params["hashtag"] = ""
	}

	if _, ok := params["limit"]; !ok {
		params["limit"] = float64(10)
	}

	// Validate hashtag
	hashtag, ok := params["hashtag"].(string)
	if !ok {
		params["hashtag"] = ""
	} else {
		// Sanitize hashtag input
		hashtag = strings.TrimSpace(hashtag)
		hashtag = html.EscapeString(hashtag)
		params["hashtag"] = hashtag
	}

	// Validate limit
	limit, ok := params["limit"].(float64)
	if !ok || limit <= 0 || limit > 100 {
		params["limit"] = float64(10)
	}

	return params, nil
}

// fetchFeedWithTimeout retrieves feed data from the API with a timeout
func fetchFeedWithTimeout(ctx context.Context, client BlueskyAPIClient, hashtag string, limit int) ([]byte, error) {
	// Create a channel for the result
	type fetchResult struct {
		data []byte
		err  error
	}
	resultCh := make(chan fetchResult, 1)

	// Fetch in goroutine
	go func() {
		data, err := fetchFeed(client, hashtag, limit)
		resultCh <- fetchResult{data, err}
	}()

	// Wait for either result or timeout
	select {
	case <-ctx.Done():
		return nil, FetchError{
			Message:   "Feed fetch timed out",
			Cause:     ctx.Err(),
			Retryable: true,
		}
	case result := <-resultCh:
		return result.data, result.err
	}
}

// fetchFeed retrieves feed data from the API
func fetchFeed(client BlueskyAPIClient, hashtag string, limit int) ([]byte, error) {
	// Build query parameters
	query := url.Values{}
	query.Set("limit", fmt.Sprintf("%d", limit))
	
	var endpoint string
	var responseData []byte
	var err error
	
	// Use the search endpoint if hashtag is provided, otherwise use timeline
	if hashtag != "" {
		// Use search endpoint for hashtags
		endpoint = "app.bsky.feed.searchPosts"
		query.Set("q", "#" + hashtag)
		responseData, err = client.Get(endpoint, query)
	} else {
		// Use timeline endpoint if no hashtag specified
		endpoint = "app.bsky.feed.getTimeline"
		responseData, err = client.Get(endpoint, query)
	}
	
	if err != nil {
		return nil, FetchError{
			Message:   fmt.Sprintf("%s API request failed", endpoint),
			Cause:     err,
			Retryable: isRetryableError(err),
		}
	}
	
	// Check if we received valid JSON
	var checkJSON map[string]interface{}
	if err := json.Unmarshal(responseData, &checkJSON); err != nil {
		return nil, FetchError{
			Message:   "Invalid JSON response from API",
			Cause:     err,
			Retryable: true,
		}
	}
	
	// Check if this is a fallback response by examining the first post's author
	if isFallbackResponse(checkJSON) {
		return responseData, nil
	}
	
	return responseData, nil
}

// isFallbackResponse determines if the response is from the fallback system
func isFallbackResponse(data map[string]interface{}) bool {
	// Check for nil or empty data
	if data == nil || len(data) == 0 {
		return false
	}
	
	// Check if this is a timeline response with feed data
	if feed, ok := data["feed"].([]interface{}); ok && len(feed) > 0 {
		// Extract first post from timeline
		firstPost, ok := feed[0].(map[string]interface{})
		if !ok {
			return false
		}
		
		// Extract post data
		postData, ok := firstPost["post"].(map[string]interface{})
		if !ok {
			return false
		}
		
		// Extract author data
		author, ok := postData["author"].(map[string]interface{})
		if !ok {
			return false
		}
		
		// Check handle
		handle, ok := author["handle"].(string)
		if !ok {
			return false
		}
		
		return handle == "fallback.system"
	}
	
	// Check if this is a search response with posts data
	if posts, ok := data["posts"].([]interface{}); ok && len(posts) > 0 {
		// Extract first post from search
		firstPost, ok := posts[0].(map[string]interface{})
		if !ok {
			return false
		}
		
		// Extract author data (search API has different structure)
		author, ok := firstPost["author"].(map[string]interface{})
		if !ok {
			return false
		}
		
		// Check handle
		handle, ok := author["handle"].(string)
		if !ok {
			return false
		}
		
		return handle == "fallback.system"
	}
	
	return false
}

// isRetryableError determines if an error should be retried
func isRetryableError(err error) bool {
	errStr := err.Error()
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "EOF") ||
		strings.Contains(errStr, "status 500") ||
		strings.Contains(errStr, "status 502") ||
		strings.Contains(errStr, "status 503") ||
		strings.Contains(errStr, "status 504")
}

// FeedResponse represents the full feed data structure
type FeedResponse struct {
	Feed []FeedItem `json:"feed"`
}

// processPostsParallel processes the feed posts with parallel sentiment analysis
func processPostsParallel(feedData []byte, hashtag string, limit int) []models.Post {
	// Try to unmarshal as timeline response first
	var feed FeedResponse
	if err := json.Unmarshal(feedData, &feed); err != nil || feed.Feed == nil {
		// If that fails, try as search response format
		var searchResp struct {
			Posts []struct {
				URI    string `json:"uri"`
				Record struct {
					Text      string `json:"text"`
					CreatedAt string `json:"createdAt"`
				} `json:"record"`
				Author struct {
					Handle string `json:"handle"`
				} `json:"author"`
			} `json:"posts"`
		}
		
		if err := json.Unmarshal(feedData, &searchResp); err != nil {
			return []models.Post{}
		}
		
		// Convert search response to feed items
		feedItems := make([]FeedItem, 0, len(searchResp.Posts))
		for _, post := range searchResp.Posts {
			item := FeedItem{}
			item.Post.URI = post.URI
			item.Post.Record.Text = post.Record.Text
			item.Post.Record.CreatedAt = post.Record.CreatedAt
			item.Post.Author.Handle = post.Author.Handle
			feedItems = append(feedItems, item)
		}
		
		// Process the converted search results
		return processItems(feedItems, hashtag, limit)
	}
	
	// For timeline responses, process as before
	return processItems(feed.Feed, hashtag, limit)
}

// processItems processes feed items with parallel sentiment analysis
func processItems(items []FeedItem, hashtag string, limit int) []models.Post {
	var (
		posts    = make([]models.Post, 0, limit)
		mu       sync.Mutex
		wg       sync.WaitGroup
		filtered = filterPosts(items, hashtag, limit)
	)

	// Process posts in parallel
	wg.Add(len(filtered))
	for _, item := range filtered {
		go func(item FeedItem) {
			defer wg.Done()
			
			// Create post with analysis
			post := models.Post{
				ID:        getPostID(item.Post.URI),
				Text:      item.Post.Record.Text,
				CreatedAt: item.Post.Record.CreatedAt,
				Author:    item.Post.Author.Handle,
				Analysis: map[string]string{
					"sentiment": analyzeSentiment(item.Post.Record.Text),
				},
			}
			
			// Add metrics if available
			post.Metrics = calculateMetrics(item.Post.Record.Text)
			
			// Add to results thread-safely
			mu.Lock()
			posts = append(posts, post)
			mu.Unlock()
		}(item)
	}
	
	// Wait for all analyses to complete
	wg.Wait()
	
	return posts
}

// FeedItem represents a single post item in the feed
type FeedItem struct {
	Post struct {
		URI string `json:"uri"`
		Record struct {
			Text      string `json:"text"`
			CreatedAt string `json:"createdAt"`
		} `json:"record"`
		Author struct {
			Handle string `json:"handle"`
		} `json:"author"`
	} `json:"post"`
}

// filterPosts filters posts based on criteria
func filterPosts(feed []FeedItem, hashtag string, limit int) []FeedItem {
	var result = make([]FeedItem, 0, limit)
	
	// When using the search endpoint, we don't need to filter by hashtag again
	// because the API has already filtered for us
	if hashtag != "" {
		// Just take the first 'limit' items from the search results
		for i, item := range feed {
			if i >= limit {
				break
			}
			result = append(result, item)
		}
	} else {
		// For non-hashtag requests, just return all posts up to the limit
		for i, item := range feed {
			if i >= limit {
				break
			}
			result = append(result, item)
		}
	}
	
	return result
}

// calculateMetrics calculates additional metrics for a post
func calculateMetrics(text string) map[string]int {
	words := strings.Fields(text)
	
	return map[string]int{
		"length": len(text),
		"words":  len(words),
	}
}

// getPostID extracts a shorter post ID from the URI
func getPostID(uri string) string {
	parts := strings.Split(uri, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// generateCacheKey creates a unique key for caching
func generateCacheKey(hashtag string, limit int) string {
	key := fmt.Sprintf("feed:%s:%d", hashtag, limit)
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// analyzeSentiment performs basic sentiment analysis
func analyzeSentiment(text string) string {
	text = strings.ToLower(text)
	
	// Simple word-based sentiment analysis
	positiveWords := []string{"good", "great", "happy", "excited", "love", "awesome"}
	negativeWords := []string{"bad", "sad", "angry", "hate", "terrible", "awful"}
	
	var positiveCount, negativeCount int
	
	for _, word := range positiveWords {
		if strings.Contains(text, word) {
			positiveCount++
		}
	}
	
	for _, word := range negativeWords {
		if strings.Contains(text, word) {
			negativeCount++
		}
	}
	
	if positiveCount > negativeCount {
		return "positive"
	} else if negativeCount > positiveCount {
		return "negative"
	}
	
	return "neutral"
}