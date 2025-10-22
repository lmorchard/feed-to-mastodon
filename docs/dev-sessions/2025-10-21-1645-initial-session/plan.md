# Implementation Plan: feed-to-mastodon

## Overview

This plan breaks down the implementation of the feed-to-mastodon tool into small, iterative steps. Each step builds on the previous one, ensuring that code is always integrated and functional.

## Architecture

```
feed-to-mastodon/
├── cmd/
│   └── feed-to-mastodon/
│       └── main.go              # Entry point
├── internal/
│   ├── config/                  # Configuration management
│   │   ├── config.go
│   │   └── config_test.go       # Unit tests
│   ├── database/                # Database layer
│   │   ├── db.go
│   │   ├── db_test.go           # Unit tests
│   │   └── migrations.go
│   ├── feed/                    # Feed fetching
│   │   ├── fetcher.go
│   │   └── fetcher_test.go      # Unit tests
│   ├── mastodon/                # Mastodon posting
│   │   ├── poster.go
│   │   └── poster_test.go       # Unit tests
│   ├── template/                # Template rendering
│   │   ├── renderer.go
│   │   └── renderer_test.go     # Unit tests
│   └── commands/                # CLI commands (no tests)
│       ├── root.go
│       ├── init.go
│       ├── fetch.go
│       ├── post.go
│       └── status.go
├── go.mod
├── go.sum
├── Makefile
└── .golangci.yml
```

## Implementation Steps

### Phase 1: Project Foundation

#### Step 1: Project Initialization and Basic Structure

**Goal:** Set up the Go module, dependencies, and basic project structure.

**Prompt:**
```
Initialize a new Go project called feed-to-mastodon with the following requirements:

1. Create go.mod with module name github.com/lorchard/feed-to-mastodon
2. Add these dependencies:
   - github.com/mattn/go-sqlite3 v1.14.32
   - github.com/mmcdole/gofeed v1.1.3
   - github.com/sirupsen/logrus v1.8.1
   - github.com/spf13/cobra v1.9.1
   - github.com/spf13/viper v1.20.1
   - github.com/mattn/go-mastodon v0.0.10

3. Create the basic directory structure:
   - cmd/feed-to-mastodon/
   - internal/config/
   - internal/database/
   - internal/feed/
   - internal/mastodon/
   - internal/template/
   - internal/commands/

4. Create a minimal main.go in cmd/feed-to-mastodon/ that just prints "feed-to-mastodon starting..."

5. Ensure the project builds successfully with `go build ./cmd/feed-to-mastodon`

Notes:
- Use Go 1.21 or later
- Include build tags for sqlite3: `// +build !windows`
```

**Validation:**
- Project builds without errors
- All dependencies resolve correctly

---

#### Step 2: Configuration Management Foundation

**Goal:** Implement configuration loading with Viper, supporting both YAML files and CLI flags.

**Prompt:**
```
Implement the configuration management system in internal/config/config.go with the following requirements:

1. Define a Config struct with these fields:
   - FeedURL (string, required)
   - MastodonServer (string, required)
   - MastodonAccessToken (string, required)
   - TemplateFile (string, default: "template.txt")
   - DatabasePath (string, default: "feed-to-mastodon.db")
   - CharacterLimit (int, default: 500)
   - MaxItems (int, default: 0 for unlimited)
   - PostVisibility (string, default: "public")
   - ContentWarning (string, optional)

2. Implement LoadConfig() function that:
   - Uses Viper to read from config.yaml (if present)
   - Sets sensible defaults for all optional fields
   - Returns *Config and error

3. Implement Validate() method on Config that:
   - Ensures required fields (FeedURL, MastodonServer, MastodonAccessToken) are set
   - Validates PostVisibility is one of: public, unlisted, private, direct
   - Returns error if validation fails

4. Add helper function to bind Viper to environment variables with prefix FEED_TO_MASTODON_

