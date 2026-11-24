package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestLoadConfig(t *testing.T) {
	t.Run("loads with defaults when no config file", func(t *testing.T) {
		// Save and restore viper state
		oldConfigFile := viper.ConfigFileUsed()
		defer func() {
			viper.Reset()
			if oldConfigFile != "" {
				viper.SetConfigFile(oldConfigFile)
			}
		}()

		viper.Reset()
		unsetEnv(t, "FEED_TO_MASTODON_INCLUDE_CATEGORIES")
		unsetEnv(t, "FEED_TO_MASTODON_EXCLUDE_CATEGORIES")
		cfg, err := LoadConfig("")
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		// Check defaults
		if cfg.TemplateFile != "post-template.txt" {
			t.Errorf("TemplateFile = %v, want %v", cfg.TemplateFile, "post-template.txt")
		}
		if cfg.DatabasePath != "feed-to-mastodon.db" {
			t.Errorf("DatabasePath = %v, want %v", cfg.DatabasePath, "feed-to-mastodon.db")
		}
		if cfg.CharacterLimit != 500 {
			t.Errorf("CharacterLimit = %v, want %v", cfg.CharacterLimit, 500)
		}
		if cfg.MaxItems != 0 {
			t.Errorf("MaxItems = %v, want %v", cfg.MaxItems, 0)
		}
		if cfg.PostVisibility != "public" {
			t.Errorf("PostVisibility = %v, want %v", cfg.PostVisibility, "public")
		}
		if len(cfg.IncludeCategories) != 0 {
			t.Errorf("IncludeCategories = %v, want empty", cfg.IncludeCategories)
		}
		if len(cfg.ExcludeCategories) != 0 {
			t.Errorf("ExcludeCategories = %v, want empty", cfg.ExcludeCategories)
		}
	})

	t.Run("loads from YAML config file", func(t *testing.T) {
		defer viper.Reset()
		viper.Reset()

		// Create temporary config file
		tmpDir := t.TempDir()
		configContent := `feed_url: https://example.com/feed.xml
mastodon_server: https://mastodon.example
mastodon_token: test-token-123
template_path: custom-template.txt
database_path: /tmp/test.db
character_limit: 1000
posts_per_run: 5
post_visibility: unlisted
content_warning: CW Test
`
		configPath := filepath.Join(tmpDir, "feed-to-mastodon.yaml")
		if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
			t.Fatalf("Failed to create test config: %v", err)
		}

		// Change to temp directory and create config there
		oldWd, err := os.Getwd()
		if err != nil {
			t.Fatalf("os.Getwd() error = %v", err)
		}
		defer func() {
			if err := os.Chdir(oldWd); err != nil {
				t.Errorf("os.Chdir() error = %v", err)
			}
		}()
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("os.Chdir() error = %v", err)
		}

		cfg, err := LoadConfig("")
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		// Verify values from config file
		if cfg.FeedURL != "https://example.com/feed.xml" {
			t.Errorf("FeedURL = %v, want %v", cfg.FeedURL, "https://example.com/feed.xml")
		}
		if cfg.MastodonServer != "https://mastodon.example" {
			t.Errorf("MastodonServer = %v", cfg.MastodonServer)
		}
		if cfg.MastodonAccessToken != "test-token-123" {
			t.Errorf("MastodonAccessToken = %v", cfg.MastodonAccessToken)
		}
		if cfg.TemplateFile != "custom-template.txt" {
			t.Errorf("TemplateFile = %v", cfg.TemplateFile)
		}
		if cfg.CharacterLimit != 1000 {
			t.Errorf("CharacterLimit = %v, want 1000", cfg.CharacterLimit)
		}
		if cfg.MaxItems != 5 {
			t.Errorf("MaxItems = %v, want 5", cfg.MaxItems)
		}
		if cfg.PostVisibility != "unlisted" {
			t.Errorf("PostVisibility = %v, want unlisted", cfg.PostVisibility)
		}
		if cfg.ContentWarning != "CW Test" {
			t.Errorf("ContentWarning = %v", cfg.ContentWarning)
		}
	})

	t.Run("merges partial config with defaults", func(t *testing.T) {
		defer viper.Reset()
		viper.Reset()

		tmpDir := t.TempDir()
		configContent := `feed_url: https://example.com/feed.xml
mastodon_server: https://mastodon.example
mastodon_token: test-token
`
		configPath := filepath.Join(tmpDir, "feed-to-mastodon.yaml")
		if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
			t.Fatalf("Failed to create test config: %v", err)
		}

		// Change to temp directory
		oldWd, err := os.Getwd()
		if err != nil {
			t.Fatalf("os.Getwd() error = %v", err)
		}
		defer func() {
			if err := os.Chdir(oldWd); err != nil {
				t.Errorf("os.Chdir() error = %v", err)
			}
		}()
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("os.Chdir() error = %v", err)
		}

		cfg, err := LoadConfig("")
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		// Required fields from config
		if cfg.FeedURL != "https://example.com/feed.xml" {
			t.Errorf("FeedURL not loaded from config")
		}

		// Defaults should still apply
		if cfg.TemplateFile != "post-template.txt" {
			t.Errorf("TemplateFile default not applied")
		}
		if cfg.CharacterLimit != 500 {
			t.Errorf("CharacterLimit default not applied")
		}
	})
}

