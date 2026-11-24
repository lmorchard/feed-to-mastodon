package commands

import (
	"fmt"

	"github.com/lorchard/feed-to-mastodon/internal/config"
	"github.com/lorchard/feed-to-mastodon/internal/database"
	"github.com/spf13/cobra"
)

const pruneSampleSize = 10

var (
	pruneExcluded bool
	pruneDryRun   bool
)

// NewPruneCmd creates the prune command.
func NewPruneCmd() *cobra.Command {
	pruneCmd := &cobra.Command{
		Use:   "prune --excluded",
		Short: "Delete unposted entries excluded by category filters",
		Long: `Prune deletes unposted entries that match the current include_categories
and exclude_categories filters.

This is the reconciliation step for entries stored before the filters existed.
Those rows stay unposted forever and keep counting toward the unposted total in
status, so they must be pruned explicitly. Prune never deletes anything unless
--excluded is passed.

Use --dry-run to preview what would be deleted without deleting.`,
		RunE: runPrune,
	}

	pruneCmd.Flags().BoolVar(&pruneExcluded, "excluded", false, "delete unposted entries excluded by category filters")
	pruneCmd.Flags().BoolVar(&pruneDryRun, "dry-run", false, "preview entries without deleting them")

	return pruneCmd
}

func runPrune(cmd *cobra.Command, args []string) error {
	if !pruneExcluded {
		return fmt.Errorf("prune requires --excluded: re-run as 'feed-to-mastodon prune --excluded' (add --dry-run to preview)")
	}

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

	categoryFilter := newCategoryFilter(cfg)
	if !categoryFilter.Enabled() {
		return fmt.Errorf("no category filters configured: set include_categories or exclude_categories before pruning")
	}

	entries, err := db.GetUnpostedEntries(0) // 0 = all entries
	if err != nil {
		return fmt.Errorf("failed to get unposted entries: %w", err)
	}

	_, excluded := partitionByCategory(entries, categoryFilter)
	if len(excluded) == 0 {
		fmt.Println("No unposted entries match the category filters")
		return nil
	}

	if pruneDryRun {
		fmt.Printf("DRY RUN: Would delete %d entries excluded by category filters\n\n", len(excluded))
		printEntrySample(excluded, pruneSampleSize)
		fmt.Println("\nRemove --dry-run to delete these entries")
		return nil
	}

	ids := make([]string, len(excluded))
	for i, entry := range excluded {
		ids[i] = entry.ID
	}

	deleted, err := db.DeleteEntries(ids)
	if err != nil {
		return fmt.Errorf("failed to delete entries: %w", err)
	}

	fmt.Printf("Deleted %d entries excluded by category filters\n", deleted)
	if deleted < len(excluded) {
		fmt.Printf("Failed to delete %d entries (see logs for details)\n", len(excluded)-deleted)
	}

	return nil
}

// printEntrySample prints up to limit entries as a numbered title/link list.
func printEntrySample(entries []*database.Entry, limit int) {
	shown := 0
	for _, entry := range entries {
		if shown >= limit {
			break
		}

		item := entryItem(entry)
		if item == nil {
			continue
		}

		shown++
		fmt.Printf("%d. %s\n", shown, item.Title)
		if item.Link != "" {
			fmt.Printf("   %s\n", item.Link)
		}
	}
}
