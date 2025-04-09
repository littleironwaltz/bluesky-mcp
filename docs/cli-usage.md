# Bluesky MCP CLI Tool Usage Guide

The Bluesky MCP CLI tool provides easy access to all Bluesky MCP API features directly from the command line without requiring any API knowledge. This guide explains how to use each command with examples.

## Prerequisites

- Go 1.21 or higher
- Bluesky account credentials set up with either:
  - Environment variables: `BSKY_ID`, `BSKY_PASSWORD`, `BSKY_HOST`
  - Configuration file: `BSKY_CONFIG_FILE` pointing to a JSON config
- For testing without credentials, you can use mock mode with: `MOCK_MODE=1`

## Building the CLI

```bash
# Navigate to the project root
cd bluesky-mcp

# Build the CLI tool
make build-cli

# The binary will be available at ./bin/bluesky-mcp-cli
```

## Available Commands

### 1. Generate Post Suggestions

Generate creative post suggestions based on your specified mood and topic.

```bash
./bin/bluesky-mcp-cli assist --mood excited --topic "artificial intelligence"
```

**Options:**
- `--mood` (required): The emotional tone for the post
  - Valid options: `happy`, `sad`, `excited`, `thoughtful`
- `--topic` (required): The subject of the post (max 200 characters)
- `--submit`: Submit the generated post directly to your Bluesky account
- `--json`: Output in JSON format instead of plain text

**Examples:**
```bash
# Generate a happy post about gardening
./bin/bluesky-mcp-cli assist --mood happy --topic gardening

# Generate an excited post about technology
./bin/bluesky-mcp-cli assist --mood excited --topic technology

# Generate and submit a post to your Bluesky account
./bin/bluesky-mcp-cli assist --mood thoughtful --topic philosophy --submit

# Generate a post in JSON format
./bin/bluesky-mcp-cli assist --mood thoughtful --topic philosophy --json
```

### 2. Submit a Post to Bluesky

Submit a post directly to your Bluesky account.

```bash
./bin/bluesky-mcp-cli submit --text "Hello, Bluesky! This is a post from the CLI tool."
```

**Options:**
- `--text` (required): The text content of the post
- `--json`: Output in JSON format instead of plain text

**Examples:**
```bash
# Submit a simple post
./bin/bluesky-mcp-cli submit --text "Just installed the new Bluesky MCP CLI tool!"

# Submit a post with hashtags
./bin/bluesky-mcp-cli submit --text "Loving the new API tools #bluesky #development"

# Submit a post with JSON output
./bin/bluesky-mcp-cli submit --text "Post with JSON response" --json
```

### 3. Analyze Posts with a Hashtag

Analyze posts containing a specific hashtag and display sentiment analysis results.

```bash
./bin/bluesky-mcp-cli feed --hashtag golang --limit 5
```

**Options:**
- `--hashtag` (required): The hashtag to analyze (without the # symbol)
- `--limit` (optional): Number of posts to analyze (default: 10, max: 100)
- `--json`: Output in JSON format instead of human-readable text

**Examples:**
```bash
# Analyze 5 posts with #golang hashtag
./bin/bluesky-mcp-cli feed --hashtag golang --limit 5

# Analyze posts with #art hashtag in JSON format
./bin/bluesky-mcp-cli feed --hashtag art --json

# Run in mock mode for testing without credentials
MOCK_MODE=1 ./bin/bluesky-mcp-cli feed --hashtag golang --limit 5
```

### 4. Monitor User Activity

Display recent posts from a specific Bluesky user.

```bash
./bin/bluesky-mcp-cli community --user user.bsky.social --limit 3
```

**Options:**
- `--user` (required): Username in the format `username.bsky.social` or `did:plc:...`
- `--limit` (optional): Number of posts to display (default: 5, max: 50)
- `--json`: Output in JSON format instead of a numbered list

**Examples:**
```bash
# Get 3 recent posts from a user
./bin/bluesky-mcp-cli community --user user.bsky.social --limit 3

# Get user posts in JSON format
./bin/bluesky-mcp-cli community --user did:plc:abcdefg --json

# Run in mock mode for testing without credentials
MOCK_MODE=1 ./bin/bluesky-mcp-cli community --user user.bsky.social --limit 3
```

### 5. Display Version Information

```bash
./bin/bluesky-mcp-cli version
```

## Authentication

The CLI uses the same authentication method as the server, looking for credentials in:

1. Environment variables:
   ```bash
   export BSKY_ID="your-bluesky-handle-or-email"
   export BSKY_PASSWORD="your-bluesky-password"
   export BSKY_HOST="https://bsky.social"  # Optional, defaults to https://bsky.social
   ```

2. Or a configuration file specified with environment variable:
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

3. For testing without credentials, use mock mode:
   ```bash
   export MOCK_MODE=1
   ./bin/bluesky-mcp-cli <command> [flags]
   ```
   
   Or inline with commands:
   ```bash
   MOCK_MODE=1 ./bin/bluesky-mcp-cli <command> [flags]
   ```

## Error Handling

The CLI provides user-friendly error messages. Common issues include:

- **Authentication Errors**: Check if your Bluesky credentials are set correctly
- **Connection Errors**: Verify your internet connection
- **Invalid Parameters**: Ensure parameters are valid (e.g., correct user handle format)
- **API Errors**: The Bluesky API may be temporarily unavailable

For more detailed information, you can view the JSON response using the `--json` flag.

## Recent Improvements

- Fixed post submission in `assist --submit` command to correctly use authenticated user's DID
- Enhanced authentication mechanism for more reliable post creation
- Improved error handling for post submission failures
- More consistent authentication management across all commands