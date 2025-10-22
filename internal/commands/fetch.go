package commands

import (
	"fmt"

	"github.com/lorchard/feed-to-mastodon/internal/config"
	"github.com/lorchard/feed-to-mastodon/internal/database"
	"github.com/lorchard/feed-to-mastodon/internal/feed"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var noPurge bool

// NewFetchCmd creates the fetch command.
func NewFetchCmd() *cobra.Command {
	fetchCmd := &cobra.Command{
		Use:   "fetch",
		Short: "Fetch feed entries and save them to the database",
		Long: `Fetch retrieves entries from the configured RSS/Atom feed and saves
them to the database. Entries that already exist (based on their ID)
are skipped automatically.

By default, entries that are no longer in the feed are purged from the
database to clean up old entries over time.`,
		RunE: runFetch,
	}

	fetchCmd.Flags().BoolVar(&noPurge, "no-purge", false, "skip purging entries that are no longer in the feed")

	return fetchCmd
}

func runFetch(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig(GetConfigFile())
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Open database
	db, err := database.New(cfg.DatabasePath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Get stats before fetch
	totalBefore, _, _, err := db.GetStats()
	if err != nil {
		return fmt.Errorf("failed to get database stats: %w", err)
	}

	// Create fetcher
	fetcher := feed.New()

	// Fetch feed
	logrus.Infof("Fetching feed from %s", cfg.FeedURL)
	feedData, err := fetcher.Fetch(cfg.FeedURL)
	if err != nil {
		return fmt.Errorf("failed to fetch feed: %w", err)
	}

	logrus.Infof("Feed: %s", feedData.Title)
	logrus.Infof("Found %d entries in feed", len(feedData.Items))

	// Save entries to database
	saved, err := fetcher.SaveEntriesToDB(feedData, db)
	if err != nil {
		return fmt.Errorf("failed to save entries: %w", err)
	}

	// Store feed metadata for use in templates
	if err := fetcher.StoreFeedMetadata(feedData, db); err != nil {
		logrus.Warnf("Failed to store feed metadata: %v", err)
	}

	// Purge entries no longer in feed (unless --no-purge is set)
	var purged int
	if !noPurge {
		purged, err = fetcher.PurgeStaleEntries(feedData, db)
		if err != nil {
			logrus.Warnf("Failed to purge stale entries: %v", err)
		}
	}

	// Get stats after fetch
	totalAfter, postedAfter, unpostedAfter, err := db.GetStats()
	if err != nil {
		return fmt.Errorf("failed to get database stats: %w", err)
	}

	// Calculate and display results
	newEntries := totalAfter - totalBefore + purged
	logrus.Infof("Saved %d new entries (skipped %d duplicates)", saved, len(feedData.Items)-saved)
	if purged > 0 {
		logrus.Infof("Purged %d entries no longer in feed", purged)
	}
	logrus.Infof("Database totals: %d total, %d posted, %d unposted",
		totalAfter, postedAfter, unpostedAfter)

	if newEntries > 0 || purged > 0 {
		fmt.Println()
		if newEntries > 0 {
			fmt.Printf("Fetched %d new entries\n", newEntries)
		}
		if purged > 0 {
			fmt.Printf("Purged %d old entries\n", purged)
		}
		if newEntries > 0 {
			fmt.Printf("Run 'feed-to-mastodon status' to see what will be posted\n")
		}
	} else {
		fmt.Println("\nNo new entries found")
	}

	return nil
}
