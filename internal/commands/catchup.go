package commands

import (
	"fmt"

	"github.com/lorchard/feed-to-mastodon/internal/config"
	"github.com/lorchard/feed-to-mastodon/internal/database"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var catchupDryRun bool

// NewCatchupCmd creates the catchup command.
func NewCatchupCmd() *cobra.Command {
	catchupCmd := &cobra.Command{
		Use:   "catchup",
		Short: "Mark all unposted entries as posted without posting them",
		Long: `Catchup marks all unposted entries as posted without actually posting them to Mastodon.

This is useful when you want to skip posting old entries and only post new entries going forward.
For example, after adding a new feed or after a long period of inactivity.

Use --dry-run to preview what would be marked without actually marking.`,
		RunE: runCatchup,
	}

	catchupCmd.Flags().BoolVar(&catchupDryRun, "dry-run", false, "preview entries without actually marking them as posted")

	return catchupCmd
}

func runCatchup(cmd *cobra.Command, args []string) error {
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

	// Get unposted entries
	entries, err := db.GetUnpostedEntries(0) // 0 = all entries
	if err != nil {
		return fmt.Errorf("failed to get unposted entries: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("No unposted entries to mark")
		return nil
	}

	logrus.Infof("Found %d unposted entries", len(entries))

	// Mark entries as posted if not dry run
	if catchupDryRun {
		fmt.Printf("\nDRY RUN: Would mark %d entries as posted\n", len(entries))
		fmt.Println("Remove --dry-run to actually mark entries as posted")
	} else {
		markedCount := 0
		for _, entry := range entries {
			if err := db.MarkAsPosted(entry.ID); err != nil {
				logrus.Errorf("Failed to mark entry %s as posted: %v", entry.ID, err)
			} else {
				markedCount++
			}
		}

		fmt.Printf("\nMarked %d entries as posted\n", markedCount)
		if markedCount < len(entries) {
			fmt.Printf("Failed to mark %d entries (see logs for details)\n", len(entries)-markedCount)
		}
	}

	return nil
}
