# Feed to Mastodon

A command-line tool that fetches RSS/Atom feeds and posts new entries to Mastodon with customizable templates.

There are many other tools like this, but this one is mine.

## Features

- Fetch RSS and Atom feeds
- Store entries in a local SQLite database
- Post entries to Mastodon with customizable templates
- Dry-run mode for testing
- Automatic duplicate detection
- Configurable post visibility and content warnings
- Character limit validation
- Support for posts-per-run limits

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

# REQUIRED: Mastodon access token
# Create at: Settings > Development > New Application
# Required scopes: write:statuses
mastodon_token: "your-access-token-here"
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

Show database status and preview next entries to be posted.

```bash
feed-to-mastodon status
```

### `post`

Post unposted entries to Mastodon.

```bash
feed-to-mastodon post [--dry-run]
```

Options:
- `--dry-run` - Preview posts without actually posting

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

# REQUIRED: Mastodon access token
mastodon_token: "your-access-token-here"

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
posts_per_run: 0
```

## Template Syntax

Templates use Go's `html/template` syntax. The feed item is available as `.Item`.

### Default Template

```
{{.Item.Title}}
{{.Item.Link}}
```

### Available Fields

- `.Item.Title` - Entry title
- `.Item.Link` - Entry link/URL
- `.Item.Description` - Entry description/summary
- `.Item.Content` - Full entry content
- `.Item.Author.Name` - Author name
- `.Item.Published` - Published date
- `.Item.Updated` - Updated date
- `.Item.GUID` - Unique identifier
- `.Item.Categories` - Entry categories/tags

See [gofeed.Item documentation](https://pkg.go.dev/github.com/mmcdole/gofeed#Item) for all available fields.

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

## Scheduled Posting with Cron

To automatically fetch and post entries, add a cron job:

```bash
# Fetch every hour
0 * * * * cd /path/to/project && /path/to/feed-to-mastodon fetch

# Post every hour (limit to 5 posts)
5 * * * * cd /path/to/project && /path/to/feed-to-mastodon post
```

Or use a single cron job with the `posts_per_run` configuration option.

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
