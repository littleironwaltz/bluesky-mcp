// Package apiclient provides API client functionality for Bluesky
package apiclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
)

// RetryConfig defines retry behavior
type RetryConfig struct {
	MaxRetries      int
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
	MaxElapsedTime  time.Duration
}

// CircuitBreakerConfig defines circuit breaker behavior
type CircuitBreakerConfig struct {
	FailureThreshold   int
	ResetTimeout       time.Duration
	SuccessThreshold   int
	LastFailureResetMs int64
}

// BlueskyClient is a client for interacting with the Bluesky API
type BlueskyClient struct {
	BaseURL            string
	HTTPClient         *http.Client
	AuthToken          string
	RetryConfig        RetryConfig
	CircuitBreaker     CircuitBreakerConfig
	mu                 sync.RWMutex
	currentFailures    int
	isCircuitOpen      bool
	circuitLastChecked time.Time
	FallbackResponses  map[string][]byte
}

// ErrCircuitOpen is returned when the circuit breaker is open
var ErrCircuitOpen = errors.New("circuit breaker is open")

// Default configurations
var (
	DefaultRetryConfig = RetryConfig{
		MaxRetries:      3,
		InitialInterval: 500 * time.Millisecond,
		MaxInterval:     5 * time.Second,
		Multiplier:      1.5,
		MaxElapsedTime:  30 * time.Second,
	}

	DefaultCircuitBreakerConfig = CircuitBreakerConfig{
		FailureThreshold:   5,
		ResetTimeout:       30 * time.Second,
		SuccessThreshold:   2,
		LastFailureResetMs: 60 * 1000, // 1 minute
	}
)

// NewClient creates a new BlueskyClient
func NewClient(baseURL string) *BlueskyClient {
	return &BlueskyClient{
		BaseURL:        baseURL,
		HTTPClient:     getHTTPClient(),
		RetryConfig:    DefaultRetryConfig,
		CircuitBreaker: DefaultCircuitBreakerConfig,
		FallbackResponses: make(map[string][]byte),
	}
}

// SetAuthToken sets the authentication token for the client
func (c *BlueskyClient) SetAuthToken(token string) {
	c.AuthToken = token
}

// SetRetryConfig sets the retry configuration
func (c *BlueskyClient) SetRetryConfig(config RetryConfig) {
	c.RetryConfig = config
}

// SetCircuitBreakerConfig sets the circuit breaker configuration
func (c *BlueskyClient) SetCircuitBreakerConfig(config CircuitBreakerConfig) {
	c.CircuitBreaker = config
}

// RegisterFallbackResponse registers a fallback response for an endpoint
func (c *BlueskyClient) RegisterFallbackResponse(endpoint string, response []byte) {
	c.FallbackResponses[endpoint] = response
}

// Get performs a GET request to the specified API endpoint
func (c *BlueskyClient) Get(endpoint string, params url.Values) ([]byte, error) {
	// Construct full URL
	apiURL := fmt.Sprintf("%s/xrpc/%s", c.BaseURL, endpoint)
	if params != nil && len(params) > 0 {
		apiURL = fmt.Sprintf("%s?%s", apiURL, params.Encode())
	}

	// Create request
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set auth token if available
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	// Execute request with retries
	ctx := context.Background()
	return c.executeRequestWithRetries(ctx, req, endpoint)
}

// Post performs a POST request to the specified API endpoint
func (c *BlueskyClient) Post(endpoint string, body interface{}) ([]byte, error) {
	// Marshal request body
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Construct full URL
	apiURL := fmt.Sprintf("%s/xrpc/%s", c.BaseURL, endpoint)

	// Create request
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	// Execute request with retries
	ctx := context.Background()
	return c.executeRequestWithRetries(ctx, req, endpoint)
}

