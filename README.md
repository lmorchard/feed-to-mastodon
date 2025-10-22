# Feed to Mastodon

A command-line tool that fetches RSS/Atom feeds and posts new entries to Mastodon with customizable templates.

There are many other tools like this, but this one is mine.

## Features

- Fetch RSS and Atom feeds
- Store entries in a local SQLite database
- Post entries to Mastodon with customizable templates
- OAuth authentication flow for Mastodon
- Dry-run mode for testing
- Automatic duplicate detection
- Configurable post visibility and content warnings
- Character limit validation
- Support for posts-per-run limits
- Catchup mode to skip old entries
- Account verification in status command

## Installation

### From Source

```bash
git clone https://github.com/lorchard/feed-to-mastodon.git
cd feed-to-mastodon
go build ./cmd/feed-to-mastodon
```

The binary will be created as `feed-to-mastodon` in the current directory.

## Quick Start

### 1. Initialize a New Project

```bash
./feed-to-mastodon init
```

This creates:
- `feed-to-mastodon.yaml` - Configuration file
- `post-template.txt` - Post template
- `feed-to-mastodon.db` - SQLite database

### 2. Configure

Edit `feed-to-mastodon.yaml`:

```yaml
# REQUIRED: Feed URL to fetch
feed_url: "https://example.com/feed.xml"

# REQUIRED: Mastodon server URL
mastodon_server: "https://mastodon.social"

# AUTHENTICATION: Choose one of two methods:
#
# Method 1 - Direct Access Token (quick but manual):
# Create at: Settings > Development > New Application
# Required scopes: write:statuses
mastodon_token: "your-access-token-here"
#
# Method 2 - OAuth Flow (recommended, see Authentication section below):
# mastodon_client_id: "your-client-id"
# mastodon_client_secret: "your-client-secret"
```

### 3. Fetch Entries

```bash
./feed-to-mastodon fetch
```

### 4. Check Status

```bash
./feed-to-mastodon status
```

### 5. Post to Mastodon

Test with dry-run first:
```bash
./feed-to-mastodon post --dry-run
```

Then post for real:
```bash
./feed-to-mastodon post
```

## Authentication

There are two ways to authenticate with Mastodon:

### Method 1: Direct Access Token

1. Go to your Mastodon instance: Settings > Development > New Application
2. Create an application with `write:statuses` scope
3. Copy the access token
4. Add it to your config:
   ```yaml
   mastodon_token: "your-access-token-here"
   ```

### Method 2: OAuth Flow (Recommended)

1. Go to your Mastodon instance: Settings > Development > New Application
2. Create an application with `read write` scopes
3. Copy the Client ID and Client Secret
4. Add them to your config:
   ```yaml
   mastodon_client_id: "your-client-id"
   mastodon_client_secret: "your-client-secret"
   ```
5. Generate authorization link:
   ```bash
   ./feed-to-mastodon link
   ```
6. Visit the URL, authorize the application, and copy the authorization code
7. Exchange the code for an access token:
   ```bash
   ./feed-to-mastodon code <authorization-code>
   ```

The access token is stored in the database and used automatically for future posts.

You can verify authentication with:
```bash
./feed-to-mastodon status
```

## Commands

### `init`

Initialize a new feed-to-mastodon project.

```bash
feed-to-mastodon init [--directory DIR]
```

Options:
- `-d, --directory` - Directory to initialize (default: current directory)

### `fetch`

Fetch feed entries and save them to the database.

```bash
feed-to-mastodon fetch
```

### `status`

Show database status, authenticated account info, and preview next entries to be posted.

```bash
feed-to-mastodon status
```

### `post`

Post unposted entries to Mastodon.

```bash
feed-to-mastodon post [--dry-run] [--posts N]
```

Options:
- `--dry-run` - Preview posts without actually posting
- `--posts N` - Maximum number of entries to post (0 = all, overrides config `posts_per_run`)

### `catchup`

Mark all unposted entries as posted without actually posting them. Useful for skipping old entries.

```bash
feed-to-mastodon catchup [--dry-run]
```

Options:
- `--dry-run` - Preview entries without actually marking them

### `link`

Generate OAuth authorization link for Mastodon authentication.

```bash
feed-to-mastodon link
```

Requires `mastodon_client_id` in config.

### `code`

Exchange OAuth authorization code for an access token.

```bash
feed-to-mastodon code <authorization-code>
```

Requires `mastodon_client_id` and `mastodon_client_secret` in config.

### Global Flags

- `-c, --config PATH` - Config file path (default: `./feed-to-mastodon.yaml`)
- `-v, --verbose` - Enable verbose output
- `--debug` - Enable debug output