func TestCategoryConfig(t *testing.T) {
	writeConfig := func(t *testing.T, content string) string {
		t.Helper()

		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "feed-to-mastodon.yaml")
		if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to create test config: %v", err)
		}

		return configPath
	}

	t.Run("loads YAML lists and normalizes values", func(t *testing.T) {
		defer viper.Reset()
		viper.Reset()
		unsetEnv(t, "FEED_TO_MASTODON_INCLUDE_CATEGORIES")
		unsetEnv(t, "FEED_TO_MASTODON_EXCLUDE_CATEGORIES")

		configPath := writeConfig(t, `feed_url: https://example.com/feed.xml
mastodon_server: https://mastodon.example
include_categories:
  - " Golang "
  - golang
  - ""
  - News
exclude_categories:
  - Wishlist
  - " wishlist "
  - "  "
`)

		cfg, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		if want := []string{"Golang", "News"}; !equalStrings(cfg.IncludeCategories, want) {
			t.Errorf("IncludeCategories = %v, want %v", cfg.IncludeCategories, want)
		}
		if want := []string{"Wishlist"}; !equalStrings(cfg.ExcludeCategories, want) {
			t.Errorf("ExcludeCategories = %v, want %v", cfg.ExcludeCategories, want)
		}
	})

	t.Run("comma-separated environment values override the file", func(t *testing.T) {
		defer viper.Reset()
		viper.Reset()

		configPath := writeConfig(t, `feed_url: https://example.com/feed.xml
mastodon_server: https://mastodon.example
include_categories: [file-include]
exclude_categories: [file-exclude]
`)

		t.Setenv("FEED_TO_MASTODON_INCLUDE_CATEGORIES", "Golang, ,news,GOLANG")
		t.Setenv("FEED_TO_MASTODON_EXCLUDE_CATEGORIES", "wishlist, Wishlist")

		cfg, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		if want := []string{"Golang", "news"}; !equalStrings(cfg.IncludeCategories, want) {
			t.Errorf("IncludeCategories = %v, want %v", cfg.IncludeCategories, want)
		}
		if want := []string{"wishlist"}; !equalStrings(cfg.ExcludeCategories, want) {
			t.Errorf("ExcludeCategories = %v, want %v", cfg.ExcludeCategories, want)
		}
	})

	t.Run("empty environment value overrides the file with an empty list", func(t *testing.T) {
		defer viper.Reset()
		viper.Reset()

		configPath := writeConfig(t, `feed_url: https://example.com/feed.xml
mastodon_server: https://mastodon.example
exclude_categories: [wishlist]
`)

		t.Setenv("FEED_TO_MASTODON_EXCLUDE_CATEGORIES", "")

		cfg, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		if len(cfg.ExcludeCategories) != 0 {
			t.Errorf("ExcludeCategories = %v, want empty", cfg.ExcludeCategories)
		}
	})
}

