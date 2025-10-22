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
	})

	t.Run("loads from YAML config file", func(t *testing.T) {
		defer viper.Reset()
		viper.Reset()

		// Create temporary config file
		tmpDir := t.TempDir()
		configContent := `feedUrl: https://example.com/feed.xml
mastodonServer: https://mastodon.example
mastodonAccessToken: test-token-123
templateFile: custom-template.txt
databasePath: /tmp/test.db
characterLimit: 1000
maxItems: 5
postVisibility: unlisted
contentWarning: CW Test
`
		configPath := filepath.Join(tmpDir, "feed-to-mastodon.yaml")
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to create test config: %v", err)
		}

		// Change to temp directory and create config there
		oldWd, _ := os.Getwd()
		defer os.Chdir(oldWd)
		os.Chdir(tmpDir)

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
		configContent := `feedUrl: https://example.com/feed.xml
mastodonServer: https://mastodon.example
mastodonAccessToken: test-token
`
		configPath := filepath.Join(tmpDir, "feed-to-mastodon.yaml")
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to create test config: %v", err)
		}

		// Change to temp directory
		oldWd, _ := os.Getwd()
		defer os.Chdir(oldWd)
		os.Chdir(tmpDir)

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
			name: "missing access token",
			config: Config{
				FeedURL:        "https://example.com/feed",
				MastodonServer: "https://mastodon.social",
				PostVisibility: "public",
			},
			wantErr: true,
			errMsg:  "mastodonAccessToken is required",
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
