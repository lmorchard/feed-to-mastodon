package commands

import (
	"encoding/json"
	"testing"

	"github.com/lorchard/feed-to-mastodon/internal/database"
	"github.com/lorchard/feed-to-mastodon/internal/filter"
	"github.com/mmcdole/gofeed"
)

func entryWithCategories(t *testing.T, id string, categories ...string) *database.Entry {
	t.Helper()

	data, err := json.Marshal(gofeed.Item{
		Title:      id,
		Link:       "https://example.com/" + id,
		Categories: categories,
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	return &database.Entry{ID: id, EntryData: data}
}

func entryIDs(entries []*database.Entry) []string {
	ids := make([]string, len(entries))
	for i, entry := range entries {
		ids[i] = entry.ID
	}
	return ids
}

func equalIDs(got []*database.Entry, want []string) bool {
	gotIDs := entryIDs(got)
	if len(gotIDs) != len(want) {
		return false
	}
	for i := range want {
		if gotIDs[i] != want[i] {
			return false
		}
	}
	return true
}

func TestPartitionByCategory(t *testing.T) {
	entries := []*database.Entry{
		entryWithCategories(t, "wishlist", "wishlist"),
		entryWithCategories(t, "news", "news", "links"),
		entryWithCategories(t, "uncategorized"),
		entryWithCategories(t, "mixed-case", "Wishlist"),
		{ID: "broken", EntryData: []byte("not json")},
	}

	t.Run("disabled filter allows everything without decoding", func(t *testing.T) {
		allowed, excluded := partitionByCategory(entries, filter.New(nil, nil))

		if len(excluded) != 0 {
			t.Fatalf("excluded = %v, want none", entryIDs(excluded))
		}
		if len(allowed) != len(entries) {
			t.Fatalf("allowed length = %d, want %d", len(allowed), len(entries))
		}
	})

	t.Run("exclusions win and undecodable entries are allowed", func(t *testing.T) {
		allowed, excluded := partitionByCategory(entries, filter.New(nil, []string{"wishlist"}))

		if want := []string{"wishlist", "mixed-case"}; !equalIDs(excluded, want) {
			t.Errorf("excluded = %v, want %v", entryIDs(excluded), want)
		}
		if want := []string{"news", "uncategorized", "broken"}; !equalIDs(allowed, want) {
			t.Errorf("allowed = %v, want %v", entryIDs(allowed), want)
		}
	})

	t.Run("include requires a matching category", func(t *testing.T) {
		allowed, excluded := partitionByCategory(entries, filter.New([]string{"links"}, nil))

		if want := []string{"news", "broken"}; !equalIDs(allowed, want) {
			t.Errorf("allowed = %v, want %v", entryIDs(allowed), want)
		}
		if want := []string{"wishlist", "uncategorized", "mixed-case"}; !equalIDs(excluded, want) {
			t.Errorf("excluded = %v, want %v", entryIDs(excluded), want)
		}
	})
}
