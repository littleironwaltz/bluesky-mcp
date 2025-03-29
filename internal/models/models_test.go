package models

import (
	"reflect"
	"testing"
)

func TestNewErrorResponse(t *testing.T) {
	tests := []struct {
		name    string
		id      int
		code    string
		message string
		want    JSONRPCResponse
	}{
		{
			name:    "Basic error response",
			id:      1,
			code:    ErrInvalidRequest,
			message: "Invalid request",
			want: JSONRPCResponse{
				JSONRPC: "2.0",
				Error: &ErrorInfo{
					Code:    ErrInvalidRequest,
					Message: "Invalid request",
				},
				ID: 1,
			},
		},
		{
			name:    "Authentication error",
			id:      2,
			code:    ErrAuthenticationError,
			message: "Authentication failed",
			want: JSONRPCResponse{
				JSONRPC: "2.0",
				Error: &ErrorInfo{
					Code:    ErrAuthenticationError,
					Message: "Authentication failed",
				},
				ID: 2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewErrorResponse(tt.id, tt.code, tt.message)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewErrorResponse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewDetailedErrorResponse(t *testing.T) {
	tests := []struct {
		name    string
		id      int
		code    string
		message string
		details string
		want    JSONRPCResponse
	}{
		{
			name:    "Detailed error response",
			id:      1,
			code:    ErrInternalError,
			message: "Internal server error",
			details: "Database connection failed",
			want: JSONRPCResponse{
				JSONRPC: "2.0",
				Error: &ErrorInfo{
					Code:    ErrInternalError,
					Message: "Internal server error",
					Details: "Database connection failed",
				},
				ID: 1,
			},
		},
		{
			name:    "Service unavailable error",
			id:      2,
			code:    ErrServiceUnavailable,
			message: "Service is temporarily unavailable",
			details: "Expected to be back online at 14:00 UTC",
			want: JSONRPCResponse{
				JSONRPC: "2.0",
				Error: &ErrorInfo{
					Code:    ErrServiceUnavailable,
					Message: "Service is temporarily unavailable",
					Details: "Expected to be back online at 14:00 UTC",
				},
				ID: 2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewDetailedErrorResponse(tt.id, tt.code, tt.message, tt.details)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewDetailedErrorResponse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPostStruct(t *testing.T) {
	// Test creating and manipulating Post struct
	post := Post{
		ID:        "123abc",
		Text:      "Hello, world!",
		CreatedAt: "2023-01-01T12:00:00Z",
		Author:    "user.bsky.social",
		Metrics: map[string]int{
			"length": 13,
			"words":  2,
		},
		Analysis: map[string]string{
			"sentiment": "positive",
		},
	}

	// Verify fields are set correctly
	if post.ID != "123abc" {
		t.Errorf("Post.ID = %v, want %v", post.ID, "123abc")
	}

	if post.Text != "Hello, world!" {
		t.Errorf("Post.Text = %v, want %v", post.Text, "Hello, world!")
	}

	if post.CreatedAt != "2023-01-01T12:00:00Z" {
		t.Errorf("Post.CreatedAt = %v, want %v", post.CreatedAt, "2023-01-01T12:00:00Z")
	}

	if post.Author != "user.bsky.social" {
		t.Errorf("Post.Author = %v, want %v", post.Author, "user.bsky.social")
	}

	// Check metrics
	expectedMetrics := map[string]int{
		"length": 13,
		"words":  2,
	}
	if !reflect.DeepEqual(post.Metrics, expectedMetrics) {
		t.Errorf("Post.Metrics = %v, want %v", post.Metrics, expectedMetrics)
	}

	// Check analysis
	expectedAnalysis := map[string]string{
		"sentiment": "positive",
	}
	if !reflect.DeepEqual(post.Analysis, expectedAnalysis) {
		t.Errorf("Post.Analysis = %v, want %v", post.Analysis, expectedAnalysis)
	}
}

func TestFeedResponse(t *testing.T) {
	// Test creating and manipulating FeedResponse struct
	posts := []Post{
		{
			ID:        "post1",
			Text:      "First post",
			CreatedAt: "2023-01-01T12:00:00Z",
			Author:    "user1.bsky.social",
			Analysis: map[string]string{
				"sentiment": "neutral",
			},
		},
		{
			ID:        "post2",
			Text:      "Second post",
			CreatedAt: "2023-01-02T12:00:00Z",
			Author:    "user2.bsky.social",
			Analysis: map[string]string{
				"sentiment": "positive",
			},
		},
	}

	response := FeedResponse{
		Posts:   posts,
		Count:   len(posts),
		Warning: "Data may be stale",
		Source:  "cache_stale",
	}

	// Verify fields are set correctly
	if response.Count != 2 {
		t.Errorf("FeedResponse.Count = %v, want %v", response.Count, 2)
	}

	if response.Warning != "Data may be stale" {
		t.Errorf("FeedResponse.Warning = %v, want %v", response.Warning, "Data may be stale")
	}

	if response.Source != "cache_stale" {
		t.Errorf("FeedResponse.Source = %v, want %v", response.Source, "cache_stale")
	}

	// Check posts
	if len(response.Posts) != 2 {
		t.Errorf("len(FeedResponse.Posts) = %v, want %v", len(response.Posts), 2)
	}

	if response.Posts[0].ID != "post1" {
		t.Errorf("FeedResponse.Posts[0].ID = %v, want %v", response.Posts[0].ID, "post1")
	}

	if response.Posts[1].ID != "post2" {
		t.Errorf("FeedResponse.Posts[1].ID = %v, want %v", response.Posts[1].ID, "post2")
	}
}