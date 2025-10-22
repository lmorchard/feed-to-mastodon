package feed

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lorchard/feed-to-mastodon/internal/database"
	"github.com/mmcdole/gofeed"
)

func TestGenerateEntryID(t *testing.T) {
	t.Run("uses GUID when available", func(t *testing.T) {
		item := &gofeed.Item{
			GUID:  "unique-guid-123",
			Title: "Test Title",
			Link:  "https://example.com",
		}

		id := GenerateEntryID(item)
		if id != "unique-guid-123" {
			t.Errorf("Expected GUID 'unique-guid-123', got %s", id)
		}
	})

	t.Run("generates SHA hash when no GUID", func(t *testing.T) {
		item := &gofeed.Item{
			Title: "Test Title",
			Link:  "https://example.com",
		}

		id := GenerateEntryID(item)
		// Should be a 64-character hex string (SHA256)
		if len(id) != 64 {
			t.Errorf("Expected 64-character hash, got %d characters", len(id))
		}
	})

	t.Run("hash is consistent for same input", func(t *testing.T) {
		item1 := &gofeed.Item{
			Title: "Test Title",
			Link:  "https://example.com",
		}
		item2 := &gofeed.Item{
			Title: "Test Title",
			Link:  "https://example.com",
		}

		id1 := GenerateEntryID(item1)
		id2 := GenerateEntryID(item2)

		if id1 != id2 {
			t.Errorf("Expected same hash for same input, got %s and %s", id1, id2)
		}
	})

	t.Run("hash is different for different inputs", func(t *testing.T) {
		item1 := &gofeed.Item{
			Title: "Test Title 1",
			Link:  "https://example.com",
		}
		item2 := &gofeed.Item{
			Title: "Test Title 2",
			Link:  "https://example.com",
		}

		id1 := GenerateEntryID(item1)
		id2 := GenerateEntryID(item2)

		if id1 == id2 {
			t.Error("Expected different hashes for different inputs")
		}
	})

	t.Run("handles nil Published date gracefully", func(t *testing.T) {
		item := &gofeed.Item{
			Title:           "Test Title",
			Link:            "https://example.com",
			PublishedParsed: nil,
		}

		id := GenerateEntryID(item)
		if id == "" {
			t.Error("Expected non-empty ID")
		}
	})

	t.Run("handles empty Title and Link", func(t *testing.T) {
		item := &gofeed.Item{
			Title: "",
			Link:  "",
		}

		id := GenerateEntryID(item)
		// Should still generate a hash (albeit from empty strings)
		if len(id) != 64 {
			t.Errorf("Expected 64-character hash, got %d characters", len(id))
		}
	})

	t.Run("includes published date in hash", func(t *testing.T) {
		pubTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

		item1 := &gofeed.Item{
			Title:           "Test Title",
			Link:            "https://example.com",
			PublishedParsed: &pubTime,
		}

		pubTime2 := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
		item2 := &gofeed.Item{
			Title:           "Test Title",
			Link:            "https://example.com",
			PublishedParsed: &pubTime2,
		}

		id1 := GenerateEntryID(item1)
		id2 := GenerateEntryID(item2)

		if id1 == id2 {
			t.Error("Expected different hashes for different published dates")
		}
	})
}

