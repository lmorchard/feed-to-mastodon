package commands

import (
	"encoding/json"
	"fmt"

	"github.com/lorchard/feed-to-mastodon/internal/config"
	"github.com/lorchard/feed-to-mastodon/internal/database"
	"github.com/mmcdole/gofeed"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// NewStatusCmd creates the status command.
func NewStatusCmd() *cobra.Command {
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show status of the feed-to-mastodon database",
		Long: `Status displays information about the current state of the database:
- Total entries, posted entries, and unposted entries
- Last fetch time and last post time
- Preview of the next entries that will be posted`,
		RunE: runStatus,
	}

	return statusCmd
}

func runStatus(cmd *cobra.Command, args []string) error {
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

	// Get database statistics
	total, posted, unposted, err := db.GetStats()
	if err != nil {
		return fmt.Errorf("failed to get database stats: %w", err)
	}

	// Get last fetch and post times
	lastFetch, err := db.GetLastFetchTime()
	if err != nil {
		return fmt.Errorf("failed to get last fetch time: %w", err)
	}

	lastPost, err := db.GetLastPostTime()
	if err != nil {
		return fmt.Errorf("failed to get last post time: %w", err)
	}

	// Display overview
	fmt.Println("Feed to Mastodon Status")
	fmt.Println("=======================")
	fmt.Printf("Feed URL: %s\n", cfg.FeedURL)
	fmt.Printf("Database: %s\n\n", cfg.DatabasePath)

	fmt.Printf("Total entries: %d\n", total)
	fmt.Printf("Posted entries: %d\n", posted)
	fmt.Printf("Unposted entries: %d\n\n", unposted)

	if lastFetch != nil {
		fmt.Printf("Last fetch: %s\n", *lastFetch)
	} else {
		fmt.Println("Last fetch: never")
	}

	if lastPost != nil {
		fmt.Printf("Last post: %s\n\n", *lastPost)
	} else {
		fmt.Println("Last post: never")
		fmt.Println()
	}

	// Show preview of next entries to be posted
	if unposted > 0 {
		fmt.Println("Next entries to be posted:")
		fmt.Println("--------------------------")

		entries, err := db.GetUnpostedEntries(5)
		if err != nil {
			return fmt.Errorf("failed to get unposted entries: %w", err)
		}

		for i, entry := range entries {
			var item gofeed.Item
			if err := json.Unmarshal(entry.EntryData, &item); err != nil {
				logrus.Warnf("Failed to unmarshal entry %s: %v", entry.ID, err)
				continue
			}

			fmt.Printf("%d. %s\n", i+1, item.Title)
			if item.Link != "" {
				fmt.Printf("   %s\n", item.Link)
			}
		}

		if unposted > 5 {
			fmt.Printf("\n... and %d more\n", unposted-5)
		}
	} else {
		fmt.Println("No unposted entries")
		if total == 0 {
			fmt.Println("\nRun 'feed-to-mastodon fetch' to fetch entries from the feed")
		}
	}

	return nil
}
