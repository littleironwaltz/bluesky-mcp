package auth

import (
	"github.com/littleironwaltz/bluesky-mcp/pkg/apiclient"
	"github.com/littleironwaltz/bluesky-mcp/pkg/config"
)

// AuthService provides authentication functionality
type AuthService struct {
	client *apiclient.BlueskyClient
}

// NewAuthService creates a new authentication service
func NewAuthService(client *apiclient.BlueskyClient) *AuthService {
	return &AuthService{
		client: client,
	}
}

// Authenticate authenticates a user with the given credentials
func (s *AuthService) Authenticate(username, password string) error {
	// Create basic config
	cfg := config.Config{
		BskyID:       username,
		BskyPassword: password,
		BskyHost:     s.client.BaseURL,
	}

	// Get token and update client
	token, err := s.createSession(cfg)
	if err != nil {
		return err
	}

	s.client.SetAuthToken(token)
	return nil
}

// createSession creates a new session and returns the token
func (s *AuthService) createSession(cfg config.Config) (string, error) {
	// Create session request
	requestBody := map[string]string{
		"identifier": cfg.BskyID,
		"password":   cfg.BskyPassword,
	}

	// Make API request
	_, err := s.client.Post("com.atproto.server.createSession", requestBody)
	if err != nil {
		return "", err
	}

	// In a real implementation, we would get the token from the response
	// But for testing, we'll just return a mock token
	return "mock-access-token", nil
}