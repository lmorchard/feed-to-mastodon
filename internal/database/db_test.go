package database

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDatabaseInitialization(t *testing.T) {
	t.Run("creates database file", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		db, err := New(dbPath)
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		defer db.Close()

		// Check file exists
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			t.Error("Database file was not created")
		}
	})

	t.Run("schema is created correctly", func(t *testing.T) {
		db, err := New(":memory:")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		defer db.Close()

		// Check entries table exists
		var tableName string
		err = db.conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='entries'").Scan(&tableName)
		if err != nil {
			t.Errorf("entries table not created: %v", err)
		}
		if tableName != "entries" {
			t.Errorf("table name = %v, want entries", tableName)
		}
	})

	t.Run("migrations table exists", func(t *testing.T) {
		db, err := New(":memory:")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		defer db.Close()

		var tableName string
		err = db.conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='schema_migrations'").Scan(&tableName)
		if err != nil {
			t.Errorf("schema_migrations table not created: %v", err)
		}
	})

	t.Run("opening existing database doesn't error", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		// Create database
		db1, err := New(dbPath)
		if err != nil {
			t.Fatalf("First New() error = %v", err)
		}
		db1.Close()

		// Open again
		db2, err := New(dbPath)
		if err != nil {
			t.Fatalf("Second New() error = %v", err)
		}
		defer db2.Close()
	})
}

func TestSaveEntry(t *testing.T) {
	t.Run("saves new entry", func(t *testing.T) {
		db, err := New(":memory:")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		defer db.Close()

		entryData := []byte(`{"title": "Test Entry"}`)
		err = db.SaveEntry("test-id-1", entryData)
		if err != nil {
			t.Errorf("SaveEntry() error = %v", err)
		}

		// Verify entry exists
		var id string
		err = db.conn.QueryRow("SELECT id FROM entries WHERE id = ?", "test-id-1").Scan(&id)
		if err != nil {
			t.Errorf("Entry not found in database: %v", err)
		}
	})

	t.Run("saves duplicate entry is ignored", func(t *testing.T) {
		db, err := New(":memory:")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		defer db.Close()

		entryData := []byte(`{"title": "Test Entry"}`)

		// Save once
		err = db.SaveEntry("test-id-1", entryData)
		if err != nil {
			t.Errorf("First SaveEntry() error = %v", err)
		}

		// Save again (should be ignored)
		err = db.SaveEntry("test-id-1", entryData)
		if err != nil {
			t.Errorf("Second SaveEntry() error = %v", err)
		}

		// Verify only one entry exists
		var count int
		err = db.conn.QueryRow("SELECT COUNT(*) FROM entries WHERE id = ?", "test-id-1").Scan(&count)
		if err != nil {
			t.Fatalf("Query error: %v", err)
		}
		if count != 1 {
			t.Errorf("Expected 1 entry, got %d", count)
		}
	})

	t.Run("fetched_at is set automatically", func(t *testing.T) {
		db, err := New(":memory:")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		defer db.Close()

		entryData := []byte(`{"title": "Test Entry"}`)
		err = db.SaveEntry("test-id-1", entryData)
		if err != nil {
			t.Errorf("SaveEntry() error = %v", err)
		}

		var fetchedAt string
		err = db.conn.QueryRow("SELECT fetched_at FROM entries WHERE id = ?", "test-id-1").Scan(&fetchedAt)
		if err != nil {
			t.Errorf("Query error: %v", err)
		}
		if fetchedAt == "" {
			t.Error("fetched_at is empty")
		}
	})

	t.Run("posted_at starts as NULL", func(t *testing.T) {
		db, err := New(":memory:")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		defer db.Close()

		entryData := []byte(`{"title": "Test Entry"}`)
		err = db.SaveEntry("test-id-1", entryData)
		if err != nil {
			t.Errorf("SaveEntry() error = %v", err)
		}

		var postedAt *string
		err = db.conn.QueryRow("SELECT posted_at FROM entries WHERE id = ?", "test-id-1").Scan(&postedAt)
		if err != nil {
			t.Errorf("Query error: %v", err)
		}
		if postedAt != nil {
			t.Errorf("posted_at should be NULL, got %v", *postedAt)
		}
	})
}

