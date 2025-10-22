package mastodon

import (
	"context"
	"fmt"

	"github.com/lorchard/feed-to-mastodon/internal/database"
	"github.com/lorchard/feed-to-mastodon/internal/template"
	mastodon "github.com/mattn/go-mastodon"
	"github.com/sirupsen/logrus"
)

// Poster handles posting content to Mastodon.
type Poster struct {
	client         *mastodon.Client
	visibility     string
	contentWarning string
}

// New creates a new Poster instance.
func New(server, accessToken, visibility, contentWarning string) (*Poster, error) {
	// Validate visibility
	validVisibilities := map[string]bool{
		"public":   true,
		"unlisted": true,
		"private":  true,
		"direct":   true,
	}

	if !validVisibilities[visibility] {
		return nil, fmt.Errorf("invalid visibility: %s (must be public, unlisted, private, or direct)", visibility)
	}

	// Create Mastodon client
	client := mastodon.NewClient(&mastodon.Config{
		Server:      server,
		AccessToken: accessToken,
	})

	return &Poster{
		client:         client,
		visibility:     visibility,
		contentWarning: contentWarning,
	}, nil
}

// Post posts content to Mastodon.
// If dryRun is true, logs what would be posted without actually posting.
func (p *Poster) Post(content string, dryRun bool) error {
	if dryRun {
		logrus.Info("DRY RUN: Would post to Mastodon")
		logrus.Debugf("DRY RUN: Content: %s", content)
		return nil
	}

	// Create toot
	toot := &mastodon.Toot{
		Status:     content,
		Visibility: p.visibility,
	}

	// Add content warning if set
	if p.contentWarning != "" {
		toot.SpoilerText = p.contentWarning
	}

	// Post to Mastodon
	status, err := p.client.PostStatus(context.Background(), toot)
	if err != nil {
		return fmt.Errorf("failed to post to Mastodon: %w", err)
	}

	logrus.Infof("Posted to Mastodon: %s", status.URL)
	return nil
}

// PostEntries posts multiple entries to Mastodon.
// Returns the count of successfully posted entries.
// Continues on individual posting errors.
func (p *Poster) PostEntries(entries []*database.Entry, renderer *template.Renderer, dryRun bool) (int, error) {
	posted := 0

	for _, entry := range entries {
		// Render template
		content, err := renderer.Render(entry.EntryData)
		if err != nil {
			logrus.Errorf("Failed to render entry %s: %v", entry.ID, err)
			continue
		}

		// Post to Mastodon
		err = p.Post(content, dryRun)
		if err != nil {
			logrus.Errorf("Failed to post entry %s: %v", entry.ID, err)
			continue
		}

		posted++
	}

	if dryRun {
		logrus.Infof("DRY RUN: Would post %d entries", posted)
	} else {
		logrus.Infof("Successfully posted %d/%d entries", posted, len(entries))
	}

	return posted, nil
}
