package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/littleironwaltz/bluesky-mcp/internal/auth"
	"github.com/littleironwaltz/bluesky-mcp/internal/models"
	"github.com/littleironwaltz/bluesky-mcp/internal/services/community"
	"github.com/littleironwaltz/bluesky-mcp/internal/services/feed"
	"github.com/littleironwaltz/bluesky-mcp/internal/services/post"
	"github.com/littleironwaltz/bluesky-mcp/pkg/config"
	"github.com/spf13/cobra"
)

// Version information
const Version = "0.1.0"

func main() {
	// Check if we're running without credentials or with MOCK_MODE env var - use mock mode
	mockMode := false
	if os.Getenv("BSKY_ID") == "" && os.Getenv("BSKY_PASSWORD") == "" {
		mockMode = true
	}
	
	// Also check for explicit MOCK_MODE environment variable
	if os.Getenv("MOCK_MODE") == "1" || os.Getenv("MOCK_MODE") == "true" {
		mockMode = true
	}

	// Create the root command
	rootCmd := &cobra.Command{
		Use:   "bluesky-mcp-cli",
		Short: "Bluesky MCP CLI - Access Bluesky MCP features from command line",
		Long: `A command-line interface for the Bluesky MCP (Model Context Protocol) service.
Provides easy access to post suggestions, feed analysis, and community management features.`,
	}

	// Add subcommands
	rootCmd.AddCommand(assistCmd(mockMode))
	rootCmd.AddCommand(submitCmd(mockMode))
	rootCmd.AddCommand(feedCmd(mockMode))
	rootCmd.AddCommand(communityCmd(mockMode))
	rootCmd.AddCommand(versionCmd())

	// Execute the command
	if err := rootCmd.Execute(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

// assistCmd generates post suggestions based on mood and topic
func assistCmd(mockMode bool) *cobra.Command {
	var mood, topic string
	var outputJSON bool
	var submitDirect bool

	cmd := &cobra.Command{
		Use:   "assist",
		Short: "Generate post suggestions",
		Long:  "Generate post suggestions based on specified mood and topic.",
		Run: func(cmd *cobra.Command, args []string) {
			// Use mock data if in mock mode or testing environment
			if mockMode {
				mockResult := map[string]interface{}{
					"suggestion": "This is a mock suggestion for testing purposes!",
				}
				if mood != "" && topic != "" {
					mockResult["suggestion"] = fmt.Sprintf("Feeling %s about %s! This is a mock suggestion.", mood, topic)
				}
				
				if submitDirect {
					mockResult["submitted"] = true
					mockResult["post_uri"] = "at://fake-user.bsky.social/post/mock123456"
					mockResult["post_cid"] = "bafyreia123456789mock"
				}
				
				if outputJSON {
					jsonOutput, _ := json.MarshalIndent(mockResult, "", "  ")
					fmt.Println(string(jsonOutput))
				} else {
					fmt.Println(mockResult["suggestion"])
					if submitDirect {
						fmt.Println("\nPost submitted successfully!")
						fmt.Println("URI:", mockResult["post_uri"])
					}
				}
				return
			}
			
			// Load configuration
			cfg := config.LoadConfig()

			// Create params
			params := map[string]interface{}{
				"mood":   mood,
				"topic":  topic,
				"submit": submitDirect,
			}

			// Call the service function
			result, err := post.GeneratePost(cfg, params)
			if err != nil {
				fmt.Printf("Error: %s\n", formatUserFriendlyError(err, "assist"))
				return
			}

			// Handle different result types depending on whether post was submitted
			if submitDirect {
				// For direct submit, result should be a map[string]interface{}
				if resultMap, ok := result.(map[string]interface{}); ok {
					if outputJSON {
						jsonOutput, err := json.MarshalIndent(resultMap, "", "  ")
						if err != nil {
							fmt.Println("Error formatting JSON:", err)
							return
						}
						fmt.Println(string(jsonOutput))
					} else {
						fmt.Println(resultMap["suggestion"])
						
						if submitted, ok := resultMap["submitted"].(bool); ok && submitted {
							fmt.Println("\nPost submitted successfully!")
							if uri, ok := resultMap["post_uri"].(string); ok {
								fmt.Println("URI:", uri)
							}
						} else if errMsg, ok := resultMap["error"].(string); ok {
							fmt.Println("\nFailed to submit post:", errMsg)
						}
					}
				} else {
					fmt.Println("Error: Unexpected response format")
				}
			} else {
				// For suggestion only, result should be a map[string]string
				if suggestion, ok := result.(map[string]string); ok {
					if outputJSON {
						jsonOutput, err := json.MarshalIndent(suggestion, "", "  ")
						if err != nil {
							fmt.Println("Error formatting JSON:", err)
							return
						}
						fmt.Println(string(jsonOutput))
					} else {
						fmt.Println(suggestion["suggestion"])
					}
				} else {
					fmt.Println("Error: Unexpected response format")
				}
			}
		},
	}

	// Add flags
	cmd.Flags().StringVar(&mood, "mood", "", "Mood for the post (e.g., happy, sad, excited, thoughtful)")
	cmd.Flags().StringVar(&topic, "topic", "", "Topic for the post")
	cmd.Flags().BoolVar(&outputJSON, "json", false, "Output in JSON format")
	cmd.Flags().BoolVar(&submitDirect, "submit", false, "Submit the generated post directly to Bluesky")

	// Mark required flags
	cmd.MarkFlagRequired("mood")
	cmd.MarkFlagRequired("topic")

	return cmd
}

// feedCmd analyzes posts with a specified hashtag
func feedCmd(mockMode bool) *cobra.Command {
	var hashtag string
	var limit int
	var outputJSON bool

	cmd := &cobra.Command{
		Use:   "feed",
		Short: "Analyze hashtag feed",
		Long:  "Analyze posts with a specified hashtag and display analysis results.",
		Run: func(cmd *cobra.Command, args []string) {
			// Use mock data if in mock mode or testing environment
			if mockMode {
				mockPosts := []models.Post{
					{
						ID:        "abc123",
						Text:      fmt.Sprintf("This is a sample post about #%s", hashtag),
						CreatedAt: "2025-04-04T13:45:00Z",
						Author:    "test.user.bsky.social",
						Analysis:  map[string]string{"sentiment": "positive"},
						Metrics:   map[string]int{"length": 35, "words": 7},
					},
					{
						ID:        "def456",
						Text:      fmt.Sprintf("Another example post mentioning #%s with some content", hashtag),
						CreatedAt: "2025-04-04T13:40:00Z",
						Author:    "another.user.bsky.social",
						Analysis:  map[string]string{"sentiment": "neutral"},
						Metrics:   map[string]int{"length": 53, "words": 9},
					},
				}
				
				// Limit the number of mock posts based on the limit parameter
				if len(mockPosts) > limit {
					mockPosts = mockPosts[:limit]
				}
				
				mockResponse := models.FeedResponse{
					Posts:  mockPosts,
					Count:  len(mockPosts),
					Source: "mock_data",
				}
				
				if outputJSON {
					jsonOutput, _ := json.MarshalIndent(mockResponse, "", "  ")
					fmt.Println(string(jsonOutput))
				} else {
					displayFeedResults(mockResponse)
				}
				return
			}
			
			// Load configuration
			cfg := config.LoadConfig()

			// Create params
			params := map[string]interface{}{
				"hashtag": hashtag,
				"limit":   float64(limit), // API expects float64
			}

			// Get auth token first to ensure we're authenticated
			_, err := auth.GetToken(cfg)
			if err != nil {
				fmt.Printf("Error: %s\n", formatUserFriendlyError(err, "feed"))
				return
			}

			// Call the service function
			result, err := feed.AnalyzeFeed(cfg, params)
			if err != nil {
				fmt.Printf("Error: %s\n", formatUserFriendlyError(err, "feed"))
				return
			}

			// Check if result is of the expected type
			feedResponse, ok := result.(models.FeedResponse)
			if !ok {
				fmt.Println("Error: Unexpected response format")
				return
			}

			// Output format handling
			if outputJSON {
				jsonOutput, err := json.MarshalIndent(feedResponse, "", "  ")
				if err != nil {
					fmt.Println("Error formatting JSON:", err)
					return
				}
				fmt.Println(string(jsonOutput))
			} else {
				// Display in user-friendly tabular format
				displayFeedResults(feedResponse)
			}
		},
	}

	// Add flags
	cmd.Flags().StringVar(&hashtag, "hashtag", "", "Hashtag to analyze (required)")
	cmd.Flags().IntVar(&limit, "limit", 10, "Number of posts to analyze (max 100)")
	cmd.Flags().BoolVar(&outputJSON, "json", false, "Output in JSON format")

	// Mark required flags
	cmd.MarkFlagRequired("hashtag")

	return cmd
}

// communityCmd displays recent posts from a specified user
func communityCmd(mockMode bool) *cobra.Command {
	var user string
	var limit int
	var outputJSON bool

	cmd := &cobra.Command{
		Use:   "community",
		Short: "Monitor user activity",
		Long:  "Display recent posts from a specified user.",
		Run: func(cmd *cobra.Command, args []string) {
			// Use mock data if in mock mode or testing environment
			if mockMode {
				mockPosts := []string{
					fmt.Sprintf("Hello world! This is a test post from %s", user),
					fmt.Sprintf("Another sample post from %s talking about something interesting", user),
					"Just sharing some thoughts with everyone today!",
				}
				
				// Limit the number of mock posts based on the limit parameter
				if len(mockPosts) > limit {
					mockPosts = mockPosts[:limit]
				}
				
				mockResult := map[string]interface{}{
					"user":        user,
					"recentPosts": mockPosts,
					"count":       len(mockPosts),
				}
				
				if outputJSON {
					jsonOutput, _ := json.MarshalIndent(mockResult, "", "  ")
					fmt.Println(string(jsonOutput))
				} else {
					displayCommunityResults(mockResult)
				}
				return
			}
			
			// Load configuration
			cfg := config.LoadConfig()

			// Create params
			params := map[string]interface{}{
				"userHandle": user,
				"limit":      float64(limit), // API expects float64
			}

			// Get auth token first to ensure we're authenticated
			_, err := auth.GetToken(cfg)
			if err != nil {
				fmt.Printf("Error: %s\n", formatUserFriendlyError(err, "community"))
				return
			}

			// Call the service function
			result, err := community.ManageCommunity(cfg, params)
			if err != nil {
				fmt.Printf("Error: %s\n", formatUserFriendlyError(err, "community"))
				return
			}

			// Output format handling
			if outputJSON {
				jsonOutput, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					fmt.Println("Error formatting JSON:", err)
					return
				}
				fmt.Println(string(jsonOutput))
			} else {
				// Display in user-friendly list format
				displayCommunityResults(result)
			}
		},
	}

	// Add flags
	cmd.Flags().StringVar(&user, "user", "", "Username (format: username.bsky.social)")
	cmd.Flags().IntVar(&limit, "limit", 5, "Number of posts to display (max 50)")
	cmd.Flags().BoolVar(&outputJSON, "json", false, "Output in JSON format")

	// Mark required flags
	cmd.MarkFlagRequired("user")

	return cmd
}

// versionCmd displays the current version
func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Display version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Bluesky MCP CLI v%s\n", Version)
		},
	}
}