func TestGetUnpostedEntries(t *testing.T) {
	t.Run("returns entries with NULL posted_at", func(t *testing.T) {
		db, err := New(":memory:")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		defer db.Close()

		// Add unposted entry
		if err := db.SaveEntry("unposted-1", []byte(`{"title": "Unposted"}`)); err != nil {
			t.Fatalf("SaveEntry() error = %v", err)
		}

		// Add posted entry
		if err := db.SaveEntry("posted-1", []byte(`{"title": "Posted"}`)); err != nil {
			t.Fatalf("SaveEntry() error = %v", err)
		}
		if err := db.MarkAsPosted("posted-1"); err != nil {
			t.Fatalf("MarkAsPosted() error = %v", err)
		}

		// Get unposted
		entries, err := db.GetUnpostedEntries(0)
		if err != nil {
			t.Fatalf("GetUnpostedEntries() error = %v", err)
		}

		if len(entries) != 1 {
			t.Errorf("Expected 1 unposted entry, got %d", len(entries))
		}
		if len(entries) > 0 && entries[0].ID != "unposted-1" {
			t.Errorf("Expected entry ID unposted-1, got %s", entries[0].ID)
		}
	})

	t.Run("correct ordering (oldest first)", func(t *testing.T) {
		db, err := New(":memory:")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		defer db.Close()

		// Add entries in specific order
		if err := db.SaveEntry("entry-1", []byte(`{"title": "First"}`)); err != nil {
			t.Fatalf("SaveEntry() error = %v", err)
		}
		if err := db.SaveEntry("entry-2", []byte(`{"title": "Second"}`)); err != nil {
			t.Fatalf("SaveEntry() error = %v", err)
		}
		if err := db.SaveEntry("entry-3", []byte(`{"title": "Third"}`)); err != nil {
			t.Fatalf("SaveEntry() error = %v", err)
		}

		entries, err := db.GetUnpostedEntries(0)
		if err != nil {
			t.Fatalf("GetUnpostedEntries() error = %v", err)
		}

		if len(entries) != 3 {
			t.Fatalf("Expected 3 entries, got %d", len(entries))
		}

		// Should be in order added (oldest first)
		if entries[0].ID != "entry-1" {
			t.Errorf("First entry should be entry-1, got %s", entries[0].ID)
		}
		if entries[1].ID != "entry-2" {
			t.Errorf("Second entry should be entry-2, got %s", entries[1].ID)
		}
		if entries[2].ID != "entry-3" {
			t.Errorf("Third entry should be entry-3, got %s", entries[2].ID)
		}
	})

	t.Run("limit parameter works", func(t *testing.T) {
		db, err := New(":memory:")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		defer db.Close()

		// Add 5 entries
		for i := 1; i <= 5; i++ {
			if err := db.SaveEntry("entry-"+string(rune('0'+i)), []byte(`{"title": "Entry"}`)); err != nil {
				t.Fatalf("SaveEntry() error = %v", err)
			}
		}

		// Get only 2
		entries, err := db.GetUnpostedEntries(2)
		if err != nil {
			t.Fatalf("GetUnpostedEntries() error = %v", err)
		}

		if len(entries) != 2 {
			t.Errorf("Expected 2 entries, got %d", len(entries))
		}
	})

	t.Run("returns empty slice when no unposted entries", func(t *testing.T) {
		db, err := New(":memory:")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		defer db.Close()

		entries, err := db.GetUnpostedEntries(0)
		if err != nil {
			t.Fatalf("GetUnpostedEntries() error = %v", err)
		}

		if entries == nil {
			t.Error("Expected empty slice, got nil")
		}
		if len(entries) != 0 {
			t.Errorf("Expected 0 entries, got %d", len(entries))
		}
	})
}

func TestMarkAsPosted(t *testing.T) {
	t.Run("marks entry as posted with timestamp", func(t *testing.T) {
		db, err := New(":memory:")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		defer db.Close()

		if err := db.SaveEntry("test-id", []byte(`{"title": "Test"}`)); err != nil {
			t.Fatalf("SaveEntry() error = %v", err)
		}

		err = db.MarkAsPosted("test-id")
		if err != nil {
			t.Errorf("MarkAsPosted() error = %v", err)
		}

		// Verify posted_at is not NULL
		var postedAt *string
		err = db.conn.QueryRow("SELECT posted_at FROM entries WHERE id = ?", "test-id").Scan(&postedAt)
		if err != nil {
			t.Fatalf("Query error: %v", err)
		}
		if postedAt == nil {
			t.Error("posted_at should not be NULL after marking as posted")
		}
	})

	t.Run("error on non-existent entry ID", func(t *testing.T) {
		db, err := New(":memory:")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		defer db.Close()

		err = db.MarkAsPosted("non-existent")
		if err == nil {
			t.Error("Expected error for non-existent entry, got nil")
		}
	})
}

