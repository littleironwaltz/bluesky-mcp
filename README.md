# Bluesky MCP (Model Context Protocol)

A Go service that implements a Model Context Protocol server for the Bluesky social network via the AT Protocol, enabling AI-powered features.

## Features

- **Feed Analysis**: Analyze Bluesky feeds and perform sentiment analysis on posts
- **Post Assistant**: Generate varied and engaging content suggestions for Bluesky posts based on mood and topic
- **Community Management**: Track user activity and monitor recent posts
- **Security**: Implements input validation, TLS security, and protection against common web vulnerabilities
- **High Availability**: Built-in redundancy features for increased reliability

## What is MCP?

The Model Context Protocol (MCP) is an API specification for integrating AI and ML models with the AT Protocol ecosystem. MCP servers provide context to applications, enabling them to leverage AI capabilities without requiring direct integration with large language models or other AI systems.

## Functionality Overview

### Feed Analysis

The Feed Analysis service provides insights into Bluesky posts by:

- Retrieving and analyzing posts from a user's feed or searching for posts with a specific hashtag
- Performing sentiment analysis to classify post tone (positive, negative, or neutral)
- Calculating metrics for each post (character count, word count)
- Implementing caching strategies for improved performance and reliability
- Processing posts in parallel for faster analysis of large datasets

### Post Assist

The Post Assist service helps users create engaging content by:

- Generating varied post suggestions based on user-specified mood and topic
- Offering multiple mood templates (happy, sad, excited, thoughtful) with natural variations
- Integrating topics with different phrasings to maintain diversity
- Randomizing template selection to prevent repetitive suggestions
- Sanitizing inputs to prevent XSS and other injection attacks

### Community Management

The Community Management service helps monitor user activity by:

- Retrieving a user's recent posts from their Bluesky author feed
- Filtering posts to only include those created within the last week
- Implementing caching to reduce API calls and improve response times
- Validating user handles with proper format checking
- Supporting both handle and DID-based user identification

## Example API Requests

### Using curl

#### Feed Analysis Request

```bash
curl -X POST "http://localhost:3000/mcp/feed-analysis" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "feed-analysis",
    "params": {
      "hashtag": "golang",
      "limit": 10
    },
    "id": 1
  }'
```

#### Post Assist Request

```bash
curl -X POST "http://localhost:3000/mcp/post-assist" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "post-assist",
    "params": {
      "mood": "excited",
      "topic": "artificial intelligence"
    },
    "id": 1
  }'
```

#### Community Management Request

```bash
curl -X POST "http://localhost:3000/mcp/community-manage" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "community-manage",
    "params": {
      "userHandle": "user.bsky.social",
      "limit": 5
    },
    "id": 1
  }'
```

## Getting Started

### Prerequisites

- Go 1.21 or higher
- Bluesky account credentials

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/littleironwaltz/bluesky-mcp.git
   cd bluesky-mcp
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Configure the service:
   
   Using environment variables:
   ```bash
   export BSKY_ID="your-bluesky-handle-or-email"
   export BSKY_PASSWORD="your-bluesky-password"
   export BSKY_HOST="https://bsky.social"  # Required for secure operation
   export BSKY_BACKUP_ID="backup-handle-or-email"  # Optional backup credentials
   export BSKY_BACKUP_PASSWORD="backup-password"   # Optional backup credentials
   ```
   
   Or using a JSON configuration file:
   ```bash
   export BSKY_CONFIG_FILE="/path/to/config.json"
   ```
   
   Example config.json:
   ```json
   {
     "BskyID": "your-bluesky-handle-or-email",
     "BskyPassword": "your-bluesky-password",
     "BskyHost": "https://bsky.social"
   }
   ```

### Building and Running the Service

```bash
# Build the server application
make build

# Run the server application
make run

# Or run the binary directly
./bin/bluesky-mcp
```

The service will start on port 3000 by default, with a health check server on port 3001.

## Running with MCP Inspector

Build and run the server with:
```
make build
make run
```

The server runs on port 3000. Connect MCP Inspector to http://localhost:3000/mcp/{method} endpoints.

### Using the CLI Tool

The project includes a command-line interface (CLI) that provides easy access to all API features.

```bash
# Build the CLI tool
make build-cli

# Run CLI commands
./bin/bluesky-mcp-cli assist --mood excited --topic "artificial intelligence"
./bin/bluesky-mcp-cli feed --hashtag golang --limit 5
./bin/bluesky-mcp-cli community --user user.bsky.social --limit 3
./bin/bluesky-mcp-cli version
```

**CLI Commands:**

