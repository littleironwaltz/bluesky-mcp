package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/littleironwaltz/bluesky-mcp/internal/models"
	"github.com/littleironwaltz/bluesky-mcp/internal/services/community"
	"github.com/littleironwaltz/bluesky-mcp/internal/services/feed"
	"github.com/littleironwaltz/bluesky-mcp/internal/services/post"
	"github.com/littleironwaltz/bluesky-mcp/pkg/config"
	"github.com/labstack/echo/v4"
)

// ValidMethods defines the allowed MCP methods
var ValidMethods = map[string]bool{
	"feed-analysis":    true,
	"post-assist":      true,
	"post-submit":      true,
	"community-manage": true,
}

// RateLimiter provides a simple rate limiting mechanism
type RateLimiter struct {
	mu            sync.Mutex
	requests      map[string][]time.Time // Map of IP to request timestamps
	windowSize    time.Duration          // Time window to track
	maxRequests   int                    // Max requests per window
	cleanupPeriod time.Duration          // How often to clean up old entries
	lastCleanup   time.Time              // Last time cleanup was performed
}

// Global rate limiter instance
var rateLimiter = &RateLimiter{
	requests:      make(map[string][]time.Time),
	windowSize:    time.Minute,
	maxRequests:   60, // 60 requests per minute
	cleanupPeriod: 5 * time.Minute,
	lastCleanup:   time.Now(),
}

// Allow checks if a request from the given IP should be allowed
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	now := time.Now()
	
	// Clean up old entries periodically
	if now.Sub(rl.lastCleanup) > rl.cleanupPeriod {
		rl.cleanup(now)
		rl.lastCleanup = now
	}
	
	// Get the list of request times for this IP
	times, exists := rl.requests[ip]
	if !exists {
		times = []time.Time{}
	}
	
	// Remove timestamps outside the window
	cutoff := now.Add(-rl.windowSize)
	validTimes := []time.Time{}
	
	for _, t := range times {
		if t.After(cutoff) {
			validTimes = append(validTimes, t)
		}
	}
	
	// Check if under the limit
	if len(validTimes) >= rl.maxRequests {
		return false
	}
	
	// Add this request
	validTimes = append(validTimes, now)
	rl.requests[ip] = validTimes
	
	return true
}

// cleanup removes old entries from the rate limiter
func (rl *RateLimiter) cleanup(now time.Time) {
	cutoff := now.Add(-rl.windowSize)
	
	for ip, times := range rl.requests {
		validTimes := []time.Time{}
		
		for _, t := range times {
			if t.After(cutoff) {
				validTimes = append(validTimes, t)
			}
		}
		
		if len(validTimes) == 0 {
			delete(rl.requests, ip)
		} else {
			rl.requests[ip] = validTimes
		}
	}
}

// HandleMCPRequest processes MCP (Model Context Protocol) requests
func HandleMCPRequest(c echo.Context, cfg config.Config) error {
	// Get client IP for rate limiting
	ip := c.RealIP()
	
	// Apply rate limiting
	if !rateLimiter.Allow(ip) {
		return respondWithError(c, http.StatusTooManyRequests, models.ErrRateLimited, "Rate limit exceeded", 0)
	}
	
	method := c.Param("method")
	
	// Validate method
	if !ValidMethods[method] {
		return respondWithError(c, http.StatusBadRequest, models.ErrInvalidRequest, 
			fmt.Sprintf("Invalid method: %s", method), 0)
	}

	// Parse request
	var req models.JSONRPCRequest
	if err := c.Bind(&req); err != nil {
		return respondWithError(c, http.StatusBadRequest, models.ErrInvalidRequest, 
			"Invalid request format", 0)
	}

	// Validate JSON-RPC version
	if req.JSONRPC != "2.0" {
		return respondWithError(c, http.StatusBadRequest, models.ErrInvalidRequest, 
			"Unsupported JSON-RPC version", req.ID)
	}

	// Process the MCP method request
	result, err := processMCPMethod(method, req.Params, cfg)
	if err != nil {
		log.Printf("Error processing '%s' request: %v", method, err)
		return handleMethodError(c, err, req.ID)
	}
	
	// Success response
	return c.JSON(http.StatusOK, models.JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      req.ID,
	})
}

