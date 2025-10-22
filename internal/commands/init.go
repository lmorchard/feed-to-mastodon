package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lorchard/feed-to-mastodon/internal/database"
	"github.com/lorchard/feed-to-mastodon/internal/template"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var initDirectory string

// NewInitCmd creates the init command.
func NewInitCmd() *cobra.Command {
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new feed-to-mastodon project",
		Long: `Initialize creates a new feed-to-mastodon project by:
- Creating a default configuration file (feed-to-mastodon.yaml)
- Creating a default template file (post-template.txt)
- Creating the SQLite database and running migrations`,
		RunE: runInit,
	}

	initCmd.Flags().StringVarP(&initDirectory, "directory", "d", ".", "directory to initialize the project in")

	return initCmd
}

func runInit(cmd *cobra.Command, args []string) error {
	// Ensure directory exists
	if err := os.MkdirAll(initDirectory, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Resolve to absolute path for clearer messaging
	absDir, err := filepath.Abs(initDirectory)
	if err != nil {
		return fmt.Errorf("failed to resolve directory path: %w", err)
	}

	logrus.Infof("Initializing feed-to-mastodon project in %s", absDir)

	// Create default config file if it doesn't exist
	configPath := filepath.Join(absDir, "feed-to-mastodon.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := createDefaultConfig(configPath); err != nil {
			return fmt.Errorf("failed to create config file: %w", err)
		}
		logrus.Infof("Created default configuration file: %s", configPath)
	} else {
		logrus.Infof("Configuration file already exists: %s", configPath)
	}

	// Create default template file if it doesn't exist
	templatePath := filepath.Join(absDir, "post-template.txt")
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		if err := createDefaultTemplate(templatePath); err != nil {
			return fmt.Errorf("failed to create template file: %w", err)
		}
		logrus.Infof("Created default template file: %s", templatePath)
	} else {
		logrus.Infof("Template file already exists: %s", templatePath)
	}

	// Create database and run migrations
	dbPath := filepath.Join(absDir, "feed-to-mastodon.db")
	db, err := database.New(dbPath)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}
	defer db.Close()

	logrus.Infof("Created database: %s", dbPath)
	logrus.Info("Initialization complete!")

	// Print next steps
	fmt.Println("\nNext steps:")
	fmt.Printf("1. Edit %s with your feed URL and Mastodon credentials\n", configPath)
	fmt.Printf("2. Optionally customize the post template in %s\n", templatePath)
	fmt.Println("3. Run 'feed-to-mastodon fetch' to fetch feed entries")
	fmt.Println("4. Run 'feed-to-mastodon status' to see what will be posted")
	fmt.Println("5. Run 'feed-to-mastodon post --dry-run' to test posting")
	fmt.Println("6. Run 'feed-to-mastodon post' to post to Mastodon")

	return nil
}

func createDefaultConfig(path string) error {
	defaultConfig := `# Feed to Mastodon Configuration
# REQUIRED: Feed URL to fetch
feed_url: "https://example.com/feed.xml"

# REQUIRED: Mastodon server URL
mastodon_server: "https://mastodon.social"

# REQUIRED: Mastodon access token
# Create a token at: Settings > Development > New Application
# Required scopes: write:statuses
mastodon_token: "your-access-token-here"

# OPTIONAL: Database file path (default: ./feed-to-mastodon.db)
database_path: "feed-to-mastodon.db"

# OPTIONAL: Template file path (default: ./post-template.txt)
template_path: "post-template.txt"

# OPTIONAL: Character limit for posts (default: 500)
character_limit: 500

# OPTIONAL: Post visibility (public, unlisted, private, direct)
# Default: public
post_visibility: "public"

# OPTIONAL: Content warning / spoiler text
# Default: none
# content_warning: "Automated post"

# OPTIONAL: Number of entries to post per run (0 = all)
# Default: 0
posts_per_run: 0
`

	return os.WriteFile(path, []byte(defaultConfig), 0644)
}

func createDefaultTemplate(path string) error {
	defaultTemplate := template.GetDefaultTemplate()
	return os.WriteFile(path, []byte(defaultTemplate), 0644)
}

// GetInitDirectory returns the init directory flag value (used for testing).
func GetInitDirectory() string {
	return initDirectory
}
