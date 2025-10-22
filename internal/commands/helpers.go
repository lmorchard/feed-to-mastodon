package commands

import (
	"fmt"

	"github.com/lorchard/feed-to-mastodon/internal/config"
	"github.com/lorchard/feed-to-mastodon/internal/database"
)

// getAccessToken retrieves the access token from config or database.
// Priority: config token > database token
func getAccessToken(cfg *config.Config, db *database.DB) (string, error) {
	// First check if access token is in config
	if cfg.MastodonAccessToken != "" {
		return cfg.MastodonAccessToken, nil
	}

	// Otherwise try to get it from the database
	token, err := db.GetSetting("mastodon_access_token")
	if err != nil {
		return "", fmt.Errorf("failed to get access token from database: %w", err)
	}

	if token == nil || *token == "" {
		return "", fmt.Errorf("no access token found - run 'link' and 'code' commands to authenticate, or set mastodon_token in config")
	}

	return *token, nil
}
