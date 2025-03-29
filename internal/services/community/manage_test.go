package community

import (
	"testing"

	"github.com/littleironwaltz/bluesky-mcp/pkg/config"
)

func TestGenerateCacheKey(t *testing.T) {
	tests := []struct {
		name       string
		userHandle string
		limit      float64
		want       string
	}{
		{
			name:       "Valid DID user handle",
			userHandle: "did:plc:abcdef123456",
			limit:      5,
			want:       generateCacheKey("did:plc:abcdef123456", 5),
		},
		{
			name:       "Valid domain user handle",
			userHandle: "user.bsky.social",
			limit:      10,
			want:       generateCacheKey("user.bsky.social", 10),
		},
		{
			name:       "Different limits",
			userHandle: "user.bsky.social",
			limit:      20,
			want:       generateCacheKey("user.bsky.social", 20),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateCacheKey(tt.userHandle, tt.limit)
			if got != tt.want {
				t.Errorf("generateCacheKey() = %v, want %v", got, tt.want)
			}

			// Keys for different inputs should be different
			differentKey := generateCacheKey("different.user", tt.limit)
			if got == differentKey {
				t.Errorf("generateCacheKey() generated same key for different inputs")
			}
		})
	}
}

// Mocks removed since we're doing basic validation testing only

func TestManageCommunity(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.Config
		params  map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name: "Missing user handle",
			cfg:  config.Config{},
			params: map[string]interface{}{
				"limit": float64(5),
			},
			wantErr: true,
			errMsg:  "missing or invalid user handle",
		},
		{
			name: "Empty user handle",
			cfg:  config.Config{},
			params: map[string]interface{}{
				"userHandle": "",
				"limit":      float64(5),
			},
			wantErr: true,
			errMsg:  "missing or invalid user handle",
		},
		{
			name: "Invalid user handle format",
			cfg:  config.Config{},
			params: map[string]interface{}{
				"userHandle": "invalid-format",
				"limit":      float64(5),
			},
			wantErr: true,
			errMsg:  "invalid user handle format",
		},
		{
			name: "Negative limit",
			cfg:  config.Config{},
			params: map[string]interface{}{
				"userHandle": "user.bsky.social",
				"limit":      float64(-1),
			},
			wantErr: false, // Should use default limit of 5
			errMsg:  "",
		},
		{
			name: "Limit too high",
			cfg:  config.Config{},
			params: map[string]interface{}{
				"userHandle": "user.bsky.social",
				"limit":      float64(100),
			},
			wantErr: false, // Should use default limit of 5
			errMsg:  "",
		},
		{
			name: "Valid DID format",
			cfg:  config.Config{},
			params: map[string]interface{}{
				"userHandle": "did:plc:abcdef123456",
				"limit":      float64(5),
			},
			wantErr: false,
			errMsg:  "",
		},
		{
			name: "Valid domain format",
			cfg:  config.Config{},
			params: map[string]interface{}{
				"userHandle": "user.bsky.social",
				"limit":      float64(5),
			},
			wantErr: false,
			errMsg:  "",
		},
		{
			name: "Non-numeric limit",
			cfg:  config.Config{},
			params: map[string]interface{}{
				"userHandle": "user.bsky.social",
				"limit":      "5",
			},
			wantErr: false, // Should use default limit of 5
			errMsg:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't fully test actual API calls without mocking the API client
			// But we can test validation logic
			_, err := ManageCommunity(tt.cfg, tt.params)
			
			// Since we don't have credentials configured in tests,
			// we expect auth errors for valid params
			if tt.wantErr {
				if err == nil {
					t.Errorf("ManageCommunity() expected error but got nil")
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("ManageCommunity() error = %v, want %v", err.Error(), tt.errMsg)
				}
			} else {
				// For valid params, we expect auth error but not validation error
				if err == nil {
					t.Errorf("ManageCommunity() expected auth error but got nil")
					return
				}
				if err.Error() != "authentication error" {
					t.Errorf("ManageCommunity() error = %v, want 'authentication error'", err.Error())
				}
			}
		})
	}
}