// displayFeedResults formats and displays feed analysis results in a user-friendly way
func displayFeedResults(feed models.FeedResponse) {
	if len(feed.Posts) == 0 {
		fmt.Println("No posts found matching your criteria.")
		return
	}

	// Create tabwriter for aligned output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	
	// Print warning if present
	if feed.Warning != "" {
		fmt.Fprintf(w, "Note: %s\n\n", feed.Warning)
	}
	
	// Print header
	fmt.Fprintf(w, "Posts with hashtag (total: %d):\n\n", feed.Count)
	
	// Print posts
	for _, post := range feed.Posts {
		// Truncate text if too long
		text := post.Text
		if len(text) > 60 {
			text = text[:57] + "..."
		}
		
		// Get sentiment in a user-friendly way
		sentiment := "Neutral"
		if s, ok := post.Analysis["sentiment"]; ok {
			switch s {
			case "positive":
				sentiment = "Positive"
			case "negative":
				sentiment = "Negative"
			}
		}
		
		// Format post info
		fmt.Fprintf(w, "Post: %s\nBy: %s\nFeeling: %s\nWords: %d\n\n", 
			text, 
			post.Author, 
			sentiment, 
			post.Metrics["words"])
	}
	
	w.Flush()
}

// displayCommunityResults formats and displays community results in a user-friendly way
func displayCommunityResults(result interface{}) {
	// Extract data from result
	data, ok := result.(map[string]interface{})
	if !ok {
		fmt.Println("Error: Unexpected response format")
		return
	}

	// Extract user and posts
	user, _ := data["user"].(string)
	count, _ := data["count"].(int)
	posts, ok := data["recentPosts"].([]string)
	if !ok {
		posts2, ok := data["recentPosts"].([]interface{})
		if !ok {
			fmt.Println("Error: Could not extract posts from response")
			return
		}
		// Convert interface slice to string slice
		posts = make([]string, len(posts2))
		for i, v := range posts2 {
			if s, ok := v.(string); ok {
				posts[i] = s
			}
		}
	}

	// Print header
	fmt.Printf("Recent posts by %s (total: %d):\n\n", user, count)

	// Print posts in a numbered list
	if len(posts) == 0 {
		fmt.Println("No recent posts found.")
		return
	}

	for i, post := range posts {
		// Truncate text if too long
		if len(post) > 70 {
			post = post[:67] + "..."
		}
		fmt.Printf("%d. %s\n", i+1, post)
	}
}