1. **assist** - Generate post suggestions
   ```
   ./bin/bluesky-mcp-cli assist --mood happy --topic programming
   ```
   Options:
   - `--mood` (required): Mood for the post (happy, sad, excited, thoughtful)
   - `--topic` (required): Topic for the post
   - `--json`: Output in JSON format

2. **feed** - Analyze hashtag feed
   ```
   ./bin/bluesky-mcp-cli feed --hashtag golang --limit 5
   ```
   Options:
   - `--hashtag` (required): Hashtag to analyze
   - `--limit` (optional): Number of posts to analyze (default: 10, max: 100)
   - `--json`: Output in JSON format

3. **community** - Monitor user activity
   ```
   ./bin/bluesky-mcp-cli community --user user.bsky.social --limit 3
   ```
   Options:
   - `--user` (required): Username (format: username.bsky.social)
   - `--limit` (optional): Number of posts to display (default: 5, max: 50)
   - `--json`: Output in JSON format

4. **version** - Display version information
   ```
   ./bin/bluesky-mcp-cli version
   ```

**Mock Mode for Testing:**

For testing without Bluesky credentials, you can use mock mode:
```bash
MOCK_MODE=1 ./bin/bluesky-mcp-cli assist --mood happy --topic programming
```

See `docs/cli-usage.md` for detailed usage instructions.

## API Endpoints

The service exposes a JSON-RPC compatible API at `/mcp/:method` where `:method` can be:

### feed-analysis

Analyze a user's feed or search for posts with a specific hashtag across the network.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "method": "feed-analysis",
  "params": {
    "hashtag": "golang",
    "limit": 10
  },
  "id": 1
}
```

**Parameters:**
- `hashtag` (string, optional): Filter posts by hashtag (uses searchPosts API to find posts across the network)
- `limit` (number, optional, default: 10, max: 100): Maximum number of posts to analyze

**Response:**
```json
{
  "jsonrpc": "2.0",
  "result": {
    "posts": [
      {
        "id": "3kuznviij5k2z",
        "text": "Learning Go is fun! #golang",
        "created_at": "2023-09-15T10:32:17.456Z",
        "author": "user.bsky.social",
        "analysis": {
          "sentiment": "positive"
        },
        "metrics": {
          "length": 25,
          "words": 5
        }
      }
    ],
    "count": 1,
    "source": "api_fresh"
  },
  "id": 1
}
```

### post-assist

Generate post suggestions based on mood and topic.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "method": "post-assist",
  "params": {
    "mood": "happy",
    "topic": "programming"
  },
  "id": 1
}
```

**Parameters:**
- `mood` (string, optional): Mood to influence the post (e.g., "happy", "sad", "excited", "thoughtful")
- `topic` (string, optional, max length: 200): Topic to include in the post

**Response:**
```json
{
  "jsonrpc": "2.0",
  "result": {
    "suggestion": "Feeling so positive right now! Has anyone else been thinking about programming?"
  },
  "id": 1
}
```

The post assistant generates varied suggestions based on the provided mood and topic, with multiple templates for each mood type and different ways to incorporate the topic.

### community-manage