// executeRequestWithRetries executes an HTTP request with built-in retries and circuit breaking
func (c *BlueskyClient) executeRequestWithRetries(ctx context.Context, req *http.Request, endpoint string) ([]byte, error) {
	// Check if circuit breaker is open
	if c.isCircuitBreakerOpen() {
		// Try fallback response if available
		if fallbackResponse, ok := c.FallbackResponses[endpoint]; ok {
			return fallbackResponse, nil
		}
		return nil, ErrCircuitOpen
	}

	// Create exponential backoff
	bOff := backoff.NewExponentialBackOff()
	bOff.InitialInterval = c.RetryConfig.InitialInterval
	bOff.MaxInterval = c.RetryConfig.MaxInterval
	bOff.Multiplier = c.RetryConfig.Multiplier
	bOff.MaxElapsedTime = c.RetryConfig.MaxElapsedTime
	
	var responseBody []byte
	err := backoff.Retry(func() error {
		var err error
		responseBody, err = c.executeRequest(req.Clone(ctx))
		
		// If succeeded, half-close the circuit breaker if it was in a half-open state
		if err == nil {
			c.recordSuccess()
			return nil
		}
		
		// Record failure and possibly open circuit breaker
		c.recordFailure()
		
		// Return errors for retry decision
		if err != nil {
			// Check if the error is retryable (network error or 5xx)
			if isRetryableError(err) {
				return err // Return the error to retry
			}
			return backoff.Permanent(err) // Don't retry other errors
		}
		
		return nil
	}, bOff)

	// If all retries failed but we have a fallback, use it
	if err != nil && c.FallbackResponses[endpoint] != nil {
		return c.FallbackResponses[endpoint], nil
	}

	return responseBody, err
}

// isRetryableError determines if an error should trigger a retry
func isRetryableError(err error) bool {
	// Network errors are generally retryable
	if err, ok := err.(net.Error); ok {
		return err.Temporary() || err.Timeout()
	}
	
	// Check for HTTP status codes in the error message
	errStr := err.Error()
	return strings.Contains(errStr, "status 500") || 
	       strings.Contains(errStr, "status 502") || 
	       strings.Contains(errStr, "status 503") || 
	       strings.Contains(errStr, "status 504") ||
	       strings.Contains(errStr, "connection refused") ||
	       strings.Contains(errStr, "no such host")
}

// isCircuitBreakerOpen checks if the circuit breaker is open
func (c *BlueskyClient) isCircuitBreakerOpen() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	now := time.Now()
	
	// If circuit is open, check if reset timeout has passed
	if c.isCircuitOpen {
		if now.Sub(c.circuitLastChecked) > c.CircuitBreaker.ResetTimeout {
			// Allow a test request (half-open state)
			c.mu.RUnlock()
			c.mu.Lock()
			c.isCircuitOpen = false // Half-open
			c.currentFailures = 0   // Reset failure count
			c.mu.Unlock()
			c.mu.RLock()
			return false
		}
		return true
	}
	
	return false
}

// recordFailure records a failed request and updates circuit breaker state
func (c *BlueskyClient) recordFailure() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.currentFailures++
	
	// If we hit the threshold, open the circuit
	if c.currentFailures >= c.CircuitBreaker.FailureThreshold {
		c.isCircuitOpen = true
		c.circuitLastChecked = time.Now()
	}
}

// recordSuccess records a successful request
func (c *BlueskyClient) recordSuccess() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Reset failure count on successful requests
	c.currentFailures = 0
	
	// If circuit was in half-open state, fully close it
	if c.isCircuitOpen {
		c.isCircuitOpen = false
	}
}

// executeRequest executes an HTTP request and processes the response
func (c *BlueskyClient) executeRequest(req *http.Request) ([]byte, error) {
	// Setup context with timeout
	ctx, cancel := context.WithTimeout(req.Context(), c.HTTPClient.Timeout)
	defer cancel()
	req = req.WithContext(ctx)

	// Execute request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for error status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errorResponse map[string]interface{}
		if err := json.Unmarshal(responseBody, &errorResponse); err == nil {
			return nil, fmt.Errorf("API error (status %d): %v", resp.StatusCode, errorResponse)
		}
		return nil, fmt.Errorf("API error (status %d)", resp.StatusCode)
	}

	return responseBody, nil
}

// Singleton HTTP client
var (
	client *http.Client
	once   sync.Once
)

// getHTTPClient returns the shared HTTP client instance
func getHTTPClient() *http.Client {
	once.Do(func() {
		transport := &http.Transport{
			// Security settings
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
			// Connection pooling settings
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 20,
			IdleConnTimeout:     90 * time.Second,
			// Additional performance settings
			DisableCompression: false,
			ForceAttemptHTTP2:  true,
			// Timeouts
			ResponseHeaderTimeout: 10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}

		// Create the client with the configured transport
		client = &http.Client{
			Transport: transport,
			Timeout:   10 * time.Second,
		}
	})

	return client
}

// For backward compatibility with existing code
func GetClient() *http.Client {
	return getHTTPClient()
}

// GetClientWithTimeout returns a client with a custom timeout
func GetClientWithTimeout(timeout time.Duration) *http.Client {
	client := getHTTPClient()
	clientCopy := *client
	clientCopy.Timeout = timeout
	return &clientCopy
}