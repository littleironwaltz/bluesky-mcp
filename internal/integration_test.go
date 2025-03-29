package internal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	
	"github.com/littleironwaltz/bluesky-mcp/internal/auth"
	"github.com/littleironwaltz/bluesky-mcp/internal/cache"
	"github.com/littleironwaltz/bluesky-mcp/internal/services/feed"
	"github.com/littleironwaltz/bluesky-mcp/pkg/apiclient"
	"github.com/littleironwaltz/bluesky-mcp/pkg/config"
)

// MockServer creates a mock Bluesky API server for testing
func setupMockServer() (*httptest.Server, *apiclient.BlueskyClient) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/xrpc/com.atproto.server.createSession":
			// Mock successful authentication
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"accessJwt": "mock-access-token",
				"refreshJwt": "mock-refresh-token",
				"handle": "test.bsky.social",
				"did": "did:plc:test12345"
			}`))
		case "/xrpc/app.bsky.feed.getTimeline":
			// Mock timeline response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"feed": [
					{
						"post": {
							"uri": "at://user.bsky.social/post/1",
							"record": {
								"text": "Test post with #golang hashtag",
								"createdAt": "2023-01-01T00:00:00Z"
							},
							"author": {
								"handle": "user.bsky.social"
							}
						}
					}
				]
			}`))
		case "/xrpc/app.bsky.feed.searchPosts":
			// Mock search response - check if query contains hashtag
			hashtag := r.URL.Query().Get("q")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			
			// Return search results with the searched hashtag
			w.Write([]byte(`{
				"posts": [
					{
						"uri": "at://user.bsky.social/post/1",
						"record": {
							"text": "Test post with ` + hashtag + ` hashtag",
							"createdAt": "2023-01-01T00:00:00Z"
						},
						"author": {
							"handle": "user.bsky.social"
						}
					},
					{
						"uri": "at://user.bsky.social/post/2",
						"record": {
							"text": "Another post with ` + hashtag + ` hashtag",
							"createdAt": "2023-01-02T00:00:00Z"
						},
						"author": {
							"handle": "user2.bsky.social"
						}
					}
				]
			}`))
		default:
			// Return 404 for other endpoints
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	// Create client with mock server URL
	client := apiclient.NewClient(server.URL)
	return server, client
}

// IntegrationTest tests the flow from authentication through API request to handler response
func TestAuthAndRequestIntegration(t *testing.T) {
	// Setup mock server
	server, client := setupMockServer()
	defer server.Close()

	// Setup auth service with mock client
	authService := auth.NewAuthService(client)

	// Authenticate
	err := authService.Authenticate("test", "password")
	if err != nil {
		t.Fatalf("Authentication failed: %v", err)
	}

	// Verify auth token is set
	if client.AuthToken == "" {
		t.Fatal("Expected auth token to be set after authentication")
	}

	// Make a request to a protected endpoint
	resp, err := client.Get("app.bsky.feed.getTimeline", nil)
	if err != nil {
		t.Fatalf("API request failed: %v", err)
	}

	// Parse response
	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify response contains feed data
	feedData, ok := result["feed"].([]interface{})
	if !ok || len(feedData) == 0 {
		t.Fatal("Expected feed data in response")
	}
}