func TestFetch(t *testing.T) {
	t.Run("fetches valid RSS feed", func(t *testing.T) {
		// Create test server with valid RSS
		rssContent := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <link>https://example.com</link>
    <description>Test Description</description>
    <item>
      <title>Test Item</title>
      <link>https://example.com/item1</link>
      <description>Test item description</description>
      <guid>item-1</guid>
    </item>
  </channel>
</rss>`

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/rss+xml")
			_, _ = w.Write([]byte(rssContent))
		}))
		defer server.Close()

		fetcher := New()
		feed, err := fetcher.Fetch(server.URL)
		if err != nil {
			t.Fatalf("Fetch() error = %v", err)
		}

		if feed.Title != "Test Feed" {
			t.Errorf("Expected feed title 'Test Feed', got %s", feed.Title)
		}
		if len(feed.Items) != 1 {
			t.Errorf("Expected 1 item, got %d", len(feed.Items))
		}
	})

	t.Run("fetches valid Atom feed", func(t *testing.T) {
		atomContent := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Test Atom Feed</title>
  <link href="https://example.com"/>
  <entry>
    <title>Test Entry</title>
    <link href="https://example.com/entry1"/>
    <id>entry-1</id>
    <summary>Test entry summary</summary>
  </entry>
</feed>`

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/atom+xml")
			_, _ = w.Write([]byte(atomContent))
		}))
		defer server.Close()

		fetcher := New()
		feed, err := fetcher.Fetch(server.URL)
		if err != nil {
			t.Fatalf("Fetch() error = %v", err)
		}

		if feed.Title != "Test Atom Feed" {
			t.Errorf("Expected feed title 'Test Atom Feed', got %s", feed.Title)
		}
		if len(feed.Items) != 1 {
			t.Errorf("Expected 1 entry, got %d", len(feed.Items))
		}
	})

	t.Run("handles network errors", func(t *testing.T) {
		fetcher := New()
		_, err := fetcher.Fetch("http://invalid-url-that-does-not-exist.example.com")
		if err == nil {
			t.Error("Expected error for invalid URL, got nil")
		}
	})

	t.Run("handles invalid feed data", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("This is not valid XML"))
		}))
		defer server.Close()

		fetcher := New()
		_, err := fetcher.Fetch(server.URL)
		if err == nil {
			t.Error("Expected error for invalid feed data, got nil")
		}
	})

	t.Run("handles 404 responses", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		fetcher := New()
		_, err := fetcher.Fetch(server.URL)
		if err == nil {
			t.Error("Expected error for 404 response, got nil")
		}
	})

	t.Run("parses feed with multiple items", func(t *testing.T) {
		rssContent := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Multi-Item Feed</title>
    <item><title>Item 1</title><guid>1</guid></item>
    <item><title>Item 2</title><guid>2</guid></item>
    <item><title>Item 3</title><guid>3</guid></item>
  </channel>
</rss>`

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/rss+xml")
			_, _ = w.Write([]byte(rssContent))
		}))
		defer server.Close()

		fetcher := New()
		feed, err := fetcher.Fetch(server.URL)
		if err != nil {
			t.Fatalf("Fetch() error = %v", err)
		}

		if len(feed.Items) != 3 {
			t.Errorf("Expected 3 items, got %d", len(feed.Items))
		}
	})
}

func TestSaveEntriesToDB(t *testing.T) {
	t.Run("saves entries correctly", func(t *testing.T) {
		db, err := database.New(":memory:")
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer db.Close()

		feed := &gofeed.Feed{
			Title: "Test Feed",
			Items: []*gofeed.Item{
				{GUID: "item-1", Title: "Item 1"},
				{GUID: "item-2", Title: "Item 2"},
			},
		}

		fetcher := New()
		count, err := fetcher.SaveEntriesToDB(feed, db)
		if err != nil {
			t.Fatalf("SaveEntriesToDB() error = %v", err)
		}

		if count != 2 {
			t.Errorf("Expected 2 entries saved, got %d", count)
		}
	})

	t.Run("marshals items to JSON correctly", func(t *testing.T) {
		db, err := database.New(":memory:")
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer db.Close()

		feed := &gofeed.Feed{
			Items: []*gofeed.Item{
				{
					GUID:        "item-1",
					Title:       "Test Item",
					Description: "Test Description",
					Link:        "https://example.com",
				},
			},
		}

		fetcher := New()
		count, err := fetcher.SaveEntriesToDB(feed, db)
		if err != nil {
			t.Fatalf("SaveEntriesToDB() error = %v", err)
		}

		if count != 1 {
			t.Errorf("Expected 1 entry saved, got %d", count)
		}

		// Verify entry was saved with JSON data
		entries, err := db.GetUnpostedEntries(0)
		if err != nil {
			t.Fatalf("GetUnpostedEntries() error = %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("Expected 1 entry in database, got %d", len(entries))
		}
		if len(entries[0].EntryData) == 0 {
			t.Error("Entry data is empty")
		}
	})

	t.Run("generates IDs for all items", func(t *testing.T) {
		db, err := database.New(":memory:")
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer db.Close()

		feed := &gofeed.Feed{
			Items: []*gofeed.Item{
				{Title: "Item 1", Link: "https://example.com/1"},
				{Title: "Item 2", Link: "https://example.com/2"},
			},
		}

		fetcher := New()
		count, err := fetcher.SaveEntriesToDB(feed, db)
		if err != nil {
			t.Fatalf("SaveEntriesToDB() error = %v", err)
		}

		if count != 2 {
			t.Errorf("Expected 2 entries saved, got %d", count)
		}
	})

	t.Run("handles empty feed", func(t *testing.T) {
		db, err := database.New(":memory:")
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer db.Close()

		feed := &gofeed.Feed{
			Items: []*gofeed.Item{},
		}

		fetcher := New()
		count, err := fetcher.SaveEntriesToDB(feed, db)
		if err != nil {
			t.Fatalf("SaveEntriesToDB() error = %v", err)
		}

		if count != 0 {
			t.Errorf("Expected 0 entries saved, got %d", count)
		}
	})

	t.Run("handles nil feed", func(t *testing.T) {
		db, err := database.New(":memory:")
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer db.Close()

		fetcher := New()
		count, err := fetcher.SaveEntriesToDB(nil, db)
		if err != nil {
			t.Fatalf("SaveEntriesToDB() error = %v", err)
		}

		if count != 0 {
			t.Errorf("Expected 0 entries saved for nil feed, got %d", count)
		}
	})
}

func TestStoreFeedMetadata(t *testing.T) {
	t.Run("stores feed metadata successfully", func(t *testing.T) {
		db, err := database.New(":memory:")
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer db.Close()

		feed := &gofeed.Feed{
			Title:       "Test Feed",
			Description: "Test Description",
			Link:        "https://example.com",
		}

		fetcher := New()
		err = fetcher.StoreFeedMetadata(feed, db)
		if err != nil {
			t.Fatalf("StoreFeedMetadata() error = %v", err)
		}

		// Verify metadata was stored
		metadata, err := db.GetSetting("feed_metadata")
		if err != nil {
			t.Fatalf("GetSetting() error = %v", err)
		}
		if metadata == nil {
			t.Fatal("Expected feed metadata to be stored")
		}
		if *metadata == "" {
			t.Error("Feed metadata is empty")
		}
	})

	t.Run("overwrites existing metadata", func(t *testing.T) {
		db, err := database.New(":memory:")
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer db.Close()

		feed1 := &gofeed.Feed{Title: "Feed 1"}
		feed2 := &gofeed.Feed{Title: "Feed 2"}

		fetcher := New()

		// Store first feed
		err = fetcher.StoreFeedMetadata(feed1, db)
		if err != nil {
			t.Fatalf("StoreFeedMetadata() error = %v", err)
		}

		// Store second feed
		err = fetcher.StoreFeedMetadata(feed2, db)
		if err != nil {
			t.Fatalf("StoreFeedMetadata() error = %v", err)
		}

		// Verify latest metadata is stored
		metadata, err := db.GetSetting("feed_metadata")
		if err != nil {
			t.Fatalf("GetSetting() error = %v", err)
		}
		if metadata == nil || *metadata == "" {
			t.Fatal("Expected feed metadata to be stored")
		}

		// Unmarshal and verify it contains the latest feed
		var feed gofeed.Feed
		err = json.Unmarshal([]byte(*metadata), &feed)
		if err != nil {
			t.Fatalf("Failed to unmarshal feed metadata: %v", err)
		}
		if feed.Title != "Feed 2" {
			t.Errorf("Expected feed title 'Feed 2', got '%s'", feed.Title)
		}
	})
}

func TestPurgeStaleEntries(t *testing.T) {
	t.Run("purges entries not in feed", func(t *testing.T) {
		db, err := database.New(":memory:")
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer db.Close()

		fetcher := New()

		// Save initial entries
		initialFeed := &gofeed.Feed{
			Items: []*gofeed.Item{
				{GUID: "item-1", Title: "Item 1"},
				{GUID: "item-2", Title: "Item 2"},
				{GUID: "item-3", Title: "Item 3"},
			},
		}
		_, err = fetcher.SaveEntriesToDB(initialFeed, db)
		if err != nil {
			t.Fatalf("SaveEntriesToDB() error = %v", err)
		}

		// New feed with only 2 items
		newFeed := &gofeed.Feed{
			Items: []*gofeed.Item{
				{GUID: "item-1", Title: "Item 1"},
				{GUID: "item-2", Title: "Item 2"},
			},
		}

		// Purge stale entries
		purged, err := fetcher.PurgeStaleEntries(newFeed, db)
		if err != nil {
			t.Fatalf("PurgeStaleEntries() error = %v", err)
		}

		if purged != 1 {
			t.Errorf("Expected 1 entry purged, got %d", purged)
		}

		// Verify database now has only 2 entries
		total, _, _, err := db.GetStats()
		if err != nil {
			t.Fatalf("GetStats() error = %v", err)
		}
		if total != 2 {
			t.Errorf("Expected 2 entries in database, got %d", total)
		}
	})

	t.Run("purges nothing when all entries still in feed", func(t *testing.T) {
		db, err := database.New(":memory:")
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer db.Close()

		fetcher := New()

		feed := &gofeed.Feed{
			Items: []*gofeed.Item{
				{GUID: "item-1", Title: "Item 1"},
				{GUID: "item-2", Title: "Item 2"},
			},
		}

		// Save entries
		_, err = fetcher.SaveEntriesToDB(feed, db)
		if err != nil {
			t.Fatalf("SaveEntriesToDB() error = %v", err)
		}

		// Purge with same feed
		purged, err := fetcher.PurgeStaleEntries(feed, db)
		if err != nil {
			t.Fatalf("PurgeStaleEntries() error = %v", err)
		}

		if purged != 0 {
			t.Errorf("Expected 0 entries purged, got %d", purged)
		}
	})

	t.Run("purges all entries when feed is empty", func(t *testing.T) {
		db, err := database.New(":memory:")
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer db.Close()

		fetcher := New()

		// Save initial entries
		initialFeed := &gofeed.Feed{
			Items: []*gofeed.Item{
				{GUID: "item-1", Title: "Item 1"},
				{GUID: "item-2", Title: "Item 2"},
			},
		}
		_, err = fetcher.SaveEntriesToDB(initialFeed, db)
		if err != nil {
			t.Fatalf("SaveEntriesToDB() error = %v", err)
		}

		// Empty feed
		emptyFeed := &gofeed.Feed{
			Items: []*gofeed.Item{},
		}

		// Purge stale entries
		purged, err := fetcher.PurgeStaleEntries(emptyFeed, db)
		if err != nil {
			t.Fatalf("PurgeStaleEntries() error = %v", err)
		}

		if purged != 2 {
			t.Errorf("Expected 2 entries purged, got %d", purged)
		}

		// Verify database is empty
		total, _, _, err := db.GetStats()
		if err != nil {
			t.Fatalf("GetStats() error = %v", err)
		}
		if total != 0 {
			t.Errorf("Expected 0 entries in database, got %d", total)
		}
	})

	t.Run("handles nil feed", func(t *testing.T) {
		db, err := database.New(":memory:")
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer db.Close()

		fetcher := New()

		_, err = fetcher.PurgeStaleEntries(nil, db)
		if err == nil {
			t.Error("Expected error for nil feed, got nil")
		}
	})

	t.Run("handles feed with new entries", func(t *testing.T) {
		db, err := database.New(":memory:")
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer db.Close()

		fetcher := New()

		// Save initial entries
		initialFeed := &gofeed.Feed{
			Items: []*gofeed.Item{
				{GUID: "item-1", Title: "Item 1"},
				{GUID: "item-2", Title: "Item 2"},
			},
		}
		_, err = fetcher.SaveEntriesToDB(initialFeed, db)
		if err != nil {
			t.Fatalf("SaveEntriesToDB() error = %v", err)
		}

		// New feed with one old item removed and two new items added
		newFeed := &gofeed.Feed{
			Items: []*gofeed.Item{
				{GUID: "item-1", Title: "Item 1"}, // kept
				{GUID: "item-3", Title: "Item 3"}, // new
				{GUID: "item-4", Title: "Item 4"}, // new
			},
		}

		// Purge stale entries
		purged, err := fetcher.PurgeStaleEntries(newFeed, db)
		if err != nil {
			t.Fatalf("PurgeStaleEntries() error = %v", err)
		}

		if purged != 1 {
			t.Errorf("Expected 1 entry purged (item-2), got %d", purged)
		}
	})

	t.Run("uses GenerateEntryID for items without GUID", func(t *testing.T) {
		db, err := database.New(":memory:")
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer db.Close()

		fetcher := New()

		// Save entries without GUIDs
		initialFeed := &gofeed.Feed{
			Items: []*gofeed.Item{
				{Title: "Item 1", Link: "https://example.com/1"},
				{Title: "Item 2", Link: "https://example.com/2"},
			},
		}
		_, err = fetcher.SaveEntriesToDB(initialFeed, db)
		if err != nil {
			t.Fatalf("SaveEntriesToDB() error = %v", err)
		}

		// New feed with only one item
		newFeed := &gofeed.Feed{
			Items: []*gofeed.Item{
				{Title: "Item 1", Link: "https://example.com/1"},
			},
		}

		// Purge stale entries
		purged, err := fetcher.PurgeStaleEntries(newFeed, db)
		if err != nil {
			t.Fatalf("PurgeStaleEntries() error = %v", err)
		}

		if purged != 1 {
			t.Errorf("Expected 1 entry purged, got %d", purged)
		}
	})
}
