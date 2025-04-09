package post

import (
	"encoding/json"
	"fmt"
	"html"
	"math/rand"
	"time"

	"github.com/littleironwaltz/bluesky-mcp/internal/auth"
	"github.com/littleironwaltz/bluesky-mcp/pkg/config"
)

// Initialize random seed
func init() {
	rand.Seed(time.Now().UnixNano())
}

// templateSelector is a function type for selecting templates
type templateSelector func(templates []string) string

// defaultTemplateSelector returns a random template
var defaultTemplateSelector templateSelector = func(templates []string) string {
	return templates[rand.Intn(len(templates))]
}

// For testing, can be replaced with a deterministic selector
var getRandomTemplate = defaultTemplateSelector

// GeneratePost creates post suggestions based on mood and topic
// Can also directly submit posts to Bluesky when submit parameter is true
func GeneratePost(cfg config.Config, params map[string]interface{}) (interface{}, error) {
	// Proper type assertions with validation
	mood, ok := params["mood"].(string)
	if !ok {
		mood = ""
	}

	topic, ok := params["topic"].(string)
	if !ok {
		topic = ""
	}

	// Check if post should be submitted directly
	submitPost, _ := params["submit"].(bool)

	// Validate inputs
	if len(topic) > 200 {
		return nil, fmt.Errorf("topic too long")
	}

	// Sanitize input to prevent XSS
	topic = html.EscapeString(topic)

	// Templates based on mood
	happyTemplates := []string{
		"Today is a great day!",
		"Feeling so positive right now!",
		"Nothing but blue skies today!",
		"So happy I could burst!",
		"What a wonderful day it's turning out to be!",
	}

	sadTemplates := []string{
		"Feeling a bit down today.",
		"Having one of those days...",
		"Sometimes things don't go as planned.",
		"Looking for a silver lining today.",
		"When it rains, it pours.",
	}

	excitedTemplates := []string{
		"I can't contain my excitement!",
		"You won't believe what just happened!",
		"This is absolutely incredible!",
		"I'm literally bouncing with energy!",
		"Big news coming your way!",
	}

	thoughtfulTemplates := []string{
		"I've been pondering something interesting.",
		"Here's a thought worth sharing:",
		"Something to consider today:",
		"Been reflecting on this lately:",
		"Food for thought:",
	}

	// Topic templates
	topicTemplates := []string{
		" I want to talk about %s.",
		" Let's discuss %s today.",
		" Has anyone else been thinking about %s?",
		" What are your thoughts on %s?",
		" %s has been on my mind lately.",
		" Anyone interested in %s?",
		" %s is something we should all explore more.",
		" I've been fascinated by %s recently.",
	}

	// Generic fallback templates
	fallbackTemplates := []string{
		"Let's post something interesting!",
		"What's on everyone's mind today?",
		"How's everyone doing?",
		"Anything exciting happening?",
		"Just wanted to check in!",
		"Happy to connect with you all!",
		"Thoughts?",
		"Open to interesting conversations today!",
	}

	suggestion := ""
	
	// Select mood template
	switch mood {
	case "happy":
		suggestion = getRandomTemplate(happyTemplates)
	case "sad":
		suggestion = getRandomTemplate(sadTemplates)
	case "excited":
		suggestion = getRandomTemplate(excitedTemplates)
	case "thoughtful":
		suggestion = getRandomTemplate(thoughtfulTemplates)
	}

	// Add topic if provided
	if topic != "" {
		if suggestion != "" {
			// If we have a mood, add the topic with a template
			topicFormat := getRandomTemplate(topicTemplates)
			suggestion += fmt.Sprintf(topicFormat, topic)
		} else {
			// If no mood but we have a topic, start with the topic
			topicFormat := getRandomTemplate(topicTemplates)
			suggestion = fmt.Sprintf(topicFormat, topic)
			// Remove leading space if present
			if len(suggestion) > 0 && suggestion[0] == ' ' {
				suggestion = suggestion[1:]
			}
		}
	}

	// Use fallback if no suggestion was generated
	if suggestion == "" {
		suggestion = getRandomTemplate(fallbackTemplates)
	}

	// If submit is true, submit the post to Bluesky
	if submitPost {
		postResult, err := SubmitPost(cfg, suggestion)
		if err != nil {
			return map[string]interface{}{
				"suggestion": suggestion,
				"submitted": false,
				"error": err.Error(),
			}, nil
		}
		return map[string]interface{}{
			"suggestion": suggestion,
			"submitted": true,
			"post_uri": postResult.URI,
			"post_cid": postResult.CID,
		}, nil
	}

	return map[string]string{"suggestion": suggestion}, nil
}

// Result contains information about a successfully created post
type PostResult struct {
	URI string `json:"uri"`
	CID string `json:"cid"`
}

// SubmitPostFunc defines the function signature for the SubmitPost function
type SubmitPostFunc func(cfg config.Config, text string) (*PostResult, error)

// SubmitPost is the actual implementation that submits a post to Bluesky
var SubmitPost SubmitPostFunc = func(cfg config.Config, text string) (*PostResult, error) {
	// Get token manager
	tokenManager := auth.GetTokenManager(cfg)
	
	// Get authenticated client
	client := tokenManager.GetClient()
	
	// Get user DID
	did := tokenManager.GetDID()
	if did == "" {
		// Force authentication if DID is empty
		_, err := tokenManager.GetToken(cfg)
		if err != nil {
			return nil, fmt.Errorf("authentication failed: %w", err)
		}
		did = tokenManager.GetDID()
		if did == "" {
			return nil, fmt.Errorf("unable to get user DID")
		}
	}

	// Create post record
	record := map[string]interface{}{
		"$type": "app.bsky.feed.post",
		"text": text,
		"createdAt": time.Now().UTC().Format(time.RFC3339),
	}

	// Create repo request
	request := map[string]interface{}{
		"repo": did,
		"collection": "app.bsky.feed.post",
		"record": record,
	}

	// Submit post
	responseBody, err := client.Post("com.atproto.repo.createRecord", request)
	if err != nil {
		return nil, fmt.Errorf("failed to create post: %w", err)
	}

	// Parse response
	var result PostResult
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("error parsing create post response: %w", err)
	}

	return &result, nil
}