// TestTokenManagerAndClientIntegration verifies that the TokenManager properly authenticates and the client it returns is usable
func TestTokenManagerAndClientIntegration(t *testing.T) {
	// Setup mock server
	server, _ := setupMockServer()
	defer server.Close()

	// Create a config pointing to our mock server
	cfg := config.Config{
		BskyHost:     server.URL,
		BskyID:       "test",
		BskyPassword: "password",
	}

	// Get token manager
	tokenManager := auth.GetTokenManager(cfg)
	if tokenManager == nil {
		t.Fatal("Expected a non-nil TokenManager")
	}

	// Get the token
	token, err := tokenManager.GetToken(cfg)
	if err != nil {
		t.Fatalf("GetToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("Expected a non-empty token")
	}

	// Get the client
	client := tokenManager.GetClient()
	if client == nil {
		t.Fatal("Expected a non-nil client from GetClient")
	}

	// Verify that the auth token is set on the client
	if client.AuthToken == "" {
		t.Fatal("Expected auth token to be set on the client")
	}

	// Make a request with the client
	resp, err := client.Get("app.bsky.feed.getTimeline", nil)
	if err != nil {
		t.Fatalf("API request with token manager's client failed: %v", err)
	}

	// Parse response
	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify response contains feed data
	feedData, ok := result["feed"].([]interface{})
	if !ok || len(feedData) == 0 {
		t.Fatal("Expected feed data in response")
	}
}

// FeedTestClient implements BlueskyAPIClient for testing
type FeedTestClient struct {
	GetResponses map[string][]byte
	AuthToken    string
}

func (c *FeedTestClient) Get(endpoint string, params url.Values) ([]byte, error) {
	if resp, ok := c.GetResponses[endpoint]; ok {
		return resp, nil
	}
	return []byte(`{"feed":[]}`), nil
}

func (c *FeedTestClient) Post(endpoint string, body interface{}) ([]byte, error) {
	return []byte(`{}`), nil
}

func (c *FeedTestClient) SetAuthToken(token string) {
	c.AuthToken = token
}

// Simplified integration test that focuses on the HTTP flow and basic response validation
func TestHandlerBasicIntegration(t *testing.T) {
	// Skip for now - this test tries to make real API calls
	t.Skip("Skipping integration test that tries to make real API calls")
	
	// Setup cache
	cacheService := cache.New()
	defer cacheService.Stop()

	// Setup echo framework
	e := echo.New()

	// Register a special handler that uses our mock client
	e.POST("/mcp/feed-analysis", func(c echo.Context) error {
		// Parse JSON-RPC request
		var request struct {
			JSONRPC string                 `json:"jsonrpc"`
			Method  string                 `json:"method"`
			Params  map[string]interface{} `json:"params"`
			ID      int                    `json:"id"`
		}
		
		if err := c.Bind(&request); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid JSON")
		}
		
		// Process only feed-analysis method
		if request.Method != "feed-analysis" {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"jsonrpc": "2.0",
				"error": map[string]interface{}{
					"code":    -32600,
					"message": "Invalid method",
				},
				"id": request.ID,
			})
		}
		
		// Create minimal config
		cfg := config.Config{}
		
		// Call the actual AnalyzeFeed function directly
		result, err := feed.AnalyzeFeed(cfg, request.Params)
		if err != nil {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"jsonrpc": "2.0",
				"error": map[string]interface{}{
					"code":    -32000,
					"message": err.Error(),
				},
				"id": request.ID,
			})
		}
		
		// Return the result
		return c.JSON(http.StatusOK, map[string]interface{}{
			"jsonrpc": "2.0",
			"result":  result,
			"id":      request.ID,
		})
	})
	
	// Test cases for different API calls
	tests := []struct {
		name       string
		request    string
		endpoint   string
		wantCode   int
		wantSource string
		wantCount  int
	}{
		{
			name:       "Timeline feed without hashtag",
			request:    `{"jsonrpc":"2.0","method":"feed-analysis","params":{"limit":10},"id":1}`,
			endpoint:   "/mcp/feed-analysis",
			wantCode:   http.StatusOK,
			wantSource: "api_fresh",
			wantCount:  1,
		},
		{
			name:       "Search feed with hashtag",
			request:    `{"jsonrpc":"2.0","method":"feed-analysis","params":{"hashtag":"golang","limit":10},"id":1}`,
			endpoint:   "/mcp/feed-analysis",
			wantCode:   http.StatusOK,
			wantSource: "api_fresh",
			wantCount:  2,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test request with JSON-RPC format
			req := httptest.NewRequest(http.MethodPost, tt.endpoint, 
				strings.NewReader(tt.request))
			req.Header.Set("Content-Type", "application/json")
			
			// Create a recorder to capture the response
			rec := httptest.NewRecorder()
			
			// Serve the request
			e.ServeHTTP(rec, req)
			
			// Check status code
			if rec.Code != tt.wantCode {
				t.Errorf("Expected status code %d, got %d", tt.wantCode, rec.Code)
			}
			
			// Parse response to verify structure
			var resp map[string]interface{}
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Errorf("Failed to parse response JSON: %v", err)
				return
			}
			
			// Verify the response has expected JSON-RPC format
			if _, ok := resp["jsonrpc"]; !ok {
				t.Errorf("Missing jsonrpc field in response")
			}
			
			if _, ok := resp["id"]; !ok {
				t.Errorf("Missing id field in response")
			}
			
			// For successful responses, check that result contains expected fields
			if rec.Code == http.StatusOK {
				result, ok := resp["result"].(map[string]interface{})
				if !ok {
					t.Errorf("Result is not a JSON object")
					return
				}
				
				// Verify count field is present
				count, ok := result["count"].(float64)
				if !ok {
					t.Errorf("Missing or invalid count field in result")
				} else if int(count) != tt.wantCount {
					t.Errorf("Expected count %d, got %d", tt.wantCount, int(count))
				}
				
				// Verify posts field is present and is an array
				posts, ok := result["posts"].([]interface{})
				if !ok {
					t.Errorf("Posts field is not an array")
				} else if len(posts) != tt.wantCount {
					t.Errorf("Expected %d posts, got %d", tt.wantCount, len(posts))
				}
				
				// Verify source matches expected value
				source, ok := result["source"].(string)
				if !ok || source != tt.wantSource {
					t.Errorf("Expected source %s, got %v", tt.wantSource, source)
				}
			}
		})
	}
}

// MockAuthService for testing
type MockAuthService struct {
	token string
}

func (m *MockAuthService) GetToken() string {
	return m.token
}

func init() {
	// Replace the GetToken function with our mock during tests
	auth.GetToken = func(cfg config.Config) (string, error) {
		return "mock-token", nil
	}
}