func TestGetStats(t *testing.T) {
	t.Run("with empty database", func(t *testing.T) {
		db, err := New(":memory:")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		defer db.Close()

		total, posted, unposted, err := db.GetStats()
		if err != nil {
			t.Fatalf("GetStats() error = %v", err)
		}

		if total != 0 || posted != 0 || unposted != 0 {
			t.Errorf("Expected (0, 0, 0), got (%d, %d, %d)", total, posted, unposted)
		}
	})

	t.Run("counts after adding entries", func(t *testing.T) {
		db, err := New(":memory:")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		defer db.Close()

		if err := db.SaveEntry("entry-1", []byte(`{}`)); err != nil {
			t.Fatalf("SaveEntry() error = %v", err)
		}
		if err := db.SaveEntry("entry-2", []byte(`{}`)); err != nil {
			t.Fatalf("SaveEntry() error = %v", err)
		}
		if err := db.SaveEntry("entry-3", []byte(`{}`)); err != nil {
			t.Fatalf("SaveEntry() error = %v", err)
		}

		total, posted, unposted, err := db.GetStats()
		if err != nil {
			t.Fatalf("GetStats() error = %v", err)
		}

		if total != 3 || posted != 0 || unposted != 3 {
			t.Errorf("Expected (3, 0, 3), got (%d, %d, %d)", total, posted, unposted)
		}
	})

	t.Run("counts after posting some entries", func(t *testing.T) {
		db, err := New(":memory:")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		defer db.Close()

		if err := db.SaveEntry("entry-1", []byte(`{}`)); err != nil {
			t.Fatalf("SaveEntry() error = %v", err)
		}
		if err := db.SaveEntry("entry-2", []byte(`{}`)); err != nil {
			t.Fatalf("SaveEntry() error = %v", err)
		}
		if err := db.SaveEntry("entry-3", []byte(`{}`)); err != nil {
			t.Fatalf("SaveEntry() error = %v", err)
		}

		if err := db.MarkAsPosted("entry-1"); err != nil {
			t.Fatalf("MarkAsPosted() error = %v", err)
		}
		if err := db.MarkAsPosted("entry-2"); err != nil {
			t.Fatalf("MarkAsPosted() error = %v", err)
		}

		total, posted, unposted, err := db.GetStats()
		if err != nil {
			t.Fatalf("GetStats() error = %v", err)
		}

		if total != 3 || posted != 2 || unposted != 1 {
			t.Errorf("Expected (3, 2, 1), got (%d, %d, %d)", total, posted, unposted)
		}
	})
}

func TestGetLastFetchTime(t *testing.T) {
	t.Run("returns nil when no entries exist", func(t *testing.T) {
		db, err := New(":memory:")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		defer db.Close()

		fetchTime, err := db.GetLastFetchTime()
		if err != nil {
			t.Fatalf("GetLastFetchTime() error = %v", err)
		}

		if fetchTime != nil {
			t.Errorf("Expected nil, got %v", fetchTime)
		}
	})

	t.Run("returns most recent timestamp", func(t *testing.T) {
		db, err := New(":memory:")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		defer db.Close()

		if err := db.SaveEntry("entry-1", []byte(`{}`)); err != nil {
			t.Fatalf("SaveEntry() error = %v", err)
		}
		if err := db.SaveEntry("entry-2", []byte(`{}`)); err != nil {
			t.Fatalf("SaveEntry() error = %v", err)
		}

		fetchTime, err := db.GetLastFetchTime()
		if err != nil {
			t.Fatalf("GetLastFetchTime() error = %v", err)
		}

		if fetchTime == nil {
			t.Error("Expected timestamp, got nil")
		}
	})
}

func TestGetLastPostTime(t *testing.T) {
	t.Run("returns nil when no entries exist", func(t *testing.T) {
		db, err := New(":memory:")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		defer db.Close()

		postTime, err := db.GetLastPostTime()
		if err != nil {
			t.Fatalf("GetLastPostTime() error = %v", err)
		}

		if postTime != nil {
			t.Errorf("Expected nil, got %v", postTime)
		}
	})

	t.Run("returns nil when no posted entries", func(t *testing.T) {
		db, err := New(":memory:")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		defer db.Close()

		if err := db.SaveEntry("entry-1", []byte(`{}`)); err != nil {
			t.Fatalf("SaveEntry() error = %v", err)
		}

		postTime, err := db.GetLastPostTime()
		if err != nil {
			t.Fatalf("GetLastPostTime() error = %v", err)
		}

		if postTime != nil {
			t.Errorf("Expected nil, got %v", postTime)
		}
	})

	t.Run("returns most recent timestamp when posted", func(t *testing.T) {
		db, err := New(":memory:")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		defer db.Close()

		if err := db.SaveEntry("entry-1", []byte(`{}`)); err != nil {
			t.Fatalf("SaveEntry() error = %v", err)
		}
		if err := db.MarkAsPosted("entry-1"); err != nil {
			t.Fatalf("MarkAsPosted() error = %v", err)
		}

		postTime, err := db.GetLastPostTime()
		if err != nil {
			t.Fatalf("GetLastPostTime() error = %v", err)
		}

		if postTime == nil {
			t.Error("Expected timestamp, got nil")
		}
	})
}

func TestMigrations(t *testing.T) {
	t.Run("GetMigrationVersion on new database", func(t *testing.T) {
		db, err := New(":memory:")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		defer db.Close()

		// After initialization, version should be 1 (initial schema)
		version, err := db.GetMigrationVersion()
		if err != nil {
			t.Fatalf("GetMigrationVersion() error = %v", err)
		}

		// Version should be 2 (initial schema + settings table migration)
		if version != 2 {
			t.Errorf("Expected version 2, got %d", version)
		}
	})

	t.Run("re-running migrations doesn't error", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		// Create database
		db1, err := New(dbPath)
		if err != nil {
			t.Fatalf("First New() error = %v", err)
		}
		db1.Close()

		// Open again (should run migrations again)
		db2, err := New(dbPath)
		if err != nil {
			t.Fatalf("Second New() error = %v", err)
		}
		defer db2.Close()
	})
}
