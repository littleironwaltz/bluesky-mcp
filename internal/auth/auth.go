package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/littleironwaltz/bluesky-mcp/pkg/apiclient"
	"github.com/littleironwaltz/bluesky-mcp/pkg/config"
)

// Session represents the response from the createSession endpoint
type Session struct {
	AccessJWT  string    `json:"accessJwt"`
	RefreshJWT string    `json:"refreshJwt"`
	Handle     string    `json:"handle"`
	DID        string    `json:"did"`
	ExpiresAt  time.Time // Local expiration tracking (not from API)
}

// TokenManager handles authentication token lifecycle
type TokenManager struct {
	client         *apiclient.BlueskyClient
	session        Session
	mutex          sync.RWMutex
	refreshing     bool
	refreshCtx     context.Context
	refreshCancel  context.CancelFunc
	retryConfig    RetryConfig
	sessionLock    sync.Mutex
	refreshBackoff backoff.BackOff
}

// RetryConfig defines retry behavior for authentication
type RetryConfig struct {
	MaxRetries      int
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
	MaxElapsedTime  time.Duration
}

// DefaultRetryConfig contains default retry settings
var DefaultRetryConfig = RetryConfig{
	MaxRetries:      3,
	InitialInterval: 1 * time.Second,
	MaxInterval:     30 * time.Second,
	Multiplier:      2,
	MaxElapsedTime:  2 * time.Minute,
}

// refreshThreshold is how long before expiration we should refresh
const refreshThreshold = 5 * time.Minute

// Global token manager instance
var (
	manager *TokenManager
	once    sync.Once
)

// BackupCredentials stores alternative authentication credentials
type BackupCredentials struct {
	BskyID       string
	BskyPassword string
	BskyHost     string
}

// Global backup credentials
var backupCredentials []BackupCredentials

// RegisterBackupCredentials registers alternative authentication credentials
func RegisterBackupCredentials(credentials BackupCredentials) {
	if manager != nil {
		manager.mutex.Lock()
		defer manager.mutex.Unlock()
	}
	backupCredentials = append(backupCredentials, credentials)
}

// GetTokenManager returns the shared token manager instance
func GetTokenManager(cfg config.Config) *TokenManager {
	once.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		
		// Initialize exponential backoff
		bOff := backoff.NewExponentialBackOff()
		bOff.InitialInterval = DefaultRetryConfig.InitialInterval
		bOff.MaxInterval = DefaultRetryConfig.MaxInterval
		bOff.Multiplier = DefaultRetryConfig.Multiplier
		bOff.MaxElapsedTime = DefaultRetryConfig.MaxElapsedTime
		
		manager = &TokenManager{
			client:         apiclient.NewClient(cfg.BskyHost),
			refreshCtx:     ctx,
			refreshCancel:  cancel,
			retryConfig:    DefaultRetryConfig,
			refreshBackoff: bOff,
		}
	})
	return manager
}

// GetToken returns a valid authentication token, creating/refreshing a session if needed
var GetToken = func(cfg config.Config) (string, error) {
	return GetTokenManager(cfg).GetToken(cfg)
}

// GetToken returns a valid authentication token
func (tm *TokenManager) GetToken(cfg config.Config) (string, error) {
	// Try to get token with read lock first
	tm.mutex.RLock()
	token, valid := tm.getValidTokenUnlocked()
	tm.mutex.RUnlock()

	if valid {
		return token, nil
	}

	// If we get here, we need a write lock
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	// Double-check token validity after acquiring write lock
	token, valid = tm.getValidTokenUnlocked()
	if valid {
		return token, nil
	}

	// Try to refresh if we have a refresh token
	if tm.session.RefreshJWT != "" {
		err := tm.refreshSessionWithRetries(cfg)
		if err == nil {
			return tm.session.AccessJWT, nil
		}
		// Fall back to creating a new session on failure
	}

	// Create a new session with retries
	return tm.createSessionWithRetries(cfg)
}

// GetClient returns the token manager's client instance
func (tm *TokenManager) GetClient() *apiclient.BlueskyClient {
	return tm.client
}

