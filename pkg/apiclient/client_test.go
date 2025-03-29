package apiclient

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	baseURL := "https://example.com"
	client := NewClient(baseURL)

	if client.BaseURL != baseURL {
		t.Errorf("Expected BaseURL %s, got %s", baseURL, client.BaseURL)
	}

	if client.HTTPClient == nil {
		t.Error("Expected HTTPClient to be initialized")
	}

	if client.FallbackResponses == nil {
		t.Error("Expected FallbackResponses to be initialized")
	}
}

func TestSetAuthToken(t *testing.T) {
	client := NewClient("https://example.com")
	token := "test-token"
	
	client.SetAuthToken(token)
	
	if client.AuthToken != token {
		t.Errorf("Expected AuthToken %s, got %s", token, client.AuthToken)
	}
}

func TestSetRetryConfig(t *testing.T) {
	client := NewClient("https://example.com")
	config := RetryConfig{
		MaxRetries:      5,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     2 * time.Second,
		Multiplier:      2.0,
		MaxElapsedTime:  10 * time.Second,
	}
	
	client.SetRetryConfig(config)
	
	if client.RetryConfig.MaxRetries != config.MaxRetries {
		t.Errorf("Expected MaxRetries %d, got %d", config.MaxRetries, client.RetryConfig.MaxRetries)
	}
	if client.RetryConfig.InitialInterval != config.InitialInterval {
		t.Errorf("Expected InitialInterval %v, got %v", config.InitialInterval, client.RetryConfig.InitialInterval)
	}
}

func TestRegisterFallbackResponse(t *testing.T) {
	client := NewClient("https://example.com")
	endpoint := "com.example.test"
	response := []byte(`{"result": "test"}`)
	
	client.RegisterFallbackResponse(endpoint, response)
	
	if string(client.FallbackResponses[endpoint]) != string(response) {
		t.Errorf("Expected fallback response %s, got %s", string(response), string(client.FallbackResponses[endpoint]))
	}
}

func TestGet(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.URL.Path != "/xrpc/com.example.test" {
			t.Errorf("Expected path %s, got %s", "/xrpc/com.example.test", r.URL.Path)
		}
		
		// Check query parameters
		if r.URL.Query().Get("param") != "value" {
			t.Errorf("Expected query param 'param=value', got %s", r.URL.Query().Get("param"))
		}
		
		// Check auth header if provided
		authHeader := r.Header.Get("Authorization")
		if r.URL.Query().Get("auth") == "true" && authHeader != "Bearer test-token" {
			t.Errorf("Expected Authorization header 'Bearer test-token', got %s", authHeader)
		}
		
		// Return response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()
	
	// Create client using test server URL
	client := NewClient(server.URL)
	
	// Test GET request with query parameters
	params := url.Values{}
	params.Add("param", "value")
	
	response, err := client.Get("com.example.test", params)
	
	if err != nil {
		t.Errorf("Get request failed: %v", err)
	}
	
	var result map[string]interface{}
	if err := json.Unmarshal(response, &result); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}
	
	if success, ok := result["success"].(bool); !ok || !success {
		t.Errorf("Expected success: true, got %v", result["success"])
	}
	
	// Test with auth token
	client.SetAuthToken("test-token")
	params.Add("auth", "true")
	
	response, err = client.Get("com.example.test", params)
	
	if err != nil {
		t.Errorf("Get request with auth failed: %v", err)
	}
}

func TestPost(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("Expected method POST, got %s", r.Method)
		}
		
		if r.URL.Path != "/xrpc/com.example.test" {
			t.Errorf("Expected path %s, got %s", "/xrpc/com.example.test", r.URL.Path)
		}
		
		// Check content type
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}
		
		// Parse request body
		var requestBody map[string]interface{}
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&requestBody); err != nil {
			t.Errorf("Failed to parse request body: %v", err)
		}
		
		// Verify request body
		if value, ok := requestBody["key"].(string); !ok || value != "value" {
			t.Errorf("Expected request body {\"key\":\"value\"}, got %v", requestBody)
		}
		
		// Return response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()
	
	// Create client using test server URL
	client := NewClient(server.URL)
	
	// Test POST request
	requestBody := map[string]string{"key": "value"}
	
	response, err := client.Post("com.example.test", requestBody)
	
	if err != nil {
		t.Errorf("Post request failed: %v", err)
	}
	
	var result map[string]interface{}
	if err := json.Unmarshal(response, &result); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}
	
	if success, ok := result["success"].(bool); !ok || !success {
		t.Errorf("Expected success: true, got %v", result["success"])
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name    string
		errText string
		want    bool
	}{
		{
			name:    "HTTP 500 error",
			errText: "request failed with status 500",
			want:    true,
		},
		{
			name:    "HTTP 502 error",
			errText: "request failed with status 502",
			want:    true,
		},
		{
			name:    "HTTP 503 error",
			errText: "request failed with status 503",
			want:    true,
		},
		{
			name:    "HTTP 504 error",
			errText: "request failed with status 504",
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
			name:    "HTTP 400 error (not retryable)",
			errText: "request failed with status 400",
			want:    false,
		},
		{
			name:    "HTTP 404 error (not retryable)",
			errText: "request failed with status 404",
			want:    false,
		},
		{
			name:    "Other error (not retryable)",
			errText: "invalid request",
			want:    false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if strings.Contains(tt.errText, "timeout") || strings.Contains(tt.errText, "temporary") {
				err = &testError{message: tt.errText, timeout: true, temp: true}
			} else {
				err = fmt.Errorf(tt.errText)
			}
			got := isRetryableError(err)
			if got != tt.want {
				t.Errorf("isRetryableError() = %v, want %v", got, tt.want)
			}
		})
	}
}

// testError implements the error interface for testing
type testError struct {
	message string
	timeout bool
	temp    bool
}

func (e *testError) Error() string {
	return e.message
}

func (e *testError) Timeout() bool {
	return e.timeout
}

func (e *testError) Temporary() bool {
	return e.temp
}