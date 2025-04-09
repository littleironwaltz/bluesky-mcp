package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// testExecuteCommand is a helper function to execute a cobra command for testing
func testExecuteCommand(root *cobra.Command, args ...string) (output string, err error) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Set up command
	root.SetArgs(args)
	err = root.Execute()

	// Reset stdout and get output
	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	
	return buf.String(), err
}

// setupRootCommand creates a root command for testing
func setupRootCommand() *cobra.Command {
	// Set mock mode for testing
	os.Setenv("MOCK_MODE", "1")
	
	rootCmd := &cobra.Command{
		Use: "bluesky-mcp-cli",
	}
	rootCmd.AddCommand(versionCmd())
	rootCmd.AddCommand(assistCmd(true))
	rootCmd.AddCommand(submitCmd(true))
	rootCmd.AddCommand(feedCmd(true))
	rootCmd.AddCommand(communityCmd(true))
	return rootCmd
}

// TestVersionCommand tests the version command
func TestVersionCommand(t *testing.T) {
	// Save environment variables and restore them after test
	originalBskyID := os.Getenv("BSKY_ID")
	originalBskyPassword := os.Getenv("BSKY_PASSWORD")
	originalMockMode := os.Getenv("MOCK_MODE")
	defer func() {
		os.Setenv("BSKY_ID", originalBskyID)
		os.Setenv("BSKY_PASSWORD", originalBskyPassword)
		os.Setenv("MOCK_MODE", originalMockMode)
	}()

	// Set up test environment
	rootCmd := setupRootCommand()

	// Test version command
	output, err := testExecuteCommand(rootCmd, "version")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	expected := "Bluesky MCP CLI v" + Version + "\n"
	if output != expected {
		t.Errorf("Expected \"%s\", got \"%s\"", expected, output)
	}
}

// TestAssistCommand tests the assist command
func TestAssistCommand(t *testing.T) {
	// Save environment variables and restore them after test
	originalBskyID := os.Getenv("BSKY_ID")
	originalBskyPassword := os.Getenv("BSKY_PASSWORD")
	originalMockMode := os.Getenv("MOCK_MODE")
	defer func() {
		os.Setenv("BSKY_ID", originalBskyID)
		os.Setenv("BSKY_PASSWORD", originalBskyPassword)
		os.Setenv("MOCK_MODE", originalMockMode)
	}()

	// Set up test environment
	rootCmd := setupRootCommand()

	// Test assist command with mock mode
	output, err := testExecuteCommand(rootCmd, "assist", "--mood", "happy", "--topic", "testing")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	// Check that output contains the expected content
	expected := "Feeling happy about testing! This is a mock suggestion."
	if output != expected+"\n" {
		t.Errorf("Expected output to contain: %s, got: %s", expected, output)
	}
	
	// Test assist command with JSON output
	output, err = testExecuteCommand(rootCmd, "assist", "--mood", "excited", "--topic", "AI", "--json")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	// Check that the output contains JSON
	if output == "" || output[0] != '{' {
		t.Errorf("Expected JSON output, got: %s", output)
	}
	
	// Test assist command with submit flag and JSON output to check structure
	output, err = testExecuteCommand(rootCmd, "assist", "--mood", "happy", "--topic", "testing", "--submit", "--json")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	// Check that output contains the expected content for post submission
	if !strings.Contains(output, "\"submitted\": true") {
		t.Errorf("Expected output to contain 'submitted' field, got: %s", output)
	}
}

// TestSubmitCommand tests the submit command
func TestSubmitCommand(t *testing.T) {
	// Save environment variables and restore them after test
	originalBskyID := os.Getenv("BSKY_ID")
	originalBskyPassword := os.Getenv("BSKY_PASSWORD")
	originalMockMode := os.Getenv("MOCK_MODE")
	defer func() {
		os.Setenv("BSKY_ID", originalBskyID)
		os.Setenv("BSKY_PASSWORD", originalBskyPassword)
		os.Setenv("MOCK_MODE", originalMockMode)
	}()

	// Set up test environment
	rootCmd := setupRootCommand()

	// Test submit command with mock mode
	output, err := testExecuteCommand(rootCmd, "submit", "--text", "This is a test post from CLI")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	// Check that output contains the expected content
	if !strings.Contains(output, "Post submitted successfully!") {
		t.Errorf("Expected output to contain 'Post submitted successfully!', got: %s", output)
	}
	
	// Test submit command with JSON output
	output, err = testExecuteCommand(rootCmd, "submit", "--text", "This is a test post with JSON output", "--json")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	// Check that the output contains JSON
	if output == "" || output[0] != '{' {
		t.Errorf("Expected JSON output, got: %s", output)
	}
}