// unsetEnv removes key for the duration of the test and restores it afterwards.
func unsetEnv(t *testing.T, key string) {
	t.Helper()

	value, ok := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("os.Unsetenv(%q) error = %v", key, err)
	}

	t.Cleanup(func() {
		if !ok {
			return
		}
		if err := os.Setenv(key, value); err != nil {
			t.Errorf("os.Setenv(%q) error = %v", key, err)
		}
	})
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: Config{
				FeedURL:             "https://example.com/feed",
				MastodonServer:      "https://mastodon.social",
				MastodonAccessToken: "token",
				PostVisibility:      "public",
			},
			wantErr: false,
		},
		{
			name: "missing feed URL",
			config: Config{
				MastodonServer:      "https://mastodon.social",
				MastodonAccessToken: "token",
				PostVisibility:      "public",
			},
			wantErr: true,
			errMsg:  "feedUrl is required",
		},
		{
			name: "missing mastodon server",
			config: Config{
				FeedURL:             "https://example.com/feed",
				MastodonAccessToken: "token",
				PostVisibility:      "public",
			},
			wantErr: true,
			errMsg:  "mastodonServer is required",
		},
		{
			name: "missing access token - basic validation passes",
			config: Config{
				FeedURL:        "https://example.com/feed",
				MastodonServer: "https://mastodon.social",
				PostVisibility: "public",
			},
			wantErr: false,
		},
		{
			name: "invalid visibility",
			config: Config{
				FeedURL:             "https://example.com/feed",
				MastodonServer:      "https://mastodon.social",
				MastodonAccessToken: "token",
				PostVisibility:      "invalid",
			},
			wantErr: true,
			errMsg:  "postVisibility must be one of",
		},
		{
			name: "valid visibility: public",
			config: Config{
				FeedURL:             "https://example.com/feed",
				MastodonServer:      "https://mastodon.social",
				MastodonAccessToken: "token",
				PostVisibility:      "public",
			},
			wantErr: false,
		},
		{
			name: "valid visibility: unlisted",
			config: Config{
				FeedURL:             "https://example.com/feed",
				MastodonServer:      "https://mastodon.social",
				MastodonAccessToken: "token",
				PostVisibility:      "unlisted",
			},
			wantErr: false,
		},
		{
			name: "valid visibility: private",
			config: Config{
				FeedURL:             "https://example.com/feed",
				MastodonServer:      "https://mastodon.social",
				MastodonAccessToken: "token",
				PostVisibility:      "private",
			},
			wantErr: false,
		},
		{
			name: "valid visibility: direct",
			config: Config{
				FeedURL:             "https://example.com/feed",
				MastodonServer:      "https://mastodon.social",
				MastodonAccessToken: "token",
				PostVisibility:      "direct",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestValidateForPosting(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "missing both access token and client credentials",
			config: Config{
				FeedURL:        "https://example.com/feed",
				MastodonServer: "https://mastodon.social",
				PostVisibility: "public",
			},
			wantErr: true,
			errMsg:  "either mastodon_token or both mastodon_client_id and mastodon_client_secret are required",
		},
		{
			name: "has access token",
			config: Config{
				FeedURL:             "https://example.com/feed",
				MastodonServer:      "https://mastodon.social",
				MastodonAccessToken: "token",
				PostVisibility:      "public",
			},
			wantErr: false,
		},
		{
			name: "has both client credentials",
			config: Config{
				FeedURL:              "https://example.com/feed",
				MastodonServer:       "https://mastodon.social",
				MastodonClientID:     "client_id",
				MastodonClientSecret: "client_secret",
				PostVisibility:       "public",
			},
			wantErr: false,
		},
		{
			name: "has only client ID",
			config: Config{
				FeedURL:          "https://example.com/feed",
				MastodonServer:   "https://mastodon.social",
				MastodonClientID: "client_id",
				PostVisibility:   "public",
			},
			wantErr: true,
			errMsg:  "either mastodon_token or both mastodon_client_id and mastodon_client_secret are required",
		},
		{
			name: "has only client secret",
			config: Config{
				FeedURL:              "https://example.com/feed",
				MastodonServer:       "https://mastodon.social",
				MastodonClientSecret: "client_secret",
				PostVisibility:       "public",
			},
			wantErr: true,
			errMsg:  "either mastodon_token or both mastodon_client_id and mastodon_client_secret are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ValidateForPosting()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateForPosting() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateForPosting() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
