package cmd

import (
	"os"

	"github.com/blontic/awsc/internal/config"
	"github.com/blontic/awsc/internal/debug"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage awsc configuration",
	Long:  `Manage the AWS orgs (SSO start URLs) awsc can log in to`,
	// Config management must work without an active org (e.g. to add the
	// first one or pick a default), so org errors are not fatal here.
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		debug.SetVerbose(verbose)
		setupOrg()
	},
}

var configAddCmd = &cobra.Command{
	Use:   "add [name]",
	Short: "Add an org (SSO start URL)",
	// "init" was the setup command before multi-org support; kept as an alias
	// so existing habits and scripts keep working.
	Aliases: []string{"init"},
	Long:    `Interactively add an org (also used for first-time setup). If no name is given you are asked for one (defaults to the start URL subdomain). The first org becomes the default.`,
	Args:    cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := ""
		if len(args) == 1 {
			name = args[0]
		}
		_, err := config.AddOrg(name)
		exitOnError(err)
	},
}

var configListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List configured orgs",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		exitOnError(config.ListOrgs(os.Stdout))
	},
}

var configUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Set the default org and switch this terminal to it",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		exitOnError(config.UseOrg(args[0]))
	},
}

var configRemoveCmd = &cobra.Command{
	Use:     "remove <name>",
	Aliases: []string{"rm"},
	Short:   "Remove an org from the awsc config",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		exitOnError(config.RemoveOrg(args[0]))
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show [name]",
	Short: "Show an org's configuration (default: the active org)",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := ""
		if len(args) == 1 {
			name = args[0]
		} else if orgErr != nil {
			exitOnError(orgErr)
		}
		exitOnError(config.ShowConfig(os.Stdout, name))
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configAddCmd, configListCmd, configUseCmd, configRemoveCmd, configShowCmd)
}