// formatUserFriendlyError converts technical errors into user-friendly messages
// submitCmd submits a post directly to Bluesky
func submitCmd(mockMode bool) *cobra.Command {
	var text string
	var outputJSON bool

	cmd := &cobra.Command{
		Use:   "submit",
		Short: "Submit a post to Bluesky",
		Long:  "Submit a post directly to your Bluesky account.",
		Run: func(cmd *cobra.Command, args []string) {
			// Use mock data if in mock mode or testing environment
			if mockMode {
				mockResult := map[string]interface{}{
					"submitted": true,
					"text": text,
					"post_uri": "at://fake-user.bsky.social/post/mock123456",
					"post_cid": "bafyreia123456789mock",
				}
				
				if outputJSON {
					jsonOutput, _ := json.MarshalIndent(mockResult, "", "  ")
					fmt.Println(string(jsonOutput))
				} else {
					fmt.Println("Post submitted successfully!")
					fmt.Println("Text:", text)
					fmt.Println("URI:", mockResult["post_uri"])
				}
				return
			}
			
			// Load configuration
			cfg := config.LoadConfig()

			// Get auth token first to ensure we're authenticated
			_, err := auth.GetToken(cfg)
			if err != nil {
				fmt.Printf("Error: %s\n", formatUserFriendlyError(err, "submit"))
				return
			}

			// Call the service function
			postResult, err := post.SubmitPost(cfg, text)
			if err != nil {
				fmt.Printf("Error: %s\n", formatUserFriendlyError(err, "submit"))
				return
			}

			// Format and display the result
			result := map[string]interface{}{
				"submitted": true,
				"text": text,
				"post_uri": postResult.URI,
				"post_cid": postResult.CID,
			}

			if outputJSON {
				jsonOutput, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					fmt.Println("Error formatting JSON:", err)
					return
				}
				fmt.Println(string(jsonOutput))
			} else {
				fmt.Println("Post submitted successfully!")
				fmt.Println("Text:", text)
				fmt.Println("URI:", postResult.URI)
			}
		},
	}

	// Add flags
	cmd.Flags().StringVar(&text, "text", "", "Text content of the post to submit")
	cmd.Flags().BoolVar(&outputJSON, "json", false, "Output in JSON format")

	// Mark required flags
	cmd.MarkFlagRequired("text")

	return cmd
}

