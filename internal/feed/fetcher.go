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
