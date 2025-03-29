package models

// Common API error response codes
const (
	ErrInvalidRequest      = "invalid_request"
	ErrInvalidParams       = "invalid_params"
	ErrAuthenticationError = "authentication_error"
	ErrAPIError            = "api_error"
	ErrNotFound            = "not_found"
	ErrInternalError       = "internal_error"
	ErrServiceUnavailable  = "service_unavailable"
	ErrTimeout             = "timeout"
	ErrRateLimited         = "rate_limited"
)

// JSONRPCRequest represents a JSON-RPC request
type JSONRPCRequest struct {
	JSONRPC string                 `json:"jsonrpc"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params"`
	ID      int                    `json:"id"`
}

// JSONRPCResponse represents a JSON-RPC response
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
	ID      int         `json:"id"`
}

// ErrorInfo provides detailed error information
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// NewErrorResponse creates a standardized error response
func NewErrorResponse(id int, code string, message string) JSONRPCResponse {
	return JSONRPCResponse{
		JSONRPC: "2.0",
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
		ID: id,
	}
}

// NewDetailedErrorResponse creates a detailed error response with additional details
func NewDetailedErrorResponse(id int, code string, message string, details string) JSONRPCResponse {
	return JSONRPCResponse{
		JSONRPC: "2.0",
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
			Details: details,
		},
		ID: id,
	}
}

// Post represents a social media post with analysis
type Post struct {
	ID        string            `json:"id,omitempty"`
	Text      string            `json:"text"`
	CreatedAt string            `json:"created_at,omitempty"`
	Author    string            `json:"author,omitempty"`
	Metrics   map[string]int    `json:"metrics,omitempty"`
	Analysis  map[string]string `json:"analysis,omitempty"`
}

// FeedResponse represents a standardized feed analysis response
type FeedResponse struct {
	Posts   []Post `json:"posts"`
	Count   int    `json:"count"`
	Warning string `json:"warning,omitempty"`
	Source  string `json:"source,omitempty"` // Indicates if data is from cache, api, etc.
}