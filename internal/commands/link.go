package commands

import (
	"fmt"
	"net/url"

	"github.com/lorchard/feed-to-mastodon/internal/config"
	"github.com/spf13/cobra"
)

// NewLinkCmd creates the link command.
func NewLinkCmd() *cobra.Command {
	linkCmd := &cobra.Command{
		Use:   "link",
		Short: "Generate OAuth authorization link",
		Long: `Generate an OAuth authorization link to begin the authentication flow.

This command requires mastodon_server and mastodon_client_id to be configured.
After visiting the link and authorizing, use the 'code' command with the
authorization code to obtain an access token.`,
		RunE: runLink,
	}

	return linkCmd
}

func runLink(cmd *cobra.Command, args []string) error {
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
		return fmt.Errorf("mastodon_client_id is required - set it in your config file or via FEED_TO_MASTODON_MASTODON_CLIENT_ID environment variable")
	}

	// Construct the authorization URL
	serverURL, err := url.Parse(cfg.MastodonServer)
	if err != nil {
		return fmt.Errorf("invalid mastodon_server URL: %w", err)
	}

	serverURL.Path = "/oauth/authorize"

	// Set query parameters
	params := url.Values{}
	params.Set("client_id", cfg.MastodonClientID)
	params.Set("scope", "read write")
	params.Set("redirect_uri", "urn:ietf:wg:oauth:2.0:oob")
	params.Set("response_type", "code")

	serverURL.RawQuery = params.Encode()

	// Display the authorization link
	fmt.Println("Authorization Link:")
	fmt.Println()
	fmt.Println(serverURL.String())
	fmt.Println()
	fmt.Println("Visit this URL in your browser to authorize the application.")
	fmt.Println("After authorizing, you will receive an authorization code.")
	fmt.Println("Use the 'code' command to exchange it for an access token:")
	fmt.Println()
	fmt.Println("  feed-to-mastodon code <authorization-code>")
	fmt.Println()

	return nil
}