Notes:
- CLI flags will be added later when we implement commands
- Keep it simple - just YAML and defaults for now
```

**Validation:**
- Can load config from YAML
- Defaults are applied correctly
- Validation catches missing required fields

---

#### Step 2.5: Configuration Unit Tests

**Goal:** Add comprehensive unit tests for the configuration module.

**Prompt:**
```
Add unit tests for internal/config/config.go in internal/config/config_test.go:

1. Test LoadConfig():
   - Test loading valid YAML config
   - Test with missing config file (should use defaults)
   - Test with partial config (should merge with defaults)
   - Test environment variable binding

2. Test Validate():
   - Test with all required fields present (should pass)
   - Test with missing FeedURL (should fail)
   - Test with missing MastodonServer (should fail)
   - Test with missing MastodonAccessToken (should fail)
   - Test with invalid PostVisibility value (should fail)
   - Test with valid PostVisibility values: public, unlisted, private, direct

3. Test default values:
   - Verify TemplateFile defaults to "template.txt"
   - Verify DatabasePath defaults to "feed-to-mastodon.db"
   - Verify CharacterLimit defaults to 500
   - Verify MaxItems defaults to 0
   - Verify PostVisibility defaults to "public"

Notes:
- Use table-driven tests where appropriate
- Create temporary config files for testing
- Clean up test files in defer statements
```

**Validation:**
- All config tests pass
- Test coverage is comprehensive
- Edge cases are handled

---

### Phase 2: Database Layer

#### Step 3: Database Schema and Migrations

**Goal:** Implement the database layer with SQLite, including schema initialization and migration support.

**Prompt:**
```
Implement the database layer in internal/database/ with the following requirements:

Reference this migration pattern from feedspool-go:
/Users/lorchard/devel/feedspool-go/internal/database/migrations.go

1. In internal/database/db.go:
   - Define DB struct wrapping *sql.DB
   - Implement New(dbPath string) (*DB, error) that:
     - Opens SQLite connection with proper pragmas (foreign_keys=ON, journal_mode=WAL)
     - Calls InitSchema()
     - Calls RunMigrations()
     - Returns *DB
   - Implement Close() error

2. In internal/database/migrations.go:
   - Define getMigrations() map[int]string with migration 1 being the initial schema
   - Implement InitSchema() that creates the initial tables:
     - schema_migrations (version INTEGER PRIMARY KEY, applied_at DATETIME)
     - entries (
         id TEXT PRIMARY KEY,
         entry_data JSON NOT NULL,
         posted_at DATETIME,
         fetched_at DATETIME NOT NULL,
         created_at DATETIME DEFAULT CURRENT_TIMESTAMP
       )
     - Create index on posted_at for efficient queries
   - Implement RunMigrations() that:
     - Checks current version from schema_migrations
     - Applies pending migrations sequentially
     - Records each migration in schema_migrations
     - Uses similar pattern to feedspool-go example
   - Implement ApplyMigration(version int, sql string) error
   - Implement GetMigrationVersion() (int, error)

3. Add logging with logrus for:
   - Database opening
   - Schema initialization
   - Migration application

Notes:
- Migration 1 is the initial schema, handled by InitSchema
- Future migrations go in getMigrations() map starting at version 2
- Use IF NOT EXISTS for safety
- Keep it simple like the feedspool-go example
```

**Validation:**
- Database file is created on first run
- Schema is initialized correctly
- Migration tracking works
- Can open existing database without errors

---

#### Step 4: Database Entry Operations

**Goal:** Implement CRUD operations for feed entries.

**Prompt:**
```
Add entry management methods to internal/database/db.go:

1. Define Entry struct:
   - ID string
   - EntryData []byte (JSON)
   - PostedAt *time.Time (nullable)
   - FetchedAt time.Time
   - CreatedAt time.Time

2. Implement SaveEntry(id string, entryJSON []byte) error:
   - Insert or ignore (to handle duplicates)
   - Set fetched_at to current time
   - Set posted_at to NULL
   - Log at debug level when saving

