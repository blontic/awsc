package cmd

import (
	"fmt"
	"os"

	"github.com/blontic/awsc/internal/config"
	"github.com/blontic/awsc/internal/debug"
	"github.com/spf13/cobra"
)

var removedConfigFlag string
var regionOverride string
var verbose bool
var orgFlag string

// orgErr records a failure to set up the active org. Commands that need an
// org exit with it; config management commands report it only when relevant.
var orgErr error

var rootCmd = &cobra.Command{
	Use:   "awsc",
	Short: "AWS Connect - CLI tool for SSO, RDS, EC2 and Secrets Manager",
	Long:  `AWS Connect - A CLI tool for AWS SSO authentication, RDS port forwarding, EC2 sessions, and Secrets Manager operations.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		debug.SetVerbose(verbose)
		if !needsOrg(cmd) {
			return
		}
		setupOrg()
		exitOnError(orgErr)
		if err := config.EnsureConfigExists(); err != nil {
			exitOnError(fmt.Errorf("setting up configuration: %w", err))
		}
		applyRegionOverride()
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&regionOverride, "region", "", "AWS region to use (overrides config)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().StringVar(&orgFlag, "org", "", "org to use, as named in the awsc config (overrides "+config.OrgEnvVar+" and the default)")

	// --config was replaced by orgs; kept hidden only to explain the change.
	rootCmd.PersistentFlags().StringVar(&removedConfigFlag, "config", "", "")
	_ = rootCmd.PersistentFlags().MarkHidden("config")
}

// needsOrg reports whether a command works with AWS or the awsc config. Help,
// version and shell completion must not migrate, sync or prompt.
func needsOrg(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		switch c.Name() {
		case "help", "version", "completion", cobra.ShellCompRequestCmd, cobra.ShellCompNoDescRequestCmd:
			return false
		}
	}
	return true
}

// setupOrg migrates an old install if needed, keeps ~/.aws/config in sync and
// selects the active org. Failures are recorded in orgErr.
func setupOrg() {
	if removedConfigFlag != "" {
		exitOnError(fmt.Errorf("--config has been removed. All orgs now live in %s.\n"+
			"Add an org with 'awsc config add <name>' and select it with --org <name>", config.GetConfigPath()))
	}
	orgErr = config.ActivateOrg(orgFlag)
}

// applyRegionOverride applies --region over the org's default region.
func applyRegionOverride() {
	if regionOverride == "" {
		return
	}
	if !config.ValidateRegion(regionOverride) {
		exitOnError(fmt.Errorf("invalid AWS region '%s'", regionOverride))
	}
	s := config.Active()
	s.DefaultRegion = regionOverride
	config.SetActive(s)
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
