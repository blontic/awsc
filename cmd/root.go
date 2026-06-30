package cmd

import (
	"fmt"
	"os"

	"github.com/blontic/awsc/internal/config"
	"github.com/blontic/awsc/internal/debug"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var regionOverride string
var verbose bool

var rootCmd = &cobra.Command{
	Use:   "awsc",
	Short: "AWS Connect - CLI tool for SSO, RDS, EC2 and Secrets Manager",
	Long:  `AWS Connect - A CLI tool for AWS SSO authentication, RDS port forwarding, EC2 sessions, and Secrets Manager operations.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		debug.SetVerbose(verbose)
		if err := config.EnsureConfigExists(); err != nil {
			fmt.Printf("Error setting up configuration: %v\n", err)
			os.Exit(1)
		}
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(func() {
		initViper(cfgFile, regionOverride)
	})
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.awsc/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&regionOverride, "region", "", "AWS region to use (overrides config)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
}

// initViper initializes viper configuration
func initViper(cfgFile, regionOverride string) {
	if cfgFile != "" {
		// Check if custom config file exists
		if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
			fmt.Printf("Error: config file '%s' does not exist\n", cfgFile)
			os.Exit(1)
		}
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Look for config in ~/.awsc/config.yaml
		viper.AddConfigPath(home + "/.awsc")
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv()

	// Read config. If the user explicitly passed --config, a read failure is
	// fatal; otherwise the config file is optional and errors are ignored.
	if err := viper.ReadInConfig(); err != nil {
		if cfgFile != "" {
			fmt.Fprintf(os.Stderr, "Error reading config file '%s': %v\n", cfgFile, err)
			os.Exit(1)
		}
	}

	// Set region override if provided
	if regionOverride != "" {
		if !config.ValidateRegion(regionOverride) {
			fmt.Fprintf(os.Stderr, "Error: invalid AWS region '%s'\n", regionOverride)
			os.Exit(1)
		}
		viper.Set("default_region", regionOverride)
	}
}

// validateLocalPort validates a user-supplied local port. A value of 0 is
// allowed and signals "use the default port" to the caller; any other value
// must be in the valid TCP port range.
func validateLocalPort(port int) error {
	if port == 0 {
		return nil
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid local port %d: must be between 1 and 65535", port)
	}
	return nil
}