// GetDID returns the authenticated user's DID
func (tm *TokenManager) GetDID() string {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()
	return tm.session.DID
}

// getValidTokenUnlocked checks if we have a valid token (must be called with lock held)
func (tm *TokenManager) getValidTokenUnlocked() (string, bool) {
	if tm.session.AccessJWT == "" {
		return "", false
	}

	now := time.Now()
	// If token is valid but close to expiration, schedule refresh
	if now.Add(refreshThreshold).After(tm.session.ExpiresAt) && now.Before(tm.session.ExpiresAt) && !tm.refreshing {
		// Don't wait for refresh, return current token and refresh in background
		go tm.refreshInBackground()
	}

	// Check if token is still valid
	if now.Before(tm.session.ExpiresAt) {
		if isValidJWT(tm.session.AccessJWT) {
			return tm.session.AccessJWT, true
		}
	}

	return "", false
}

// refreshInBackground refreshes the token in the background
func (tm *TokenManager) refreshInBackground() {
	tm.mutex.Lock()
	if tm.refreshing {
		tm.mutex.Unlock()
		return
	}
	
	tm.refreshing = true
	refreshToken := tm.session.RefreshJWT
	host := tm.client.BaseURL 
	tm.mutex.Unlock()

	defer func() {
		tm.mutex.Lock()
		tm.refreshing = false
		tm.mutex.Unlock()
	}()

	// Don't proceed if there's no refresh token
	if refreshToken == "" {
		return
	}

	// Create temporary client for refresh to avoid modifying the shared one
	client := apiclient.NewClient(host)
	
	// Create refresh backoff
	bOff := backoff.NewExponentialBackOff()
	bOff.InitialInterval = tm.retryConfig.InitialInterval
	bOff.MaxInterval = tm.retryConfig.MaxInterval
	bOff.Multiplier = tm.retryConfig.Multiplier
	bOff.MaxElapsedTime = tm.retryConfig.MaxElapsedTime
	
	// Create temporary TokenManager for refresh to avoid modifying the main one's backoff
	tempManager := &TokenManager{
		client:      client,
		retryConfig: tm.retryConfig,
	}
	
	// Try to refresh with retries
	_ = tempManager.retryOperation(func() error {
		// Refresh the session
		reqBody := map[string]string{
			"refreshJwt": refreshToken,
		}

		responseBody, err := client.Post("com.atproto.server.refreshSession", reqBody)
		if err != nil {
			return err
		}

		var session Session
		if err := json.Unmarshal(responseBody, &session); err != nil {
			return err
		}

		// Update the session
		session.ExpiresAt = time.Now().Add(1 * time.Hour)
		
		tm.mutex.Lock()
		tm.session = session
		tm.mutex.Unlock()
		
		return nil
	})
}

// refreshSessionWithRetries refreshes a session with retry logic
func (tm *TokenManager) refreshSessionWithRetries(cfg config.Config) error {
	tm.sessionLock.Lock()
	defer tm.sessionLock.Unlock()
	
	// Use the shared retry operation helper
	return tm.retryOperation(func() error {
		return tm.refreshSessionUnlocked(cfg)
	})
}

// retryOperation executes an operation with exponential backoff retry logic
func (tm *TokenManager) retryOperation(operation func() error) error {
	bOff := backoff.NewExponentialBackOff()
	bOff.InitialInterval = tm.retryConfig.InitialInterval
	bOff.MaxInterval = tm.retryConfig.MaxInterval
	bOff.Multiplier = tm.retryConfig.Multiplier
	bOff.MaxElapsedTime = tm.retryConfig.MaxElapsedTime
	
	return backoff.Retry(func() error {
		err := operation()
		if err != nil && isRetryableError(err) {
			return err // Retry on retryable errors
		}
		if err != nil {
			return backoff.Permanent(err) // Don't retry on non-retryable errors
		}
		return nil // Success
	}, bOff)
}