3. Implement GetUnpostedEntries(limit int) ([]*Entry, error):
   - Select entries where posted_at IS NULL
   - Order by fetched_at ASC (oldest first)
   - Apply limit if > 0
   - Return slice of Entry pointers

4. Implement MarkAsPosted(id string) error:
   - Update posted_at to current timestamp
   - Return error if entry doesn't exist
   - Log at debug level when marking

5. Implement GetStats() (total, posted, unposted int, err error):
   - Count total entries
   - Count where posted_at IS NOT NULL
   - Count where posted_at IS NULL
   - Return all three counts

6. Implement GetLastFetchTime() (*time.Time, error):
   - Return MAX(fetched_at) from entries
   - Return nil if no entries exist

7. Implement GetLastPostTime() (*time.Time, error):
   - Return MAX(posted_at) from entries where posted_at IS NOT NULL
   - Return nil if no posted entries exist

Notes:
- Use prepared statements for safety
- Handle NULL values correctly with *time.Time
- Log errors with context
```

**Validation:**
- Can save entries
- Can retrieve unposted entries in correct order
- Can mark entries as posted
- Stats calculations are accurate

---

#### Step 4.5: Database Unit Tests

**Goal:** Add comprehensive unit tests for the database module.

**Prompt:**
```
Add unit tests for internal/database/ in internal/database/db_test.go:

1. Test database initialization:
   - Test New() creates database file
   - Test schema is created correctly
   - Test migrations table exists
   - Test opening existing database doesn't error

2. Test SaveEntry():
   - Test saving new entry
   - Test saving duplicate entry (should be ignored)
   - Test with valid JSON data
   - Test fetched_at is set automatically
   - Test posted_at starts as NULL

3. Test GetUnpostedEntries():
   - Test returns entries with NULL posted_at
   - Test correct ordering (oldest first)
   - Test limit parameter works (0 for unlimited)
   - Test returns empty slice when no unposted entries

4. Test MarkAsPosted():
   - Test marks entry as posted with timestamp
   - Test error on non-existent entry ID
   - Test posted_at is set to non-NULL

5. Test GetStats():
   - Test with empty database (0, 0, 0)
   - Test counts after adding entries
   - Test counts after posting some entries
   - Test counts are accurate

6. Test GetLastFetchTime() and GetLastPostTime():
   - Test returns nil when no entries exist
   - Test returns correct timestamp
   - Test returns most recent timestamp when multiple entries

7. Test migrations:
   - Test GetMigrationVersion() on new database
   - Test migrations are applied in order
   - Test re-running migrations doesn't error
   - Test migration version is tracked correctly

Notes:
- Use in-memory SQLite database (:memory:) for faster tests
- Use temporary files for tests that need persistent storage
- Clean up test databases in defer statements
- Test concurrent access if relevant
```

**Validation:**
- All database tests pass
- Test coverage includes happy path and error cases
- Migrations are thoroughly tested

---

### Phase 3: Core Business Logic

#### Step 5: Feed Fetching

**Goal:** Implement feed fetching and parsing using gofeed.

**Prompt:**
```
Implement feed fetching in internal/feed/fetcher.go:

1. Define Fetcher struct:
   - parser *gofeed.Parser

2. Implement New() *Fetcher:
   - Initialize gofeed parser
   - Return Fetcher instance

3. Implement Fetch(feedURL string) (*gofeed.Feed, error):
   - Use parser.ParseURL(feedURL)
   - Return feed and any errors
   - Log fetch attempt and results

4. Implement GenerateEntryID(item *gofeed.Item) string:
   - If item.GUID is not empty, return it
   - Otherwise, create SHA256 hash of (Title + Link + Published.String())
   - Return hash as hex string
   - Handle nil Published gracefully

5. Implement SaveEntriesToDB(feed *gofeed.Feed, db *database.DB) (int, error):
   - Iterate through feed.Items
   - For each item:
     - Generate ID using GenerateEntryID
     - Marshal entire item to JSON
     - Save to database with db.SaveEntry()
     - Count saved entries
   - Return count and any error
   - Log how many entries were saved

