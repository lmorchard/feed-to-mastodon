package commands

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lorchard/feed-to-mastodon/internal/config"
	"github.com/lorchard/feed-to-mastodon/internal/database"
	mastodon "github.com/mattn/go-mastodon"
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

	// Apply category filters to the unposted backlog, if any filters are set.
	categoryFilter := newCategoryFilter(cfg)
	var postable []*database.Entry
	excludedCount := 0
	if categoryFilter.Enabled() && unposted > 0 {
		unpostedEntries, err := db.GetUnpostedEntries(0)
		if err != nil {
			return fmt.Errorf("failed to get unposted entries: %w", err)
		}

		allowed, excluded := partitionByCategory(unpostedEntries, categoryFilter)
		postable = allowed
		excludedCount = len(excluded)
	}

	// Display overview
	fmt.Println("Feed to Mastodon Status")
	fmt.Println("=======================")
	fmt.Printf("Feed URL: %s\n", cfg.FeedURL)
	fmt.Printf("Database: %s\n", cfg.DatabasePath)

	// Try to show Mastodon account info
	accessToken, err := getAccessToken(cfg, db)
	if err != nil {
		fmt.Printf("Mastodon Account: Not authenticated (%v)\n\n", err)
	} else {
		// Create Mastodon client
		client := mastodon.NewClient(&mastodon.Config{
			Server:      cfg.MastodonServer,
			AccessToken: accessToken,
		})

		// Get account info
		account, err := client.GetAccountCurrentUser(context.Background())
		if err != nil {
			fmt.Printf("Mastodon Account: Authentication error (%v)\n\n", err)
		} else {
			fmt.Printf("Mastodon Account: @%s@%s\n", account.Username, cfg.MastodonServer)
			fmt.Printf("Display Name: %s\n\n", account.DisplayName)
		}
	}

	fmt.Printf("Total entries: %d\n", total)
	fmt.Printf("Posted entries: %d\n", posted)
	if categoryFilter.Enabled() {
		fmt.Printf("Unposted entries: %d\n", unposted)
		fmt.Printf("Excluded by category filters: %d\n\n", excludedCount)
	} else {
		fmt.Printf("Unposted entries: %d\n\n", unposted)
	}

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
	if categoryFilter.Enabled() && unposted > 0 {
		if len(postable) == 0 {
			fmt.Printf("All %d unposted entries are excluded by category filters\n", excludedCount)
			fmt.Println("Run 'feed-to-mastodon prune --excluded' to delete them")
			return nil
		}

		fmt.Println("Next entries to be posted:")
		fmt.Println("--------------------------")
		printPostPreview(postable, 5)

		if len(postable) > 5 {
			fmt.Printf("\n... and %d more\n", len(postable)-5)
		}
	} else if unposted > 0 {
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

// printPostPreview prints up to limit postable entries as a numbered list.
func printPostPreview(entries []*database.Entry, limit int) {
	shown := 0
	for _, entry := range entries {
		if shown >= limit {
			break
		}

		var item gofeed.Item
		if err := json.Unmarshal(entry.EntryData, &item); err != nil {
			logrus.Warnf("Failed to unmarshal entry %s: %v", entry.ID, err)
			continue
		}

		shown++
		fmt.Printf("%d. %s\n", shown, item.Title)
		if item.Link != "" {
			fmt.Printf("   %s\n", item.Link)
		}
	}
}