// createSessionWithRetries creates a new session with retry logic
func (tm *TokenManager) createSessionWithRetries(cfg config.Config) (string, error) {
	tm.sessionLock.Lock()
	defer tm.sessionLock.Unlock()
	
	// Try with main credentials first
	var token string
	err := tm.retryOperation(func() error {
		var operationErr error
		token, operationErr = tm.createSessionUnlocked(cfg)
		return operationErr
	})
	
	// If main credentials succeeded, return the token
	if err == nil {
		return token, nil
	}
	
	// If main credentials failed, try backup credentials
	if len(backupCredentials) > 0 {
		for _, backupCfg := range backupCredentials {
			// Create temporary config from backup credentials
			tempCfg := config.Config{
				BskyID:       backupCfg.BskyID,
				BskyPassword: backupCfg.BskyPassword,
				BskyHost:     backupCfg.BskyHost,
			}
			
			// If host is empty, use the main host
			if tempCfg.BskyHost == "" {
				tempCfg.BskyHost = cfg.BskyHost
			}
			
			// Try with backup credentials
			backupErr := tm.retryOperation(func() error {
				var operationErr error
				token, operationErr = tm.createSessionUnlocked(tempCfg)
				return operationErr
			})
			
			// If successful with backup, return the token
			if backupErr == nil {
				return token, nil
			}
		}
	}
	
	// All attempts failed
	return token, err
}

// createSessionUnlocked creates a new session (must be called with write lock held)
func (tm *TokenManager) createSessionUnlocked(cfg config.Config) (string, error) {
	// Validate credentials
	if cfg.BskyID == "" || cfg.BskyPassword == "" {
		return "", errors.New("missing Bluesky credentials in configuration")
	}

	// Create session request
	requestBody := map[string]string{
		"identifier": cfg.BskyID,
		"password":   cfg.BskyPassword,
	}

	// Make API request
	responseBody, err := tm.client.Post("com.atproto.server.createSession", requestBody)
	if err != nil {
		return "", fmt.Errorf("authentication failed: %w", err)
	}

	// Parse response
	var session Session
	if err := json.Unmarshal(responseBody, &session); err != nil {
		return "", fmt.Errorf("error parsing session response: %w", err)
	}

	// Set expiration (tokens typically last 2 hours, but we'll use 1 hour to be safe)
	session.ExpiresAt = time.Now().Add(1 * time.Hour)
	
	// Update session
	tm.session = session
	
	// Update client auth token
	tm.client.SetAuthToken(session.AccessJWT)
	
	return session.AccessJWT, nil
}

// refreshSessionUnlocked refreshes an existing session (must be called with write lock held)
func (tm *TokenManager) refreshSessionUnlocked(cfg config.Config) error {
	// Validate refresh token
	if tm.session.RefreshJWT == "" {
		return errors.New("no refresh token available")
	}

	requestBody := map[string]string{
		"refreshJwt": tm.session.RefreshJWT,
	}

	// Make API request
	responseBody, err := tm.client.Post("com.atproto.server.refreshSession", requestBody)
	if err != nil {
		return fmt.Errorf("token refresh failed: %w", err)
	}

	// Parse response
	var session Session
	if err := json.Unmarshal(responseBody, &session); err != nil {
		return fmt.Errorf("error parsing refresh response: %w", err)
	}

	// Set expiration
	session.ExpiresAt = time.Now().Add(1 * time.Hour)
	
	// Update session
	tm.session = session
	
	// Update client auth token
	tm.client.SetAuthToken(session.AccessJWT)
	
	return nil
}

// Stop cancels any background operations
func (tm *TokenManager) Stop() {
	if tm.refreshCancel != nil {
		tm.refreshCancel()
	}
}

// isValidJWT performs a basic check that the JWT has a valid format
func isValidJWT(token string) bool {
	return strings.HasPrefix(token, "eyJ") && len(token) >= 100
}

// isRetryableError determines if an error should trigger a retry
func isRetryableError(err error) bool {
	errStr := err.Error()
	return strings.Contains(errStr, "request failed") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "i/o timeout") ||
		strings.Contains(errStr, "status 500") ||
		strings.Contains(errStr, "status 502") ||
		strings.Contains(errStr, "status 503") ||
		strings.Contains(errStr, "status 504") ||
		strings.Contains(errStr, "EOF")
}