package post

import (
	"regexp"
	"strings"
	"testing"

	"github.com/littleironwaltz/bluesky-mcp/pkg/config"
)

func TestGeneratePost(t *testing.T) {
	// Set up deterministic template selection for testing
	originalSelector := getRandomTemplate
	getRandomTemplate = func(templates []string) string {
		return templates[0] // Always use first template for predictable tests
	}
	// Restore original function after test
	defer func() {
		getRandomTemplate = originalSelector
	}()
	
	tests := []struct {
		name    string
		cfg     config.Config
		params  map[string]interface{}
		check   func(string) bool
		wantErr bool
	}{
		{
			name: "Happy mood with topic",
			cfg:  config.Config{},
			params: map[string]interface{}{
				"mood":  "happy",
				"topic": "programming",
			},
			check: func(suggestion string) bool {
				// Should start with a happy template and include the topic
				return strings.HasPrefix(suggestion, "Today is a great day!") &&
					strings.Contains(suggestion, "programming")
			},
			wantErr: false,
		},
		{
			name: "Happy mood without topic",
			cfg:  config.Config{},
			params: map[string]interface{}{
				"mood": "happy",
			},
			check: func(suggestion string) bool {
				// Should be just a happy template
				return suggestion == "Today is a great day!"
			},
			wantErr: false,
		},
		{
			name: "Sad mood with topic",
			cfg:  config.Config{},
			params: map[string]interface{}{
				"mood":  "sad",
				"topic": "programming",
			},
			check: func(suggestion string) bool {
				// Should start with a sad template and include the topic
				return strings.HasPrefix(suggestion, "Feeling a bit down today.") &&
					strings.Contains(suggestion, "programming")
			},
			wantErr: false,
		},
		{
			name: "Excited mood with topic",
			cfg:  config.Config{},
			params: map[string]interface{}{
				"mood":  "excited",
				"topic": "programming",
			},
			check: func(suggestion string) bool {
				// Should start with an excited template and include the topic
				return strings.HasPrefix(suggestion, "I can't contain my excitement!") &&
					strings.Contains(suggestion, "programming")
			},
			wantErr: false,
		},
		{
			name: "Thoughtful mood with topic",
			cfg:  config.Config{},
			params: map[string]interface{}{
				"mood":  "thoughtful",
				"topic": "programming",
			},
			check: func(suggestion string) bool {
				// Should start with a thoughtful template and include the topic
				return strings.HasPrefix(suggestion, "I've been pondering something interesting.") &&
					strings.Contains(suggestion, "programming")
			},
			wantErr: false,
		},
		{
			name: "No mood with topic",
			cfg:  config.Config{},
			params: map[string]interface{}{
				"topic": "programming",
			},
			check: func(suggestion string) bool {
				// Should be a topic template with the topic
				return strings.Contains(suggestion, "talk about programming") ||
					strings.Contains(suggestion, "programming")
			},
			wantErr: false,
		},
		{
			name:   "Empty params",
			cfg:    config.Config{},
			params: map[string]interface{}{},
			check: func(suggestion string) bool {
				// Should be a fallback template
				return suggestion == "Let's post something interesting!"
			},
			wantErr: false,
		},
		{
			name: "Non-string mood",
			cfg:  config.Config{},
			params: map[string]interface{}{
				"mood":  123,
				"topic": "programming",
			},
			check: func(suggestion string) bool {
				// Should handle topic without mood
				return strings.Contains(suggestion, "programming")
			},
			wantErr: false,
		},
		{
			name: "Non-string topic",
			cfg:  config.Config{},
			params: map[string]interface{}{
				"mood":  "happy",
				"topic": 123,
			},
			check: func(suggestion string) bool {
				// Should be just a happy template
				return suggestion == "Today is a great day!"
			},
			wantErr: false,
		},
		{
			name: "Topic too long",
			cfg:  config.Config{},
			params: map[string]interface{}{
				"mood":  "happy",
				"topic": string(make([]rune, 201)), // 201 characters
			},
			check:   func(suggestion string) bool { return false },
			wantErr: true,
		},
		{
			name: "Topic with potential XSS",
			cfg:  config.Config{},
			params: map[string]interface{}{
				"mood":  "happy",
				"topic": "<script>alert('xss')</script>",
			},
			check: func(suggestion string) bool {
				// Should sanitize the XSS attempt
				return strings.Contains(suggestion, "Today is a great day!") &&
					strings.Contains(suggestion, "&lt;script&gt;") &&
					!strings.Contains(suggestion, "<script>")
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GeneratePost(tt.cfg, tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("GeneratePost() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if tt.wantErr {
				return
			}
			
			gotMap, ok := got.(map[string]string)
			if !ok {
				t.Errorf("GeneratePost() returned type = %T, want map[string]string", got)
				return
			}
			
			suggestion, exists := gotMap["suggestion"]
			if !exists {
				t.Errorf("GeneratePost() response missing 'suggestion' key")
				return
			}
			
			if !tt.check(suggestion) {
				t.Errorf("GeneratePost() = %v, did not pass validation check", suggestion)
			}
		})
	}
}

func TestRandomTemplates(t *testing.T) {
	// Use default random template selector for this test
	// (it's already using the default, no need to override)
	
	// Test that we get varied outputs
	params := map[string]interface{}{
		"mood":  "happy",
		"topic": "programming",
	}
	
	// Run multiple times to check randomness
	suggestions := make(map[string]bool)
	for i := 0; i < 10; i++ {
		result, err := GeneratePost(config.Config{}, params)
		if err != nil {
			t.Fatalf("GeneratePost() error = %v", err)
		}
		
		gotMap, ok := result.(map[string]string)
		if !ok {
			t.Fatalf("GeneratePost() returned type = %T, want map[string]string", result)
		}
		
		suggestion := gotMap["suggestion"]
		suggestions[suggestion] = true
	}
	
	// We should have at least 2 different suggestions
	// Note: There's a tiny chance this could fail randomly, but it's very unlikely
	if len(suggestions) < 2 {
		t.Errorf("Expected multiple different suggestions, got only %d unique suggestions", len(suggestions))
	}
	
	// Ensure all outputs match expected format for happy mood
	// They should all be happy mood templates with topic templates
	happyPrefix := regexp.MustCompile(`^(Today is a great day|Feeling so positive right now|Nothing but blue skies today|So happy I could burst|What a wonderful day it's turning out to be)`)
	topicPattern := regexp.MustCompile(`programming`)
	
	for suggestion := range suggestions {
		if !happyPrefix.MatchString(suggestion) {
			t.Errorf("Suggestion doesn't start with happy template: %s", suggestion)
		}
		
		if !topicPattern.MatchString(suggestion) {
			t.Errorf("Suggestion doesn't contain topic: %s", suggestion)
		}
	}
}