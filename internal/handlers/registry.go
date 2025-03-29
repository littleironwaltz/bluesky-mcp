package handlers

import (
	"github.com/littleironwaltz/bluesky-mcp/internal/cache"
	"github.com/littleironwaltz/bluesky-mcp/internal/services/feed"
	"github.com/littleironwaltz/bluesky-mcp/pkg/apiclient"
	"github.com/littleironwaltz/bluesky-mcp/pkg/config"
	"github.com/labstack/echo/v4"
)

// RegisterHandlers sets up all the handler routes
func RegisterHandlers(e *echo.Echo, client *apiclient.BlueskyClient, cache *cache.Cache, feedService *feed.FeedService) {
	// Setup MCP endpoint
	e.POST("/xrpc/bluesky.mcp.feed.analyze", func(c echo.Context) error {
		// Simple mock implementation for testing
		cfg := config.Config{
			BskyHost: client.BaseURL,
		}
		
		// Parse request and pass to handler
		return HandleMCPRequest(c, cfg)
	})
}