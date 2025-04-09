package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/littleironwaltz/bluesky-mcp/internal/models"
	"github.com/littleironwaltz/bluesky-mcp/pkg/config"
	"github.com/labstack/echo/v4"
)

func TestRateLimiter(t *testing.T) {
	// Create a new rate limiter with smaller limits for testing
	rl := &RateLimiter{
		requests:      make(map[string][]time.Time),
		windowSize:    100 * time.Millisecond,
		maxRequests:   3, // Only allow 3 requests per window
		cleanupPeriod: 200 * time.Millisecond,
		lastCleanup:   time.Now(),
	}

	// Test requests within limits
	for i := 0; i < 3; i++ {
		if !rl.Allow("127.0.0.1") {
			t.Errorf("Expected request %d to be allowed", i+1)
		}
	}

	// The next request should be denied (over limit)
	if rl.Allow("127.0.0.1") {
		t.Errorf("Expected request to be denied (over limit)")
	}

	// Different IP should be allowed
	if !rl.Allow("127.0.0.2") {
		t.Errorf("Expected request from different IP to be allowed")
	}

	// Wait for window to expire
	time.Sleep(110 * time.Millisecond)

	// Should be allowed again
	if !rl.Allow("127.0.0.1") {
		t.Errorf("Expected request to be allowed after window expiry")
	}

	// Test cleanup
	time.Sleep(210 * time.Millisecond)
	
	// Trigger cleanup by making a request
	rl.Allow("127.0.0.3")
	
	// Check if old entries were cleaned up
	rl.mu.Lock()
	_, exists := rl.requests["127.0.0.1"]
	rl.mu.Unlock()
	
	if exists {
		t.Errorf("Expected old entries to be cleaned up")
	}
}

func TestHandleMCPRequestValidationErrors(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		requestBody    string
		wantStatusCode int
		wantErrorCode  string
	}{
		{
			name:           "Invalid method",
			method:         "invalid-method",
			requestBody:    `{"jsonrpc": "2.0", "method": "invalid-method", "params": {}, "id": 1}`,
			wantStatusCode: http.StatusBadRequest,
			wantErrorCode:  models.ErrInvalidRequest,
		},
		{
			name:           "Invalid JSON-RPC version",
			method:         "feed-analysis",
			requestBody:    `{"jsonrpc": "1.0", "method": "feed-analysis", "params": {}, "id": 1}`,
			wantStatusCode: http.StatusBadRequest,
			wantErrorCode:  models.ErrInvalidRequest,
		},
		{
			name:           "Invalid JSON format",
			method:         "feed-analysis",
			requestBody:    `{invalid json}`,
			wantStatusCode: http.StatusBadRequest,
			wantErrorCode:  models.ErrInvalidRequest,
		},
		{
			name:           "Post submit method with missing text",
			method:         "post-submit",
			requestBody:    `{"jsonrpc": "2.0", "method": "post-submit", "params": {}, "id": 1}`,
			wantStatusCode: http.StatusBadRequest,
			wantErrorCode:  models.ErrInvalidParams,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup Echo
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.requestBody))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath("/mcp/:method")
			c.SetParamNames("method")
			c.SetParamValues(tt.method)

			// Execute the handler
			cfg := config.Config{}
			err := HandleMCPRequest(c, cfg)

			// Check if the handler returned an error
			if err != nil {
				t.Errorf("HandleMCPRequest() returned error: %v", err)
				return
			}

			// Check status code
			if rec.Code != tt.wantStatusCode {
				t.Errorf("HandleMCPRequest() status code = %v, want %v", rec.Code, tt.wantStatusCode)
			}

			// Check error code in response
			var response models.JSONRPCResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				t.Errorf("Failed to unmarshal response: %v", err)
				return
			}

			if response.Error == nil {
				t.Errorf("Expected error in response, got nil")
				return
			}

			if response.Error.Code != tt.wantErrorCode {
				t.Errorf("Error code = %v, want %v", response.Error.Code, tt.wantErrorCode)
			}
		})
	}
}

func TestHandleMethodError(t *testing.T) {
	tests := []struct {
		name           string
		errString      string
		wantStatusCode int
		wantErrorCode  string
	}{
		{
			name:           "Timeout error",
			errString:      "timeout processing 'feed-analysis' request",
			wantStatusCode: http.StatusGatewayTimeout,
			wantErrorCode:  models.ErrTimeout,
		},
		{
			name:           "Authentication error",
			errString:      "authentication failed",
			wantStatusCode: http.StatusUnauthorized,
			wantErrorCode:  models.ErrAuthenticationError,
		},
		{
			name:           "Not found error",
			errString:      "resource not found",
			wantStatusCode: http.StatusNotFound,
			wantErrorCode:  models.ErrNotFound,
		},
		{
			name:           "Invalid parameter error",
			errString:      "invalid parameter",
			wantStatusCode: http.StatusBadRequest,
			wantErrorCode:  models.ErrInvalidParams,
		},
		{
			name:           "API error",
			errString:      "API error",
			wantStatusCode: http.StatusBadGateway,
			wantErrorCode:  models.ErrAPIError,
		},
		{
			name:           "Failed to create post error",
			errString:      "failed to create post",
			wantStatusCode: http.StatusBadGateway,
			wantErrorCode:  models.ErrAPIError,
		},
		{
			name:           "Generic error",
			errString:      "something went wrong",
			wantStatusCode: http.StatusInternalServerError,
			wantErrorCode:  models.ErrInternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup Echo
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Execute the handler
			err := handleMethodError(c, echo.NewHTTPError(http.StatusInternalServerError, tt.errString), 1)

			// Check if the handler returned an error
			if err != nil {
				t.Errorf("handleMethodError() returned error: %v", err)
				return
			}

			// Check status code
			if rec.Code != tt.wantStatusCode {
				t.Errorf("handleMethodError() status code = %v, want %v", rec.Code, tt.wantStatusCode)
			}

			// Check error code in response
			var response models.JSONRPCResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				t.Errorf("Failed to unmarshal response: %v", err)
				return
			}

			if response.Error == nil {
				t.Errorf("Expected error in response, got nil")
				return
			}

			if response.Error.Code != tt.wantErrorCode {
				t.Errorf("Error code = %v, want %v", response.Error.Code, tt.wantErrorCode)
			}

			// Check request ID is correct
			if response.ID != 1 {
				t.Errorf("Response ID = %v, want %v", response.ID, 1)
			}
		})
	}
}