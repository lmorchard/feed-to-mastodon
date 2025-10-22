package commands

import (
	"context"
	"fmt"

	"github.com/lorchard/feed-to-mastodon/internal/config"
	"github.com/lorchard/feed-to-mastodon/internal/database"
	mastodon "github.com/mattn/go-mastodon"
	"github.com/spf13/cobra"
)

// NewCodeCmd creates the code command.
func NewCodeCmd() *cobra.Command {
	codeCmd := &cobra.Command{
		Use:   "code <authorization-code>",
		Short: "Exchange authorization code for access token",
		Long: `Exchange an OAuth authorization code for an access token.

This command takes the authorization code you received after visiting the
authorization link (from the 'link' command) and exchanges it for an access token.
The access token is then stored in the database for future use.

This requires mastodon_server, mastodon_client_id, and mastodon_client_secret
to be configured.`,
		Args: cobra.ExactArgs(1),
		RunE: runCode,
	}

	return codeCmd
}

func runCode(cmd *cobra.Command, args []string) error {
	authCode := args[0]

	// Load configuration
	cfg, err := config.LoadConfig(GetConfigFile())
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Validate we have the required fields
	if cfg.MastodonServer == "" {
		return fmt.Errorf("mastodon_server is required")
	}

	if cfg.MastodonClientID == "" {
		return fmt.Errorf("mastodon_client_id is required")
	}

	if cfg.MastodonClientSecret == "" {
		return fmt.Errorf("mastodon_client_secret is required")
	}

	// Create Mastodon client with client credentials
	client := mastodon.NewClient(&mastodon.Config{
		Server:       cfg.MastodonServer,
		ClientID:     cfg.MastodonClientID,
		ClientSecret: cfg.MastodonClientSecret,
	})

	// Exchange authorization code for access token
	fmt.Println("Exchanging authorization code for access token...")
	err = client.GetUserAccessToken(context.Background(), authCode, "urn:ietf:wg:oauth:2.0:oob")
	if err != nil {
		return fmt.Errorf("failed to exchange authorization code: %w", err)
	}

	// Get the access token from the client config
	accessToken := client.Config.AccessToken
	if accessToken == "" {
		return fmt.Errorf("received empty access token")
	}

	// Open database to store the token
	db, err := database.New(cfg.DatabasePath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Store the access token in the database
	err = db.SetSetting("mastodon_access_token", accessToken)
	if err != nil {
		return fmt.Errorf("failed to store access token: %w", err)
	}

	fmt.Println()
	fmt.Println("✓ Successfully obtained and stored access token!")
	fmt.Println()
	fmt.Println("You can now use the 'status' command to verify your account")
	fmt.Println("and the 'post' command to post entries to Mastodon.")
	fmt.Println()

	return nil
}
