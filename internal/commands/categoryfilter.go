package commands

import (
	"encoding/json"

	"github.com/lorchard/feed-to-mastodon/internal/config"
	"github.com/lorchard/feed-to-mastodon/internal/database"
	"github.com/lorchard/feed-to-mastodon/internal/filter"
	"github.com/mmcdole/gofeed"
	"github.com/sirupsen/logrus"
)

// newCategoryFilter builds the category filter from the current configuration.
func newCategoryFilter(cfg *config.Config) *filter.Filter {
	return filter.New(cfg.IncludeCategories, cfg.ExcludeCategories)
}

// partitionByCategory splits unposted entries into those the category filter
// allows and those it excludes. A disabled filter allows every entry without
// decoding it. Entries whose stored JSON cannot be decoded are treated as
// allowed so the poster can report the real error.
func partitionByCategory(entries []*database.Entry, f *filter.Filter) (allowed, excluded []*database.Entry) {
	if !f.Enabled() {
		return entries, nil
	}

	allowed = make([]*database.Entry, 0, len(entries))
	excluded = make([]*database.Entry, 0)

	for _, entry := range entries {
		item := entryItem(entry)
		if item == nil || f.Allowed(item.Categories) {
			allowed = append(allowed, entry)
			continue
		}
		excluded = append(excluded, entry)
	}

	return allowed, excluded
}

// entryItem decodes an entry's stored feed item. It returns nil when the data
// cannot be decoded, logging the reason at debug level.
func entryItem(entry *database.Entry) *gofeed.Item {
	var item gofeed.Item
	if err := json.Unmarshal(entry.EntryData, &item); err != nil {
		logrus.Debugf("Could not decode entry %s for category filtering: %v", entry.ID, err)
		return nil
	}

	return &item
}