// TestFeedCommand tests the feed command
func TestFeedCommand(t *testing.T) {
	// Save environment variables and restore them after test
	originalBskyID := os.Getenv("BSKY_ID")
	originalBskyPassword := os.Getenv("BSKY_PASSWORD")
	originalMockMode := os.Getenv("MOCK_MODE")
	defer func() {
		os.Setenv("BSKY_ID", originalBskyID)
		os.Setenv("BSKY_PASSWORD", originalBskyPassword)
		os.Setenv("MOCK_MODE", originalMockMode)
	}()

	// Set up test environment
	rootCmd := setupRootCommand()

	// Test feed command with mock mode
	output, err := testExecuteCommand(rootCmd, "feed", "--hashtag", "golang", "--limit", "1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	// Check that output contains the expected content
	if !bytes.Contains([]byte(output), []byte("Posts with hashtag")) {
		t.Errorf("Expected output to contain 'Posts with hashtag', got: %s", output)
	}
	
	// Test feed command with JSON output
	output, err = testExecuteCommand(rootCmd, "feed", "--hashtag", "golang", "--json")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	// Check that the output contains JSON
	if output == "" || output[0] != '{' {
		t.Errorf("Expected JSON output, got: %s", output)
	}
}

// TestCommunityCommand tests the community command
func TestCommunityCommand(t *testing.T) {
	// Save environment variables and restore them after test
	originalBskyID := os.Getenv("BSKY_ID")
	originalBskyPassword := os.Getenv("BSKY_PASSWORD")
	originalMockMode := os.Getenv("MOCK_MODE")
	defer func() {
		os.Setenv("BSKY_ID", originalBskyID)
		os.Setenv("BSKY_PASSWORD", originalBskyPassword)
		os.Setenv("MOCK_MODE", originalMockMode)
	}()

	// Set up test environment
	rootCmd := setupRootCommand()

	// Test community command with mock mode
	output, err := testExecuteCommand(rootCmd, "community", "--user", "test.user", "--limit", "2")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	// Check that output contains the expected content
	if !bytes.Contains([]byte(output), []byte("Recent posts by test.user")) {
		t.Errorf("Expected output to contain 'Recent posts by test.user', got: %s", output)
	}
	
	// Test community command with JSON output
	output, err = testExecuteCommand(rootCmd, "community", "--user", "test.user", "--json")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	// Check that the output contains JSON
	if output == "" || output[0] != '{' {
		t.Errorf("Expected JSON output, got: %s", output)
	}
}

// TestFormatUserFriendlyError tests the error formatting function
func TestFormatUserFriendlyError(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		command  string
		expected string
	}{
		{
			name:     "Authentication error",
			err:      fakeError("missing Bluesky credentials"),
			command:  "feed",
			expected: "Authentication failed. Please check your Bluesky credentials are set correctly.\nYou can set them using environment variables (BSKY_ID, BSKY_PASSWORD) or a config file.",
		},
		{
			name:     "Connection error",
			err:      fakeError("connection refused"),
			command:  "feed",
			expected: "Could not connect to the Bluesky API. Please check your internet connection and try again.",
		},
		{
			name:     "Timeout error",
			err:      fakeError("timeout"),
			command:  "feed",
			expected: "The request timed out. Please try again later.",
		},
		{
			name:     "Feed analysis error",
			err:      fakeError("feed analysis failed"),
			command:  "feed",
			expected: "Feed analysis failed. Please try with a different hashtag or fewer posts.",
		},
		{
			name:     "Invalid user handle",
			err:      fakeError("invalid user handle format"),
			command:  "community",
			expected: "Invalid user handle format. Please use the format username.bsky.social or a valid DID.",
		},
		{
			name:     "Topic too long",
			err:      fakeError("topic too long"),
			command:  "assist",
			expected: "Topic is too long. Please keep it under 200 characters.",
		},
		{
			name:     "Generic error",
			err:      fakeError("something unexpected happened"),
			command:  "assist",
			expected: "An error occurred: something unexpected happened",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := formatUserFriendlyError(tc.err, tc.command)
			if result != tc.expected {
				t.Errorf("Expected \"%s\", got \"%s\"", tc.expected, result)
			}
		})
	}
}

// fakeError implements the error interface for testing
type fakeError string

func (e fakeError) Error() string {
	return string(e)
}