Notes:
- gofeed provides parsed Item and Feed structs
- We store the entire Item as JSON for maximum flexibility
- Duplicate entries are handled by database (insert or ignore)
```

**Validation:**
- Can fetch real RSS/Atom feeds
- Generates consistent IDs for entries
- Saves entries to database
- Handles feed errors gracefully

---

#### Step 5.5: Feed Fetching Unit Tests

**Goal:** Add unit tests for the feed fetching module.

**Prompt:**
```
Add unit tests for internal/feed/fetcher.go in internal/feed/fetcher_test.go:

1. Test GenerateEntryID():
   - Test with item that has GUID (should return GUID)
   - Test with item without GUID (should return SHA hash)
   - Test hash is consistent for same input
   - Test hash is different for different inputs
   - Test handles nil Published date gracefully
   - Test handles empty Title/Link gracefully

2. Test Fetch():
   - Mock/stub HTTP responses for testing (or use test server)
   - Test fetching valid RSS feed
   - Test fetching valid Atom feed
   - Test handling network errors
   - Test handling invalid feed data
   - Test handling 404 responses
   - Test parsing feed with multiple items

3. Test SaveEntriesToDB():
   - Test with mock database
   - Test counts saved entries correctly
   - Test marshals items to JSON correctly
   - Test generates IDs for all items
   - Test handles empty feed (0 items)
   - Test handles database errors gracefully

Notes:
- Use httptest.Server for testing HTTP requests
- Create sample RSS/Atom feed XML for testing
- Mock database interface if needed
- Test with real gofeed.Item structures
```

**Validation:**
- All feed tests pass
- ID generation is thoroughly tested
- Network error handling is tested

---

#### Step 6: Template Rendering

**Goal:** Implement template rendering with custom functions.

**Prompt:**
```
Implement template rendering in internal/template/renderer.go:

1. Define Renderer struct:
   - tmpl *template.Template
   - characterLimit int

2. Define TemplateData struct:
   - Item (will hold unmarshaled gofeed.Item)
   - Feed (will hold feed-level data if needed)

3. Implement New(templatePath string, characterLimit int) (*Renderer, error):
   - Read template file
   - Parse template with html/template
   - Add custom function "truncate" that takes (string, int) and returns truncated string
   - Store parsed template and characterLimit
   - Return Renderer