Track user activity and monitor recent posts.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "method": "community-manage",
  "params": {
    "userHandle": "user.bsky.social",
    "limit": 5
  },
  "id": 1
}
```

**Parameters:**
- `userHandle` (string, required): Bluesky handle (format: username.bsky.social or did:plc:...)
- `limit` (number, optional, default: 5, max: 50): Maximum number of posts to return

**Response:**
```json
{
  "jsonrpc": "2.0",
  "result": {
    "user": "user.bsky.social",
    "recentPosts": ["Hello world", "Another post"],
    "count": 2
  },
  "id": 1
}
```

## Health Checking

The service includes a dedicated health check server running on port 3001:

- `/health` or `/healthz` - Returns HTTP 200 with `{"status":"ok"}` when the service is healthy

This can be used by load balancers and monitoring tools to check service status.

## Project Structure

```
📂 bluesky-mcp/
├── 📂 cmd/
│   ├── 📂 bluesky-mcp/        # Main server entry point
│   │   └── 📄 main.go
│   └── 📂 cli/                # CLI application
│       └── 📄 main.go
├── 📂 internal/               # Private application code
│   ├── 📂 auth/               # Authentication with Bluesky API
│   ├── 📂 cache/              # Caching implementation
│   ├── 📂 handlers/           # API handlers
│   ├── 📂 models/             # Data models
│   └── 📂 services/           # Business logic
│       ├── 📂 community/      # Community management 
│       ├── 📂 feed/           # Feed analysis
│       └── 📂 post/           # Post assistance
├── 📂 pkg/                    # Reusable packages
│   ├── 📂 apiclient/          # Bluesky API client
│   └── 📂 config/             # Configuration
├── 📂 api/                    # API specifications
├── 📂 configs/                # Configuration templates
│   └── 📂 fallbacks/          # Fallback responses for API failures
├── 📂 docs/                   # Documentation
│   └── 📄 cli-usage.md        # CLI usage documentation
└── 📄 Makefile                # Build commands
```

## Security Features

- Input validation and sanitization for all API parameters
- Parameter boundary enforcement (minimum/maximum values)
- TLS 1.2+ enforcement for all HTTP communications
- JWT token format validation
- Structured error responses with error codes
- URL parameter encoding to prevent injection attacks
- Content sanitization to prevent XSS attacks
- Method parameter whitelisting for API endpoints
- Secure HTTP headers (Content Security Policy, XSS Protection)
- Rate limiting to prevent abuse
- Consistent authentication mechanism across all services
- Shared authentication client with automatic token refresh

## Reliability Features

- **Circuit Breaker Pattern**: Prevents cascading failures when external services fail
- **Retry Mechanism**: Automatic retries with exponential backoff for transient errors
- **Fallback Responses**: Static fallback data when upstream services are unavailable
- **Stale-While-Revalidate**: Serve stale data while fetching fresh data in the background
- **Backup Credentials**: Support for backup authentication credentials
- **Persistent Cache**: Disk-based cache with automatic recovery after restarts
- **Separate Health Server**: Dedicated health check server on a different port
- **Graceful Degradation**: Returns partial results when possible instead of failing
- **Request Timeouts**: All requests have appropriate timeouts to prevent resource exhaustion
- **Rate Limiting**: Prevents overload from excessive requests
- **Shared Authentication Client**: Consistent authentication across all services
- **Centralized Token Management**: Single token manager for all API requests

## Performance Optimizations

- HTTP connection pooling with optimized settings
- Background token refresh to eliminate authentication delays
- Read-write mutex for concurrent authentication access
- Response caching with TTL and cache statistics
- Memory-efficient data structures with pre-allocated slices
- Improved HTTP client with better error handling
- Graceful server shutdown with context support
- Parallel processing for feed analysis 

## Code Quality Features

- **Type Definitions**: Clear type definitions instead of anonymous structs
- **Function Separation**: Complex operations split into smaller, focused functions
- **Helper Functions**: Reusable helpers to eliminate code duplication
- **Error Handling**: Standardized error handling patterns with user-friendly messages
- **Reduced Nesting**: Avoiding deep nesting for better readability
- **Proper Comments**: Documentation for all public functions and types
- **Consistent Patterns**: Standard approaches for retry operations
- **Clean Architecture**: Clear separation of concerns between components
- **Comprehensive Tests**: Unit tests with high coverage for all key modules (up to 100% in core components)
- **Integration Testing**: Tests verify component interactions work correctly
- **Authentication Management**: Proper authentication token sharing between services
- **Mock Mode**: Testing support with mock data when credentials aren't available
- **CLI Design**: User-friendly command-line interface with clear outputs and options

## Development

### Build and Test

```bash
# Build the server
make build

# Build the CLI
make build-cli

# Build both server and CLI
make build-all

# Run tests
make test

# Run tests with coverage
make test-coverage

# Run tests for a specific package
go test ./internal/services/feed

# Run a specific test function
go test -run TestFunctionName ./internal/services/feed 

# Run benchmarks
go test ./internal -bench=Cache

# Format code
make fmt

# Lint code
make lint

# Vet code
make vet
```

### Test Coverage

The project includes comprehensive test coverage:

- **Models**: 100% coverage
- **Config**: 96.9% coverage
- **Cache**: 87.6% coverage
- **Auth**: 90.8% coverage
- **Handlers**: 60.0% coverage
- **API Client**: 59.4% coverage
- **Services**: 43.6% - 100% coverage

Tests include:
- Unit tests for all key components
- Integration tests for component interaction
- Benchmarks for performance-critical code
- Table-driven tests with multiple scenarios
- CLI tests with mock data support

### Configuration Options

Environment Variables:
- `BSKY_ID` - Your Bluesky handle or email
- `BSKY_PASSWORD` - Your Bluesky password
- `BSKY_HOST` - Bluesky API host (default: https://bsky.social)
- `BSKY_CONFIG_FILE` - Path to a JSON configuration file (overrides environment variables)
- `BSKY_BACKUP_ID` - Backup Bluesky handle or email
- `BSKY_BACKUP_PASSWORD` - Backup Bluesky password
- `MOCK_MODE` - Set to "1" or "true" to enable mock mode for CLI testing without credentials

## License

[MIT License](LICENSE)

## Acknowledgments

- [AT Protocol](https://atproto.com/) - The protocol powering Bluesky
- [Echo Framework](https://echo.labstack.com/) - Web framework for Go