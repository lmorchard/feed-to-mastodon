package commands

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	verbose bool
	debug   bool
)

// InitRootCmd initializes and returns the root command.
func InitRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "feed-to-mastodon",
		Short: "Fetch RSS/Atom feeds and post entries to Mastodon",
		Long: `feed-to-mastodon is a CLI tool that fetches RSS or Atom feeds,
stores entries in a SQLite database, and posts them to Mastodon
using customizable templates.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Configure logging based on flags
			setupLogging()
		},
	}

	// Add persistent flags
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is ./feed-to-mastodon.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug output")

	// Add subcommands
	rootCmd.AddCommand(NewInitCmd())
	rootCmd.AddCommand(NewFetchCmd())
	rootCmd.AddCommand(NewStatusCmd())
	rootCmd.AddCommand(NewPostCmd())
	rootCmd.AddCommand(NewCatchupCmd())
	rootCmd.AddCommand(NewLinkCmd())
	rootCmd.AddCommand(NewCodeCmd())

	return rootCmd
}

// setupLogging configures logrus based on the verbose and debug flags.
func setupLogging() {
	// Set default log level
	logrus.SetLevel(logrus.InfoLevel)

	// Override with verbose or debug if set
	if debug {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.Debug("Debug logging enabled")
	} else if verbose {
		logrus.SetLevel(logrus.InfoLevel)
		logrus.Info("Verbose logging enabled")
	}

	// Use a consistent format
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
}

// Execute runs the root command.
func Execute() {
	rootCmd := InitRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// GetConfigFile returns the config file path from the flag.
func GetConfigFile() string {
	return cfgFile
}
