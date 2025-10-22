package commands

import (
	"fmt"

	"github.com/lorchard/feed-to-mastodon/internal/config"
	"github.com/lorchard/feed-to-mastodon/internal/database"
	"github.com/lorchard/feed-to-mastodon/internal/mastodon"
	"github.com/lorchard/feed-to-mastodon/internal/template"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var dryRun bool

// NewPostCmd creates the post command.
func NewPostCmd() *cobra.Command {
	postCmd := &cobra.Command{
		Use:   "post",
		Short: "Post unposted entries to Mastodon",
		Long: `Post retrieves unposted entries from the database and posts them
to Mastodon using the configured template. Entries are marked as posted
after successful posting.

Use --dry-run to preview what would be posted without actually posting.`,
		RunE: runPost,
	}

	postCmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview posts without actually posting to Mastodon")

	return postCmd
}

func runPost(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig(GetConfigFile())
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Open database
	db, err := database.New(cfg.DatabasePath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Get access token from config or database
	accessToken, err := getAccessToken(cfg, db)
	if err != nil {
		return fmt.Errorf("authentication required: %w", err)
	}

	// Validate configuration (but don't require access token since we got it from DB)
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Get unposted entries
	limit := cfg.MaxItems
	entries, err := db.GetUnpostedEntries(limit)
	if err != nil {
		return fmt.Errorf("failed to get unposted entries: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("No unposted entries to post")
		fmt.Println("\nRun 'feed-to-mastodon fetch' to fetch new entries")
		return nil
	}

	logrus.Infof("Found %d unposted entries", len(entries))

	// Create template renderer
	renderer, err := template.New(cfg.TemplateFile, cfg.CharacterLimit)
	if err != nil {
		return fmt.Errorf("failed to create template renderer: %w", err)
	}

	// Create Mastodon poster
	poster, err := mastodon.New(
		cfg.MastodonServer,
		accessToken,
		cfg.PostVisibility,
		cfg.ContentWarning,
	)
	if err != nil {
		return fmt.Errorf("failed to create Mastodon poster: %w", err)
	}

	// Post entries
	if dryRun {
		fmt.Println("DRY RUN: Previewing posts without actually posting")
		fmt.Println()
	}

	posted, err := poster.PostEntries(entries, renderer, dryRun)
	if err != nil {
		return fmt.Errorf("failed to post entries: %w", err)
	}

	// Mark entries as posted if not dry run
	if !dryRun {
		for _, entry := range entries[:posted] {
			if err := db.MarkAsPosted(entry.ID); err != nil {
				logrus.Errorf("Failed to mark entry %s as posted: %v", entry.ID, err)
			}
		}
	}

	// Display summary
	fmt.Printf("\n")
	if dryRun {
		fmt.Printf("DRY RUN: Would have posted %d entries\n", posted)
		fmt.Println("Remove --dry-run to actually post to Mastodon")
	} else {
		fmt.Printf("Successfully posted %d entries to Mastodon\n", posted)
		if posted < len(entries) {
			fmt.Printf("Failed to post %d entries (see logs for details)\n", len(entries)-posted)
		}
	}

	return nil
}
