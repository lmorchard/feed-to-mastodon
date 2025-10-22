# Session Spec: initial-session

## Overview

Let's build a golang-based command line tool implementing a utility that fetches an RSS/Atom feed and posts new entries to a Mastodon account.

## Dependencies

Start with the following dependencies as building blocks:

- github.com/mattn/go-sqlite3 v1.14.32
-	github.com/mmcdole/gofeed v1.1.3
-	github.com/sirupsen/logrus v1.8.1
-	github.com/spf13/cobra v1.9.1
-	github.com/spf13/viper v1.20.1
- github.com/mattn/go-mastodon v0.0.10

## Configuration

The tool should be configurable via a YAML configuration file and/or command line options, with CLI options taking precedence over YAML settings.

### Configuration Scope
- One feed per configuration (keep it simple)
- All configuration options should have sensible defaults

### Configuration Options

The following should be configurable via YAML and/or CLI options:

- **Feed URL**: The RSS/Atom feed to monitor (required)
- **Mastodon Server URL**: The Mastodon instance URL (required)
- **Mastodon Access Token**: Authentication token for the Mastodon API (required)
- **Template File**: Path to the post template file (configurable, with default)
- **Database Location**: Path to the SQLite database file (configurable, with default)
- **Character Limit**: Maximum characters for Mastodon posts (configurable, default 500)
- **Max Items**: Maximum number of posts to submit in one `post` command run
- **Post Visibility**: Mastodon post visibility (public, unlisted, private, direct)
- **Content Warning**: Optional content warning text for all posts

## Database Schema

Store state in a local SQLite database with automatic creation and migration support.

### Entry Tracking Table

Track feed entries with the following fields:

- **Entry ID**: Primary key
  - Use GUID from feed item
  - Fallback: SHA hash of (title + link + pubdate) if GUID is not present
- **Entry Data**: Full feed entry content stored as JSON
  - Captures all data parsed by gofeed
  - Provides flexibility for future template enhancements
- **Posted At**: Timestamp when entry was posted to Mastodon
  - NULL = not yet posted (ready to post)
  - Non-NULL = already posted
- **Fetched At**: Timestamp when entry was fetched from feed

### Migration System

- Roll our own simple migration tracking (embedded in internal module)
- Auto-run migrations on every startup
- Migrations should be infrequent and simple

## Commands

The tool should support the following subcommands:

### `init` Command

Generate sample configuration and template files.

- Write `config.yaml` and `template.txt` to specified directory
- Support `--directory` / `-d` flag (default: current directory `./`)
- Refuse to overwrite existing files (no `--force` flag for safety)

### `fetch` Command

Fetch the RSS/Atom feed and store new entries in the database.

- Parse feed using gofeed
- Store all entry data as JSON
- Mark entries as ready to post (postedAt = NULL)
- If feed is unreachable or invalid: log error and exit with error code

### `post` Command

Post unposted entries from the database to Mastodon.

- Process entries where postedAt is NULL
- Process oldest entries first (to naturally work through backlog on repeated runs)
- Respect `--max-items` limit if specified
- Render each entry using the configured template
- Validate post length against configured character limit
- If posting fails: log error but leave entry ready for retry (don't mark as posted)
- Support `--dry-run` flag to preview without posting

### `status` Command

Show the current status of the feed and posting process.

Display:
- Feed URL being monitored
- Total entries in database
- Number of unposted entries
- Number of posted entries
- Last fetch time
- Last post time

## Templating

Use Go's `html/template` package for post formatting.

### Template Features

- Expose all properties from gofeed's Item and Feed structures
  - Title, Link, Description/Summary, Published date, Author
  - Categories, Tags, Content (full text)
  - All feed-level metadata
- Provide custom `truncate` function to limit text length
  - Example: `{{.Description | truncate 200}}`
- Template validates against configured character limit

### Default Template

Provide a sensible default template (e.g., title + link).

## Logging

Use logrus for logging with configurable levels.

### Log Levels

- Support `--verbose` flag (info level)
  - In dry-run mode: show count of posts that would be made
- Support `--debug` flag (debug level)
  - In dry-run mode: show full post content and source item properties (JSON)

## Development Tools

We have an initial Makefile to help with building and testing the tool - check out the commands there, they're copied from another project and we may need to customize them.

### Linting

Use github.com/golangci/golangci-lint/cmd/golangci-lint@latest with a basic configuration.

### Formatting

Use mvdan.cc/gofumpt@latest

## Future Features

Features to consider for future iterations:

- Media attachments: Upload and attach feed entry enclosures (images, etc.) to Mastodon posts
- Multiple feed support: Monitor multiple feeds with a single tool instance
- Rate limiting: Automatic throttling to respect Mastodon API limits
- Posting frequency controls: Schedule or delay posts
- Retry with backoff: Smart retry logic for failed posts
