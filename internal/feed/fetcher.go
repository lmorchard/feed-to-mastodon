package feed

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/lorchard/feed-to-mastodon/internal/database"
	"github.com/mmcdole/gofeed"
	"github.com/sirupsen/logrus"
)

// Fetcher handles fetching and parsing RSS/Atom feeds.
type Fetcher struct {
	parser *gofeed.Parser
}

// New creates a new Fetcher instance.
func New() *Fetcher {
	return &Fetcher{
		parser: gofeed.NewParser(),
	}
}

// Fetch retrieves and parses a feed from the given URL.
func (f *Fetcher) Fetch(feedURL string) (*gofeed.Feed, error) {
	logrus.Infof("Fetching feed: %s", feedURL)

	feed, err := f.parser.ParseURL(feedURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed: %w", err)
	}

	logrus.Infof("Successfully fetched feed: %s (%d items)", feed.Title, len(feed.Items))
	return feed, nil
}

// GenerateEntryID generates a unique ID for a feed entry.
// Uses the item's GUID if available, otherwise creates a SHA256 hash
// of the title, link, and published date.
func GenerateEntryID(item *gofeed.Item) string {
	// Use GUID if available
	if item.GUID != "" {
		return item.GUID
	}

	// Otherwise, hash title + link + published date
	var publishedStr string
	if item.Published != "" {
		publishedStr = item.Published
	} else if item.PublishedParsed != nil {
		publishedStr = item.PublishedParsed.String()
	}

	// Combine fields for hashing
	combined := item.Title + item.Link + publishedStr

	// Generate SHA256 hash
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}

// SaveEntriesToDB saves all feed items to the database.
// Returns the count of entries saved (may be less than total if some are duplicates).
func (f *Fetcher) SaveEntriesToDB(feed *gofeed.Feed, db *database.DB) (int, error) {
	if feed == nil || len(feed.Items) == 0 {
		logrus.Debug("No items to save")
		return 0, nil
	}

	savedCount := 0
	for _, item := range feed.Items {
		// Generate ID
		id := GenerateEntryID(item)

		// Marshal item to JSON
		itemJSON, err := json.Marshal(item)
		if err != nil {
			logrus.Warnf("Failed to marshal item %s: %v", id, err)
			continue
		}

		// Save to database
		err = db.SaveEntry(id, itemJSON)
		if err != nil {
			logrus.Warnf("Failed to save entry %s: %v", id, err)
			continue
		}

		savedCount++
	}

	logrus.Infof("Saved %d/%d entries to database", savedCount, len(feed.Items))
	return savedCount, nil
}

// StoreFeedMetadata stores feed metadata in the database for use in templates.
func (f *Fetcher) StoreFeedMetadata(feed *gofeed.Feed, db *database.DB) error {
	feedJSON, err := json.Marshal(feed)
	if err != nil {
		return fmt.Errorf("failed to marshal feed data: %w", err)
	}

	if err := db.SetSetting("feed_metadata", string(feedJSON)); err != nil {
		return fmt.Errorf("failed to store feed metadata: %w", err)
	}

	logrus.Debug("Stored feed metadata")
	return nil
}

// PurgeStaleEntries removes entries from the database that are no longer in the feed.
// Returns the number of entries purged.
func (f *Fetcher) PurgeStaleEntries(feed *gofeed.Feed, db *database.DB) (int, error) {
	if feed == nil {
		return 0, fmt.Errorf("feed is nil")
	}

	// Collect IDs from current feed
	feedIDs := make(map[string]bool)
	for _, item := range feed.Items {
		id := GenerateEntryID(item)
		feedIDs[id] = true
	}

	// Get all IDs from database
	dbIDs, err := db.GetAllEntryIDs()
	if err != nil {
		return 0, fmt.Errorf("failed to get database entry IDs: %w", err)
	}

	// Find IDs that are in DB but not in feed
	toPurge := make([]string, 0)
	for _, dbID := range dbIDs {
		if !feedIDs[dbID] {
			toPurge = append(toPurge, dbID)
		}
	}

	// Delete entries no longer in feed
	if len(toPurge) == 0 {
		logrus.Debug("No stale entries to purge")
		return 0, nil
	}

	logrus.Infof("Purging %d entries no longer in feed", len(toPurge))
	purged, err := db.DeleteEntries(toPurge)
	if err != nil {
		return purged, fmt.Errorf("error during purge: %w", err)
	}

	logrus.Infof("Purged %d entries", purged)
	return purged, nil
}
