package feed

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"testing"
	"time"
)

func TestValidateParams(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]interface{}
		want    map[string]interface{}
		wantErr bool
	}{
		{
			name:   "Empty params",
			params: map[string]interface{}{},
			want: map[string]interface{}{
				"hashtag": "",
				"limit":   float64(10),
			},
			wantErr: false,
		},
		{
			name: "Valid params",
			params: map[string]interface{}{
				"hashtag": "test",
				"limit":   float64(20),
			},
			want: map[string]interface{}{
				"hashtag": "test",
				"limit":   float64(20),
			},
			wantErr: false,
		},
		{
			name: "Invalid limit type",
			params: map[string]interface{}{
				"hashtag": "test",
				"limit":   "20",
			},
			want: map[string]interface{}{
				"hashtag": "test",
				"limit":   float64(10),
			},
			wantErr: false,
		},
		{
			name: "Limit too high",
			params: map[string]interface{}{
				"hashtag": "test",
				"limit":   float64(200),
			},
			want: map[string]interface{}{
				"hashtag": "test",
				"limit":   float64(10),
			},
			wantErr: false,
		},
		{
			name: "Invalid hashtag type",
			params: map[string]interface{}{
				"hashtag": 123,
				"limit":   float64(20),
			},
			want: map[string]interface{}{
				"hashtag": "",
				"limit":   float64(20),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateParams(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateParams() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("validateParams() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAnalyzeSentiment(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "Positive text",
			text: "I am feeling good and happy today",
			want: "positive",
		},
		{
			name: "Negative text",
			text: "This is terrible and makes me sad",
			want: "negative",
		},
		{
			name: "Neutral text",
			text: "Just sharing some information about the weather",
			want: "neutral",
		},
		{
			name: "Mixed text with more positive",
			text: "Despite the bad weather, I'm happy and excited",
			want: "positive",
		},
		{
			name: "Mixed text with more negative",
			text: "Even though it's a great day, I feel terrible and hate it",
			want: "negative",
		},
		{
			name: "Empty text",
			text: "",
			want: "neutral",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := analyzeSentiment(tt.text); got != tt.want {
				t.Errorf("analyzeSentiment() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateMetrics(t *testing.T) {
	tests := []struct {
		name string
		text string
		want map[string]int
	}{
		{
			name: "Empty text",
			text: "",
			want: map[string]int{"length": 0, "words": 0},
		},
		{
			name: "Single word",
			text: "Hello",
			want: map[string]int{"length": 5, "words": 1},
		},
		{
			name: "Multiple words",
			text: "Hello world, how are you?",
			want: map[string]int{"length": 25, "words": 5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateMetrics(tt.text)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("calculateMetrics() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetPostID(t *testing.T) {
	tests := []struct {
		name string
		uri  string
		want string
	}{
		{
			name: "Valid URI",
			uri:  "at://user.bsky.social/post/3kuznviij5k2z",
			want: "3kuznviij5k2z",
		},
		{
			name: "Empty URI",
			uri:  "",
			want: "",
		},
		{
			name: "URI without trailing ID",
			uri:  "at://user.bsky.social/post/",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getPostID(tt.uri); got != tt.want {
				t.Errorf("getPostID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterPosts(t *testing.T) {
	// Create test posts
	posts := []FeedItem{
		{
			Post: struct {
				URI    string "json:\"uri\""
				Record struct {
					Text      string "json:\"text\""
					CreatedAt string "json:\"createdAt\""
				} "json:\"record\""
				Author struct {
					Handle string "json:\"handle\""
				} "json:\"author\""
			}{
				URI: "at://user.bsky.social/post/1",
				Record: struct {
					Text      string "json:\"text\""
					CreatedAt string "json:\"createdAt\""
				}{
					Text:      "Post with #golang tag",
					CreatedAt: "2023-01-01T00:00:00Z",
				},
				Author: struct {
					Handle string "json:\"handle\""
				}{
					Handle: "user1.bsky.social",
				},
			},
		},
		{
			Post: struct {
				URI    string "json:\"uri\""
				Record struct {
					Text      string "json:\"text\""
					CreatedAt string "json:\"createdAt\""
				} "json:\"record\""
				Author struct {
					Handle string "json:\"handle\""
				} "json:\"author\""
			}{
				URI: "at://user.bsky.social/post/2",
				Record: struct {
					Text      string "json:\"text\""
					CreatedAt string "json:\"createdAt\""
				}{
					Text:      "Post with #javascript tag",
					CreatedAt: "2023-01-02T00:00:00Z",
				},
				Author: struct {
					Handle string "json:\"handle\""
				}{
					Handle: "user2.bsky.social",
				},
			},
		},
		{
			Post: struct {
				URI    string "json:\"uri\""
				Record struct {
					Text      string "json:\"text\""
					CreatedAt string "json:\"createdAt\""
				} "json:\"record\""
				Author struct {
					Handle string "json:\"handle\""
				} "json:\"author\""
			}{
				URI: "at://user.bsky.social/post/3",
				Record: struct {
					Text      string "json:\"text\""
					CreatedAt string "json:\"createdAt\""
				}{
					Text:      "Another post with #golang",
					CreatedAt: "2023-01-03T00:00:00Z",
				},
				Author: struct {
					Handle string "json:\"handle\""
				}{
					Handle: "user3.bsky.social",
				},
			},
		},
	}

	tests := []struct {
		name    string
		feed    []FeedItem
		hashtag string
		limit   int
		want    int
	}{
		{
			name:    "With hashtag, limited posts",
			feed:    posts,
			hashtag: "golang",
			limit:   1,
			want:    1, // Limited to 1
		},
		{
			name:    "With hashtag, all fitting posts",
			feed:    posts,
			hashtag: "golang",
			limit:   10,
			want:    3, // All posts (since we don't filter by content anymore)
		},
		{
			name:    "No filter, all posts up to limit",
			feed:    posts,
			hashtag: "",
			limit:   10,
			want:    3, // All posts
		},
		{
			name:    "No filter, limit results",
			feed:    posts,
			hashtag: "",
			limit:   2,
			want:    2, // Limited to 2
		},
		{
			name:    "Empty feed",
			feed:    []FeedItem{},
			hashtag: "golang",
			limit:   10,
			want:    0, // No posts
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterPosts(tt.feed, tt.hashtag, tt.limit)
			if len(got) != tt.want {
				t.Errorf("filterPosts() returned %v posts, want %v", len(got), tt.want)
			}
		})
	}
}

func TestGenerateCacheKey(t *testing.T) {
	tests := []struct {
		name    string
		hashtag string
		limit   int
		want    string
	}{
		{
			name:    "With hashtag",
			hashtag: "golang",
			limit:   10,
			want:    generateCacheKey("golang", 10),
		},
		{
			name:    "Without hashtag",
			hashtag: "",
			limit:   10,
			want:    generateCacheKey("", 10),
		},
		{
			name:    "Different limits",
			hashtag: "golang",
			limit:   20,
			want:    generateCacheKey("golang", 20),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateCacheKey(tt.hashtag, tt.limit)
			if got != tt.want {
				t.Errorf("generateCacheKey() = %v, want %v", got, tt.want)
			}

			// Keys for different inputs should be different
			if tt.name != "Without hashtag" {
				differentKey := generateCacheKey("different", tt.limit)
				if got == differentKey {
					t.Errorf("generateCacheKey() generated same key for different inputs")
				}
			}
		})
	}
}

// Mock for testing AnalyzeFeed without real API calls
type mockClient struct {
	mockResponse     []byte
	mockError        error
	BaseURL          string
	AuthToken        string
	LastEndpoint     string
	LastQueryParams  map[string]string
	EndpointResponse map[string][]byte
}

func (m *mockClient) Get(endpoint string, query url.Values) ([]byte, error) {
	m.LastEndpoint = endpoint
	m.LastQueryParams = make(map[string]string)
	for k, vs := range query {
		if len(vs) > 0 {
			m.LastQueryParams[k] = vs[0]
		}
	}
	
	// If endpoint-specific responses are configured, use those
	if m.EndpointResponse != nil {
		if resp, ok := m.EndpointResponse[endpoint]; ok {
			return resp, m.mockError
		}
	}
	
	return m.mockResponse, m.mockError
}

func (m *mockClient) Post(endpoint string, body interface{}) ([]byte, error) {
	return m.mockResponse, m.mockError
}

func (m *mockClient) SetAuthToken(token string) {
	m.AuthToken = token
}

func TestProcessPostsParallel(t *testing.T) {
	// Create test feed data JSON for timeline
	timelineJSON := []byte(`{
		"feed": [
			{
				"post": {
					"uri": "at://user.bsky.social/post/1",
					"record": {
						"text": "Post with positive sentiment happy good",
						"createdAt": "2023-01-01T00:00:00Z"
					},
					"author": {
						"handle": "user1.bsky.social"
					}
				}
			},
			{
				"post": {
					"uri": "at://user.bsky.social/post/2",
					"record": {
						"text": "Post with negative sentiment sad bad terrible",
						"createdAt": "2023-01-02T00:00:00Z"
					},
					"author": {
						"handle": "user2.bsky.social"
					}
				}
			},
			{
				"post": {
					"uri": "at://user.bsky.social/post/3",
					"record": {
						"text": "Post with neutral sentiment",
						"createdAt": "2023-01-03T00:00:00Z"
					},
					"author": {
						"handle": "user3.bsky.social"
					}
				}
			}
		]
	}`)
	
	// Create test feed data JSON for search
	searchJSON := []byte(`{
		"posts": [
			{
				"uri": "at://user.bsky.social/post/1",
				"record": {
					"text": "Post with positive sentiment happy good",
					"createdAt": "2023-01-01T00:00:00Z"
				},
				"author": {
					"handle": "user1.bsky.social"
				}
			},
			{
				"uri": "at://user.bsky.social/post/2",
				"record": {
					"text": "Post with negative sentiment sad bad terrible",
					"createdAt": "2023-01-02T00:00:00Z"
				},
				"author": {
					"handle": "user2.bsky.social"
				}
			}
		]
	}`)
	
	tests := []struct {
		name      string
		jsonData  []byte
		hashtag   string
		limit     int
		wantCount int
	}{
		{
			name:      "Timeline format, no hashtag filter",
			jsonData:  timelineJSON,
			hashtag:   "",
			limit:     10,
			wantCount: 3,
		},
		{
			name:      "Timeline format, with limit",
			jsonData:  timelineJSON,
			hashtag:   "",
			limit:     2,
			wantCount: 2,
		},
		{
			name:      "Search format, with hashtag",
			jsonData:  searchJSON,
			hashtag:   "golang",
			limit:     10,
			wantCount: 2,
		},
		{
			name:      "Search format, with limit",
			jsonData:  searchJSON,
			hashtag:   "golang",
			limit:     1,
			wantCount: 1,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := processPostsParallel(tt.jsonData, tt.hashtag, tt.limit)
			
			// Verify count
			if len(results) != tt.wantCount {
				t.Errorf("Expected %d results, got %d", tt.wantCount, len(results))
			}
			
			// Verify first result
			if len(results) > 0 {
				// Verify key fields are populated
				result := results[0]
				
				// Verify ID
				if result.ID == "" {
					t.Errorf("Expected post ID to be populated")
				}
				
				// Verify sentiment analysis
				sentimentValue, ok := result.Analysis["sentiment"]
				if !ok || sentimentValue == "" {
					t.Errorf("Expected sentiment analysis for post %s", result.ID)
				}
				
				// Verify text field is populated
				if result.Text == "" {
					t.Errorf("Expected text to be populated for post %s", result.ID)
				}
				
				// Verify metrics are calculated
				if result.Metrics == nil || len(result.Metrics) == 0 {
					t.Errorf("Expected metrics to be calculated for post %s", result.ID)
				}
				
				// Verify timestamps
				if result.CreatedAt == "" {
					t.Errorf("Expected createdAt to be populated for post %s", result.ID)
				}
				
				// Verify author
				if result.Author == "" {
					t.Errorf("Expected author to be populated for post %s", result.ID)
				}
			}
		})
	}
}

func TestIsFallbackResponse(t *testing.T) {
	tests := []struct {
		name string
		data map[string]interface{}
		want bool
	}{
		{
			name: "Timeline fallback response",
			data: map[string]interface{}{
				"feed": []interface{}{
					map[string]interface{}{
						"post": map[string]interface{}{
							"author": map[string]interface{}{
								"handle": "fallback.system",
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "Timeline regular response",
			data: map[string]interface{}{
				"feed": []interface{}{
					map[string]interface{}{
						"post": map[string]interface{}{
							"author": map[string]interface{}{
								"handle": "user.bsky.social",
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "Search fallback response",
			data: map[string]interface{}{
				"posts": []interface{}{
					map[string]interface{}{
						"author": map[string]interface{}{
							"handle": "fallback.system",
						},
					},
				},
			},
			want: true,
		},
		{
			name: "Search regular response",
			data: map[string]interface{}{
				"posts": []interface{}{
					map[string]interface{}{
						"author": map[string]interface{}{
							"handle": "user.bsky.social",
						},
					},
				},
			},
			want: false,
		},
		{
			name: "Empty response",
			data: map[string]interface{}{},
			want: false,
		},
		{
			name: "Nil response",
			data: nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isFallbackResponse(tt.data)
			if got != tt.want {
				t.Errorf("isFallbackResponse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name    string
		errText string
		want    bool
	}{
		{
			name:    "Timeout error",
			errText: "request timed out: timeout",
			want:    true,
		},
		{
			name:    "Connection refused",
			errText: "connection refused",
			want:    true,
		},
		{
			name:    "No such host",
			errText: "no such host",
			want:    true,
		},
		{
			name:    "EOF error",
			errText: "unexpected EOF",
			want:    true,
		},
		{
			name:    "500 error",
			errText: "request failed with status 500",
			want:    true,
		},
		{
			name:    "503 error",
			errText: "request failed with status 503",
			want:    true,
		},
		{
			name:    "Non-retryable error",
			errText: "invalid request",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := fmt.Errorf(tt.errText)
			got := isRetryableError(err)
			if got != tt.want {
				t.Errorf("isRetryableError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFetchFeed(t *testing.T) {
	timelineJSON := []byte(`{"feed":[{"post":{"uri":"test/uri"}}]}`)
	searchJSON := []byte(`{"posts":[{"uri":"test/uri"}]}`)
	
	tests := []struct {
		name            string
		hashtag         string
		limit           int
		expectedEndpoint string
		expectedQuery   map[string]string
	}{
		{
			name:            "Timeline request without hashtag",
			hashtag:         "",
			limit:           10,
			expectedEndpoint: "app.bsky.feed.getTimeline",
			expectedQuery:   map[string]string{"limit": "10"},
		},
		{
			name:            "Search request with hashtag",
			hashtag:         "golang",
			limit:           20,
			expectedEndpoint: "app.bsky.feed.searchPosts",
			expectedQuery:   map[string]string{"limit": "20", "q": "#golang"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			client := &mockClient{
				EndpointResponse: map[string][]byte{
					"app.bsky.feed.getTimeline": timelineJSON,
					"app.bsky.feed.searchPosts": searchJSON,
				},
			}
			
			// Call fetchFeed
			_, err := fetchFeed(client, tt.hashtag, tt.limit)
			
			// Verify no error
			if err != nil {
				t.Errorf("fetchFeed() error = %v", err)
			}
			
			// Verify endpoint
			if client.LastEndpoint != tt.expectedEndpoint {
				t.Errorf("fetchFeed() used endpoint = %v, want %v", 
					client.LastEndpoint, tt.expectedEndpoint)
			}
			
			// Verify query params
			for k, v := range tt.expectedQuery {
				if client.LastQueryParams[k] != v {
					t.Errorf("fetchFeed() query param %s = %v, want %v", 
						k, client.LastQueryParams[k], v)
				}
			}
		})
	}
}

func TestFetchFeedWithTimeout(t *testing.T) {
	// This test just verifies the function doesn't crash since the real timeout
	// is difficult to test reliably in unit tests without mocking everything
	
	// Create mock client
	client := &mockClient{
		mockResponse: []byte(`{"feed":[]}`),
	}
	
	// Create context with a reasonable timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	
	// Call fetchFeedWithTimeout
	_, err := fetchFeedWithTimeout(ctx, client, "test", 10)
	
	// Just check it doesn't error out
	if err != nil {
		t.Errorf("fetchFeedWithTimeout() unexpected error: %v", err)
	}
}