// processMCPMethod handles the execution of a specific MCP method with timeout
func processMCPMethod(method string, params map[string]interface{}, cfg config.Config) (interface{}, error) {
	resultCh := make(chan interface{}, 1)
	errCh := make(chan error, 1)
	
	// Set appropriate timeout based on method
	var timeout time.Duration
	switch method {
	case "feed-analysis":
		timeout = 15 * time.Second
	case "post-assist":
		timeout = 5 * time.Second
	case "post-submit":
		timeout = 10 * time.Second
	case "community-manage":
		timeout = 10 * time.Second
	default:
		timeout = 10 * time.Second
	}
	
	// Process in a goroutine
	go func() {
		var result interface{}
		var err error
		
		switch method {
		case "feed-analysis":
			result, err = feed.AnalyzeFeed(cfg, params)
		case "post-assist":
			result, err = post.GeneratePost(cfg, params)
		case "post-submit":
			// For direct post submission
			text, ok := params["text"].(string)
			if !ok || text == "" {
				err = fmt.Errorf("invalid parameter: text is required")
				break
			}
			postResult, postErr := post.SubmitPost(cfg, text)
			if postErr != nil {
				err = postErr
				break
			}
			result = map[string]interface{}{
				"submitted": true,
				"post_uri": postResult.URI,
				"post_cid": postResult.CID,
			}
		case "community-manage":
			result, err = community.ManageCommunity(cfg, params)
		}
		
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- result
	}()
	
	// Wait for result or timeout
	select {
	case result := <-resultCh:
		return result, nil
	case err := <-errCh:
		return nil, err
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout processing '%s' request", method)
	}
}

// handleMethodError categorizes errors and returns an appropriate response
func handleMethodError(c echo.Context, err error, requestID int) error {
	errString := err.Error()
	
	// Check for known error types
	switch {
	case strings.Contains(errString, "timeout"):
		return respondWithError(c, http.StatusGatewayTimeout, models.ErrTimeout, 
			"Request timed out", requestID)
			
	case strings.Contains(errString, "authentication"):
		return respondWithError(c, http.StatusUnauthorized, models.ErrAuthenticationError, 
			"Authentication failed", requestID)
			
	case strings.Contains(errString, "not found") || strings.Contains(errString, "404"):
		return respondWithError(c, http.StatusNotFound, models.ErrNotFound, 
			"Resource not found", requestID)
			
	case strings.Contains(errString, "invalid") || strings.Contains(errString, "parameter") ||
		 strings.Contains(errString, "validation"):
		return respondWithError(c, http.StatusBadRequest, models.ErrInvalidParams, 
			"Invalid parameters", requestID)
			
	case strings.Contains(errString, "server") || strings.Contains(errString, "API error") ||
		 strings.Contains(errString, "status 5") || strings.Contains(errString, "failed to create post"):
		return respondWithError(c, http.StatusBadGateway, models.ErrAPIError, 
			"Upstream API error", requestID)
	
	default:
		return respondWithError(c, http.StatusInternalServerError, models.ErrInternalError, 
			"Internal server error", requestID)
	}
}

// respondWithError creates a standardized error response
func respondWithError(c echo.Context, httpStatus int, errorCode, message string, id int) error {
	// Log all errors except rate limits (to avoid log spam)
	if errorCode != models.ErrRateLimited {
		log.Printf("Error response: %s - %s", errorCode, message)
	}
	
	// For 5xx errors, use detailed error format with timestamp
	if httpStatus >= 500 {
		timestamp := time.Now().Format(time.RFC3339)
		details := fmt.Sprintf("Error occurred at %s, please try again later", timestamp)
		return c.JSON(httpStatus, models.NewDetailedErrorResponse(id, errorCode, message, details))
	}
	
	return c.JSON(httpStatus, models.NewErrorResponse(id, errorCode, message))
}