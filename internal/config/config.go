package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	FeedURL              string
	MastodonServer       string
	MastodonAccessToken  string
	TemplateFile         string
	DatabasePath         string
	CharacterLimit       int
	MaxItems             int
	PostVisibility       string
	ContentWarning       string
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig() (*Config, error) {
	// Set defaults
	viper.SetDefault("templateFile", "template.txt")
	viper.SetDefault("databasePath", "feed-to-mastodon.db")
	viper.SetDefault("characterLimit", 500)
	viper.SetDefault("maxItems", 0)
	viper.SetDefault("postVisibility", "public")
	viper.SetDefault("contentWarning", "")

	// Set config file name and paths
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.config/feed-to-mastodon")

	// Bind environment variables with prefix
	viper.SetEnvPrefix("FEED_TO_MASTODON")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Read config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		// It's okay if config file doesn't exist, we'll use defaults
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Unmarshal into Config struct
	cfg := &Config{
		FeedURL:             viper.GetString("feedUrl"),
		MastodonServer:      viper.GetString("mastodonServer"),
		MastodonAccessToken: viper.GetString("mastodonAccessToken"),
		TemplateFile:        viper.GetString("templateFile"),
		DatabasePath:        viper.GetString("databasePath"),
		CharacterLimit:      viper.GetInt("characterLimit"),
		MaxItems:            viper.GetInt("maxItems"),
		PostVisibility:      viper.GetString("postVisibility"),
		ContentWarning:      viper.GetString("contentWarning"),
	}

	return cfg, nil
}

// Validate checks that required fields are set and valid
func (c *Config) Validate() error {
	if c.FeedURL == "" {
		return fmt.Errorf("feedUrl is required")
	}

	if c.MastodonServer == "" {
		return fmt.Errorf("mastodonServer is required")
	}

	if c.MastodonAccessToken == "" {
		return fmt.Errorf("mastodonAccessToken is required")
	}

	// Validate post visibility
	validVisibilities := map[string]bool{
		"public":   true,
		"unlisted": true,
		"private":  true,
		"direct":   true,
	}

	if !validVisibilities[c.PostVisibility] {
		return fmt.Errorf("postVisibility must be one of: public, unlisted, private, direct")
	}

	return nil
}