4. Implement truncate function:
   - Take string and max length
   - If string is shorter, return as-is
   - Otherwise truncate to max length and append "..."
   - Handle rune boundaries correctly (don't split multi-byte characters)

5. Implement Render(entryJSON []byte) (string, error):
   - Unmarshal entryJSON into gofeed.Item
   - Create TemplateData with the item
   - Execute template into buffer
   - Get rendered string
   - If length exceeds characterLimit, log warning
   - Return rendered string

6. Implement GetDefaultTemplate() string:
   - Return a simple default template:
     "{{.Item.Title}}\n{{.Item.Link}}"

Notes:
- Use html/template for potential HTML support in Mastodon
- The truncate function should be conservative with multi-byte chars
- Don't fail if over character limit, just warn
```

**Validation:**
- Can load and parse templates
- Truncate function works correctly
- Renders feed items properly
- Default template is sensible

---

#### Step 6.5: Template Rendering Unit Tests

**Goal:** Add unit tests for the template rendering module.

**Prompt:**
```
Add unit tests for internal/template/renderer.go in internal/template/renderer_test.go:

1. Test truncate function:
   - Test string shorter than limit (returns as-is)
   - Test string longer than limit (truncates and adds "...")
   - Test with multi-byte characters (UTF-8 safety)
   - Test with emoji characters
   - Test with exactly limit length
   - Test empty string
   - Test zero limit

2. Test New():
   - Test loading valid template file
   - Test template file not found error
   - Test invalid template syntax error
   - Test custom functions are registered (truncate)
   - Test character limit is stored

3. Test Render():
   - Test rendering simple template
   - Test rendering with all item fields populated
   - Test rendering with minimal item fields
   - Test using truncate function in template
   - Test with invalid JSON (should error)
   - Test character limit warning is logged when exceeded
   - Test template execution errors

4. Test GetDefaultTemplate():
   - Test returns non-empty string
   - Test returned template is valid

5. Integration tests:
   - Test complete flow: load template, render entry
   - Test with real gofeed.Item data structure
   - Test complex templates with conditionals

Notes:
- Create temporary template files for testing
- Use sample gofeed.Item JSON for realistic tests
- Verify logged warnings for oversized content
- Clean up temp files in defer
```

**Validation:**
- All template tests pass
- Truncate function handles Unicode correctly
- Template rendering edge cases covered

---

#### Step 7: Mastodon Posting

**Goal:** Implement Mastodon posting functionality.

**Prompt:**
```
Implement Mastodon posting in internal/mastodon/poster.go:

1. Define Poster struct:
   - client *mastodon.Client
   - visibility string
   - contentWarning string

2. Implement New(server, accessToken, visibility, contentWarning string) (*Poster, error):
   - Create mastodon.Client with server and access token
   - Validate visibility is one of: public, unlisted, private, direct
   - Store in Poster struct
   - Return Poster

3. Implement Post(content string, dryRun bool) error:
   - If dryRun is true:
     - Log at info level: "DRY RUN: Would post to Mastodon"
     - Log at debug level: full content
     - Return nil (don't actually post)
   - Otherwise:
     - Create mastodon.Toot with:
       - Status: content
       - Visibility: stored visibility
       - SpoilerText: contentWarning (if set)
     - Call client.PostStatus()
     - Log success or error
     - Return error if posting failed

4. Implement PostEntries(entries []*database.Entry, renderer *template.Renderer, dryRun bool) (posted int, err error):
   - For each entry:
     - Render using renderer.Render(entry.EntryData)
     - Post rendered content (with dryRun flag)
     - If successful and not dryRun, increment counter
     - If error, log but continue to next entry
   - Return count of posted entries

Notes:
- Respect dry run mode - absolutely no API calls when dryRun=true
- Log individual posting errors but don't stop the batch
- Return count of successfully posted entries
```

**Validation:**
- Can connect to Mastodon instance
- Dry run mode doesn't make API calls
- Real posting works
- Handles posting errors gracefully

---

#### Step 7.5: Mastodon Posting Unit Tests

**Goal:** Add unit tests for the Mastodon posting module.

**Prompt:**
```
Add unit tests for internal/mastodon/poster.go in internal/mastodon/poster_test.go:

1. Test New():
   - Test with valid server and token
   - Test with valid visibility values (public, unlisted, private, direct)
   - Test with invalid visibility value (should error)
   - Test content warning is stored correctly
   - Test client is initialized

2. Test Post() with dry run:
   - Test dry run = true doesn't make API calls
   - Test dry run logs appropriate messages
   - Test dry run returns no error
   - Mock the logger to verify log output

3. Test Post() with real posting:
   - Mock Mastodon API client
   - Test successful post
   - Test post with visibility setting
   - Test post with content warning
   - Test API error handling
   - Test network error handling
   - Verify correct Toot structure is created

4. Test PostEntries():
   - Mock renderer and database entries
   - Test posting multiple entries
   - Test with dry run mode
   - Test counter increments correctly
   - Test continues on individual posting errors
   - Test returns correct count of posted entries
   - Test with empty entries list

Notes:
- Mock the Mastodon client interface to avoid real API calls
- Use test server or mocks for HTTP interactions
- Verify logging at appropriate levels
- Test error scenarios thoroughly
```

**Validation:**
- All Mastodon tests pass
- Dry run mode is thoroughly tested
- API interactions are properly mocked
- Error handling is comprehensive

---

### Phase 4: CLI Commands

#### Step 8: Root Command and Logging Setup

**Goal:** Set up Cobra root command with global flags and logging.

**Prompt:**
```
Implement the root command in internal/commands/root.go:

1. Define global variables for flags:
   - configFile (string)
   - verbose (bool)
   - debug (bool)

2. Implement InitRootCmd() *cobra.Command:
   - Create root command with Use: "feed-to-mastodon"
   - Add Short and Long descriptions
   - Add PersistentPreRun that:
     - Sets up logrus based on verbose/debug flags
     - If debug: set logrus.DebugLevel
     - If verbose: set logrus.InfoLevel
     - Otherwise: set logrus.WarnLevel
     - Set logrus formatter to TextFormatter
   - Add persistent flags:
     - --config for config file path
     - --verbose for info level logging
     - --debug for debug level logging
   - Bind flags to Viper

3. Update cmd/feed-to-mastodon/main.go:
   - Import internal/commands
   - Call commands.InitRootCmd()
   - Add subcommands (will be implemented in next steps)
   - Execute root command
   - Exit with proper code on error

Notes:
- Logging setup must happen before any commands run
- Global flags should be available to all subcommands
- Keep main.go minimal
```

**Validation:**
- Can run `feed-to-mastodon --help`
- Logging levels work correctly
- Config file flag is recognized

---

#### Step 9: Init Command

**Goal:** Implement the init command to generate sample config and template files.

**Prompt:**
```
Implement the init command in internal/commands/init.go:

1. Define default config YAML as const string:
```yaml
# Feed to Mastodon Configuration

# RSS/Atom feed URL to monitor (required)
feedUrl: "https://example.com/feed.xml"

# Mastodon server URL (required)
mastodonServer: "https://mastodon.social"

# Mastodon access token (required)
# Get this from your Mastodon instance: Settings > Development > New Application
mastodonAccessToken: "your-access-token-here"

# Path to post template file
templateFile: "template.txt"

# Path to SQLite database
databasePath: "feed-to-mastodon.db"

# Maximum characters for Mastodon posts
characterLimit: 500

# Maximum number of posts per run (0 = unlimited)
maxItems: 10

# Post visibility: public, unlisted, private, or direct
postVisibility: "public"

# Optional content warning for all posts
contentWarning: ""
```

2. Define default template as const string:
```
{{.Item.Title}}
{{.Item.Link}}
```

3. Implement InitInitCmd() *cobra.Command:
   - Create command with Use: "init"
   - Add Short description: "Generate sample configuration and template files"
   - Add --directory/-d flag (default: ".")
   - Implement Run function that:
     - Gets directory from flag
     - Checks if config.yaml exists, error if it does
     - Checks if template.txt exists, error if it does
     - Writes default config to directory/config.yaml
     - Writes default template to directory/template.txt
     - Logs success messages
     - Returns error if writing fails

4. Wire command to root command in main.go

Notes:
- Don't overwrite existing files - fail with clear error
- Create directory if it doesn't exist
- Make it clear where files were written
```

**Validation:**
- Can generate config.yaml and template.txt
- Refuses to overwrite existing files
- Files contain correct default content

---

#### Step 10: Fetch Command

**Goal:** Implement the fetch command to retrieve and store feed entries.

**Prompt:**
```
Implement the fetch command in internal/commands/fetch.go:

1. Implement InitFetchCmd(cfg *config.Config) *cobra.Command:
   - Create command with Use: "fetch"
   - Add Short description: "Fetch RSS/Atom feed and store new entries"
   - Implement Run function that:
     - Opens database using cfg.DatabasePath
     - Defer db.Close()
     - Creates feed.Fetcher
     - Calls fetcher.Fetch(cfg.FeedURL)
     - If fetch fails, log error and exit with code 1
     - Calls fetcher.SaveEntriesToDB(feed, db)
     - Logs how many entries were saved
     - Returns any errors

2. Update root command initialization:
   - Load config using config.LoadConfig()
   - Pass config to InitFetchCmd()
   - Add fetch command to root

3. Handle config file flag:
   - If --config is specified, tell Viper to read that file
   - Otherwise use default config.yaml in current directory if it exists

Notes:
- Exit with non-zero code if feed is unreachable
- Database is automatically created if it doesn't exist
- Duplicate entries are silently ignored (by DB layer)
```

**Validation:**
- Can fetch real RSS/Atom feeds
- Stores entries in database
- Handles network errors appropriately
- Can run multiple times without duplicates

---

#### Step 11: Status Command

**Goal:** Implement the status command to show current state.

**Prompt:**
```
Implement the status command in internal/commands/status.go:

1. Implement InitStatusCmd(cfg *config.Config) *cobra.Command:
   - Create command with Use: "status"
   - Add Short description: "Show current status of feed and posting"
   - Implement Run function that:
     - Opens database
     - Defer db.Close()
     - Calls db.GetStats() for total, posted, unposted counts
     - Calls db.GetLastFetchTime()
     - Calls db.GetLastPostTime()
     - Formats and prints output:
       ```
       Feed URL: [cfg.FeedURL]
       Database: [cfg.DatabasePath]

       Entries:
         Total: [total]
         Posted: [posted]
         Unposted: [unposted]

       Last fetch: [time or "Never"]
       Last post: [time or "Never"]
       ```
     - Use human-friendly time format

2. Wire command to root in main.go

Notes:
- Handle case where database doesn't exist yet (show zeros)
- Format timestamps nicely (e.g., "2 hours ago" or full timestamp)
- Keep output clean and readable
```

**Validation:**
- Shows correct statistics
- Handles empty database gracefully
- Output is clear and useful

---

#### Step 12: Post Command

**Goal:** Implement the post command to publish entries to Mastodon.

**Prompt:**
```
Implement the post command in internal/commands/post.go:

1. Define flags:
   - dryRun (bool)
   - maxItems (int, overrides config)

2. Implement InitPostCmd(cfg *config.Config) *cobra.Command:
   - Create command with Use: "post"
   - Add Short description: "Post unposted entries to Mastodon"
   - Add --dry-run flag
   - Add --max-items flag (overrides config value)
   - Implement Run function that:
     - Opens database
     - Defer db.Close()
     - Determines maxItems (flag overrides config)
     - Gets unposted entries: db.GetUnpostedEntries(maxItems)
     - If no entries, log "No entries to post" and exit
     - Creates template.Renderer from cfg.TemplateFile and cfg.CharacterLimit
     - Creates mastodon.Poster with cfg values
     - For each entry:
       - Render template
       - If verbose/debug: log rendered content
       - Post to Mastodon (respecting dryRun flag)
       - If successful and not dryRun: db.MarkAsPosted(entry.ID)
       - If error: log error but continue
     - Log summary: "Posted X of Y entries" (or "Would post X entries" if dry run)
     - Count successful vs failed posts

3. Wire command to root in main.go

4. Integrate verbose/debug flags:
   - In verbose mode: show count that would be posted
   - In debug mode: show full rendered content and source JSON

Notes:
- Process oldest entries first (handled by DB query)
- Continue on individual posting errors
- In dry-run, show what would be posted but don't mark as posted
- Respect --max-items to control batch size
```

**Validation:**
- Dry run shows what would be posted
- Real posting works
- Marks entries as posted correctly
- Respects max-items limit
- Handles posting failures gracefully

---

### Phase 5: Integration and Polish

#### Step 13: End-to-End Integration

**Goal:** Ensure all commands work together seamlessly.

**Prompt:**
```
Perform end-to-end integration testing and polish:

1. Test complete workflow:
   - Run `init` to generate config
   - Edit config with real feed URL and Mastodon credentials
   - Run `fetch` to get entries
   - Run `status` to verify entries were fetched
   - Run `post --dry-run` to preview
   - Run `post --max-items 1` to post one entry
   - Run `status` again to verify posting

2. Add error handling improvements:
   - Better error messages for common issues (invalid config, network errors, etc.)
   - Graceful handling of missing config file
   - Clear instructions when required fields are missing

3. Add integration tests:
   - Test with sample RSS feed
   - Verify entry ID generation consistency
   - Test template rendering edge cases

4. Update logging:
   - Ensure consistent log levels across all components
   - Add helpful debug logs for troubleshooting
   - Make info-level logs useful but not verbose

Notes:
- Test with real feeds to catch edge cases
- Verify database migrations work on existing databases
- Ensure config validation catches all issues early
```

**Validation:**
- Complete workflow succeeds
- Error messages are helpful
- Logging is appropriate at each level

---

#### Step 14: Documentation and Build Setup

**Goal:** Add documentation and finalize build configuration.

**Prompt:**
```
Add documentation and build setup:

1. Update/create .golangci.yml with basic linting rules:
   - Enable: errcheck, govet, staticcheck, unused
   - Set timeout appropriately
   - Exclude vendor and generated code

2. Verify Makefile has appropriate targets:
   - build: compile the binary
   - test: run tests with `go test ./internal/...`
   - lint: run golangci-lint
   - format: run gofumpt
   - clean: remove build artifacts
   - coverage: generate test coverage report

3. Create README.md with:
   - Project description
   - Installation instructions
   - Quick start guide
   - Configuration reference
   - Template syntax and examples
   - Command reference
   - Example workflows

4. Add example templates:
   - Simple (title + link)
   - Detailed (title, description, link, tags)
   - Custom formatting examples

5. Add .gitignore:
   - Binary outputs
   - Database files
   - Config files with tokens
   - Build artifacts

Notes:
- README should be clear for first-time users
- Include real examples
- Document the template functions (especially truncate)
```

**Validation:**
- Linting passes
- Tests pass
- Build succeeds
- Documentation is clear and complete

---

## Summary of Prompts

This plan provides 19 sequential prompts that:

1. **Phase 1 (Steps 1-2.5):** Set up project foundation, configuration, and tests
2. **Phase 2 (Steps 3-4.5):** Implement database layer with migrations and tests
3. **Phase 3 (Steps 5-7.5):** Build core business logic (feed, template, mastodon) with tests
4. **Phase 4 (Steps 8-12):** Implement all CLI commands (no tests needed per spec)
5. **Phase 5 (Steps 13-14):** Integration and polish

Each step:
- Builds on previous work
- Has clear validation criteria
- Produces working, integrated code
- Is small enough to implement safely
- Is large enough to make meaningful progress
- Includes unit tests for all internal modules (not commands)

## Dependency Graph

```
Step 1 (Project Init)
  └─> Step 2 (Config)
        └─> Step 2.5 (Config Tests)
              └─> Step 3 (DB Schema)
                    └─> Step 4 (DB Operations)
                          └─> Step 4.5 (DB Tests)
                                ├─> Step 5 (Feed Fetching)
                                │     └─> Step 5.5 (Feed Tests)
                                ├─> Step 6 (Templates)
                                │     └─> Step 6.5 (Template Tests)
                                └─> Step 7 (Mastodon)
                                      └─> Step 7.5 (Mastodon Tests)
                                            └─> Step 8 (Root Command)
                                                  ├─> Step 9 (Init Cmd)
                                                  ├─> Step 10 (Fetch Cmd)
                                                  ├─> Step 11 (Status Cmd)
                                                  └─> Step 12 (Post Cmd)
                                                        ├─> Step 13 (Integration)
                                                        └─> Step 14 (Docs)
```

## Notes for Implementation

- Follow Go best practices throughout
- Keep functions small and focused
- Use interfaces where appropriate for testability
- Log errors with context
- Handle edge cases gracefully
- Validate inputs early
- Use prepared statements for SQL
- Close resources properly with defer
- Test incrementally as you build
- Run `go test ./internal/...` after each test step to verify all tests pass
- Use table-driven tests where appropriate
- Mock external dependencies (HTTP, database, Mastodon API) in tests
- Aim for high test coverage in internal modules
- Clean up test resources (files, databases) in defer statements