func formatUserFriendlyError(err error, command string) string {
	errMsg := err.Error()

	// Authentication errors
	if strings.Contains(errMsg, "missing Bluesky credentials") || 
	   strings.Contains(errMsg, "authentication failed") {
		return "Authentication failed. Please check your Bluesky credentials are set correctly.\n" +
			"You can set them using environment variables (BSKY_ID, BSKY_PASSWORD) or a config file."
	}

	// Connection/API errors
	if strings.Contains(errMsg, "connection refused") || 
	   strings.Contains(errMsg, "no such host") || 
	   strings.Contains(errMsg, "request failed") {
		return "Could not connect to the Bluesky API. Please check your internet connection and try again."
	}

	// Timeout errors
	if strings.Contains(errMsg, "timeout") || 
	   strings.Contains(errMsg, "deadline exceeded") {
		return "The request timed out. Please try again later."
	}

	// Command-specific errors
	switch command {
	case "feed":
		if strings.Contains(errMsg, "feed analysis failed") {
			return "Feed analysis failed. Please try with a different hashtag or fewer posts."
		}
	case "community":
		if strings.Contains(errMsg, "missing or invalid user handle") {
			return "Invalid user handle. Please enter a correct name (e.g., user.bsky.social)."
		}
		if strings.Contains(errMsg, "invalid user handle format") {
			return "Invalid user handle format. Please use the format username.bsky.social or a valid DID."
		}
	case "assist", "submit":
		if strings.Contains(errMsg, "topic too long") {
			return "Topic is too long. Please keep it under 200 characters."
		}
		if strings.Contains(errMsg, "failed to create post") {
			return "Failed to create post. Please check your account permissions and try again."
		}
	}

	// If we don't have a specific message, return a generic one with the technical error
	return fmt.Sprintf("An error occurred: %s", errMsg)
}