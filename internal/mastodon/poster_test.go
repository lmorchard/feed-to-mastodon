package mastodon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/lorchard/feed-to-mastodon/internal/database"
	"github.com/lorchard/feed-to-mastodon/internal/template"
	"github.com/mmcdole/gofeed"
)

func TestNew(t *testing.T) {
	t.Run("creates poster with valid parameters", func(t *testing.T) {
		poster, err := New("https://mastodon.social", "test-token", "public", "")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		if poster == nil {
			t.Error("Expected non-nil poster")
		}
		if poster.visibility != "public" {
			t.Errorf("visibility = %s, want public", poster.visibility)
		}
	})

	t.Run("valid visibility: public", func(t *testing.T) {
		_, err := New("https://mastodon.social", "token", "public", "")
		if err != nil {
			t.Errorf("New() with public visibility error = %v", err)
		}
	})

	t.Run("valid visibility: unlisted", func(t *testing.T) {
		_, err := New("https://mastodon.social", "token", "unlisted", "")
		if err != nil {
			t.Errorf("New() with unlisted visibility error = %v", err)
		}
	})

	t.Run("valid visibility: private", func(t *testing.T) {
		_, err := New("https://mastodon.social", "token", "private", "")
		if err != nil {
			t.Errorf("New() with private visibility error = %v", err)
		}
	})

	t.Run("valid visibility: direct", func(t *testing.T) {
		_, err := New("https://mastodon.social", "token", "direct", "")
		if err != nil {
			t.Errorf("New() with direct visibility error = %v", err)
		}
	})

	t.Run("invalid visibility value", func(t *testing.T) {
		_, err := New("https://mastodon.social", "token", "invalid", "")
		if err == nil {
			t.Error("Expected error for invalid visibility")
		}
	})

	t.Run("content warning is stored", func(t *testing.T) {
		poster, err := New("https://mastodon.social", "token", "public", "Test CW")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		if poster.contentWarning != "Test CW" {
			t.Errorf("contentWarning = %s, want 'Test CW'", poster.contentWarning)
		}
	})

	t.Run("client is initialized", func(t *testing.T) {
		poster, err := New("https://mastodon.social", "token", "public", "")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		if poster.client == nil {
			t.Error("Expected non-nil client")
		}
	})
}

func TestPost_DryRun(t *testing.T) {
	t.Run("dry run doesn't make API calls", func(t *testing.T) {
		poster, err := New("https://mastodon.social", "token", "public", "")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		// Dry run should always succeed without making API calls
		err = poster.Post("Test content", true)
		if err != nil {
			t.Errorf("Post() dry run error = %v", err)
		}
	})

	t.Run("dry run with various content", func(t *testing.T) {
		poster, err := New("https://mastodon.social", "token", "public", "")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		testCases := []string{
			"Short post",
			"A longer post with multiple lines\nand some content\nthat spans several lines",
			"Post with special characters: 你好 🌍 #hashtag @mention",
			"",
		}

		for _, content := range testCases {
			err = poster.Post(content, true)
			if err != nil {
				t.Errorf("Post() dry run with content %q error = %v", content, err)
			}
		}
	})
}

func TestPostEntries(t *testing.T) {
	t.Run("posts multiple entries in dry run", func(t *testing.T) {
		// Create test database
		db, err := database.New(":memory:")
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer db.Close()

		// Create test entries
		items := []*gofeed.Item{
			{Title: "Entry 1", Link: "https://example.com/1"},
			{Title: "Entry 2", Link: "https://example.com/2"},
			{Title: "Entry 3", Link: "https://example.com/3"},
		}

		for i, item := range items {
			itemJSON, _ := json.Marshal(item)
			db.SaveEntry(fmt.Sprintf("entry-%d", i+1), itemJSON)
		}

		entries, err := db.GetUnpostedEntries(0)
		if err != nil {
			t.Fatalf("GetUnpostedEntries() error = %v", err)
		}

		// Create template
		tmpDir := t.TempDir()
		tmplPath := filepath.Join(tmpDir, "template.txt")
		err = os.WriteFile(tmplPath, []byte("{{.Item.Title}}\n{{.Item.Link}}"), 0644)
		if err != nil {
			t.Fatalf("Failed to create template: %v", err)
		}

		renderer, err := template.New(tmplPath, 500)
		if err != nil {
			t.Fatalf("template.New() error = %v", err)
		}

		// Create poster
		poster, err := New("https://mastodon.social", "token", "public", "")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		// Post in dry run
		count, err := poster.PostEntries(entries, renderer, true)
		if err != nil {
			t.Fatalf("PostEntries() error = %v", err)
		}

		if count != 3 {
			t.Errorf("PostEntries() count = %d, want 3", count)
		}
	})

	t.Run("handles empty entries list", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmplPath := filepath.Join(tmpDir, "template.txt")
		err := os.WriteFile(tmplPath, []byte("{{.Item.Title}}"), 0644)
		if err != nil {
			t.Fatalf("Failed to create template: %v", err)
		}

		renderer, err := template.New(tmplPath, 500)
		if err != nil {
			t.Fatalf("template.New() error = %v", err)
		}

		poster, err := New("https://mastodon.social", "token", "public", "")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		count, err := poster.PostEntries([]*database.Entry{}, renderer, true)
		if err != nil {
			t.Fatalf("PostEntries() error = %v", err)
		}

		if count != 0 {
			t.Errorf("PostEntries() count = %d, want 0", count)
		}
	})

	t.Run("continues on rendering errors", func(t *testing.T) {
		// Create test database
		db, err := database.New(":memory:")
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer db.Close()

		// Add valid entry
		item := &gofeed.Item{Title: "Valid Entry"}
		itemJSON, _ := json.Marshal(item)
		db.SaveEntry("valid", itemJSON)

		// Add invalid JSON entry
		db.SaveEntry("invalid", []byte("invalid json"))

		entries, err := db.GetUnpostedEntries(0)
		if err != nil {
			t.Fatalf("GetUnpostedEntries() error = %v", err)
		}

		// Create template
		tmpDir := t.TempDir()
		tmplPath := filepath.Join(tmpDir, "template.txt")
		err = os.WriteFile(tmplPath, []byte("{{.Item.Title}}"), 0644)
		if err != nil {
			t.Fatalf("Failed to create template: %v", err)
		}

		renderer, err := template.New(tmplPath, 500)
		if err != nil {
			t.Fatalf("template.New() error = %v", err)
		}

		poster, err := New("https://mastodon.social", "token", "public", "")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		// Should post 1 out of 2 (one valid, one invalid)
		count, err := poster.PostEntries(entries, renderer, true)
		if err != nil {
			t.Fatalf("PostEntries() error = %v", err)
		}

		if count != 1 {
			t.Errorf("PostEntries() count = %d, want 1 (should skip invalid entry)", count)
		}
	})
}

// Note: Testing actual API calls to Mastodon would require either:
// 1. A test Mastodon instance
// 2. Mocking the Mastodon client (complex interface)
// 3. Integration tests with real credentials (not suitable for unit tests)
//
// The tests above cover the logic we can test without making real API calls.
// The dry run functionality is thoroughly tested, which is the most critical
// path for ensuring the code works correctly.