## Configuration Reference

All configuration options in `feed-to-mastodon.yaml`:

```yaml
# REQUIRED: Feed URL to fetch
feed_url: "https://example.com/feed.xml"

# REQUIRED: Mastodon server URL
mastodon_server: "https://mastodon.social"

# AUTHENTICATION: Choose one of two methods:
#
# Method 1 - Direct Access Token:
# mastodon_token: "your-access-token-here"
#
# Method 2 - OAuth Flow (recommended):
# mastodon_client_id: "your-client-id"
# mastodon_client_secret: "your-client-secret"
# (Then use 'link' and 'code' commands to obtain access token)

# OPTIONAL: Database file path (default: ./feed-to-mastodon.db)
database_path: "feed-to-mastodon.db"

# OPTIONAL: Template file path (default: ./post-template.txt)
template_path: "post-template.txt"

# OPTIONAL: Character limit for posts (default: 500)
character_limit: 500

# OPTIONAL: Post visibility (public, unlisted, private, direct)
# Default: public
post_visibility: "public"

# OPTIONAL: Content warning / spoiler text
# Default: none
# content_warning: "Automated post"

# OPTIONAL: Number of entries to post per run (0 = all)
# Default: 0
# Can be overridden with --posts flag
posts_per_run: 0
```

## Template Syntax

Templates use Go's `text/template` syntax. Both the feed item and feed metadata are available in templates.

### Default Template

```
{{.Item.Title}}

{{if .Item.Description}}{{truncate .Item.Description 100}}{{else if .Item.Content}}{{truncate .Item.Content 100}}{{end}}

{{.Item.Link}}

{{range .Item.Categories}}#{{.}} {{end}}

Via {{.Feed.Title}}
```

### Available Fields

#### Item Fields (`.Item`)

- `.Item.Title` - Entry title
- `.Item.Link` - Entry link/URL
- `.Item.Description` - Entry description/summary
- `.Item.Content` - Full entry content
- `.Item.Author.Name` - Author name
- `.Item.Published` - Published date
- `.Item.Updated` - Updated date
- `.Item.GUID` - Unique identifier
- `.Item.Categories` - Entry categories/tags (array of strings)

See [gofeed.Item documentation](https://pkg.go.dev/github.com/mmcdole/gofeed#Item) for all available fields.

#### Feed Fields (`.Feed`)

- `.Feed.Title` - Feed title
- `.Feed.Description` - Feed description
- `.Feed.Link` - Feed link/URL
- `.Feed.Author.Name` - Feed author
- `.Feed.Language` - Feed language
- `.Feed.Copyright` - Copyright information

See [gofeed.Feed documentation](https://pkg.go.dev/github.com/mmcdole/gofeed#Feed) for all available fields.

### Template Functions

#### `truncate`

Truncate a string to a maximum length (UTF-8 aware).

```
{{.Item.Description | truncate 100}}
```

### Example Templates

#### Simple

```
{{.Item.Title}}
{{.Item.Link}}
```

#### With Description

```
{{.Item.Title}}

{{.Item.Description}}

{{.Item.Link}}
```

#### With Author and Truncation

```
{{.Item.Title}}
{{if .Item.Author}}By: {{.Item.Author.Name}}{{end}}

{{.Item.Description | truncate 200}}

🔗 {{.Item.Link}}
```

#### With Categories as Hashtags

```
{{.Item.Title}}

{{.Item.Description | truncate 150}}

{{.Item.Link}}

{{range .Item.Categories}}#{{.}} {{end}}
```

#### With Feed Attribution

```
{{.Item.Title}}

{{if .Item.Description}}{{truncate .Item.Description 100}}{{else if .Item.Content}}{{truncate .Item.Content 100}}{{end}}

{{.Item.Link}}

Via {{.Feed.Title}}
```

## Scheduled Posting with Cron

To automatically fetch and post entries, add a cron job:

```bash
# Fetch every hour
0 * * * * cd /path/to/project && /path/to/feed-to-mastodon fetch

# Post every hour (limit to 5 posts per run)
5 * * * * cd /path/to/project && /path/to/feed-to-mastodon post --posts 5
```

Or use a single cron job with the `posts_per_run` configuration option.

**Tip**: When setting up a new feed, use the `catchup` command to mark existing entries as posted so you only post new entries going forward:

```bash
./feed-to-mastodon catchup
```

## Development

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build ./cmd/feed-to-mastodon
```

## License

This project is licensed under the terms specified in the LICENSE file.

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.
