package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/blontic/awsc/internal/aws"
	"github.com/blontic/awsc/internal/config"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the org, account and role this terminal uses",
	Long: `Show the org, account and role this terminal uses.

Only local files are read and nothing is changed, so it is fast enough for a
shell prompt (see --short). Use --check to also confirm with AWS that the
credentials work.`,
	Run: runStatusCommand,
}

var statusShort bool
var statusCheck bool

func init() {
	statusCmd.Flags().BoolVar(&statusShort, "short", false, "Print only <account>/<role> (for shell prompts)")
	statusCmd.Flags().BoolVar(&statusCheck, "check", false, "Also confirm with AWS that the credentials work")
	rootCmd.AddCommand(statusCmd)
}

func runStatusCommand(cmd *cobra.Command, args []string) {
	if statusCheck {
		// Checking with AWS needs the full setup other commands get.
		setupOrg()
		exitOnError(orgErr)
		applyRegionOverride()
	} else if regionOverride != "" && !config.ValidateRegion(regionOverride) {
		exitOnError(fmt.Errorf("invalid AWS region '%s'", regionOverride))
	}

	status, err := config.CurrentStatus(orgFlag, regionOverride)
	exitOnError(err)

	if statusShort {
		fmt.Println(shortStatus(status))
	} else {
		printStatus(os.Stdout, status)
	}

	if statusCheck {
		ctx := context.Background()
		identity, err := aws.NewIdentityManager(ctx)
		exitOnError(err)
		arn, err := identity.CallerARN(ctx)
		exitOnError(err)
		if !statusShort {
			fmt.Printf("Identity: %s\n", arn)
		}
	}
}

func shortStatus(s *config.Status) string {
	account := s.AccountName
	if account == "" {
		account = s.AccountID
	}
	return account + "/" + s.RoleName
}

func printStatus(w io.Writer, s *config.Status) {
	account := s.AccountID
	if s.AccountName != "" {
		account = fmt.Sprintf("%s (%s)", s.AccountName, s.AccountID)
	}
	fmt.Fprintf(w, "Org:      %s\n", s.Org)
	fmt.Fprintf(w, "Account:  %s\n", account)
	fmt.Fprintf(w, "Role:     %s\n", s.RoleName)
	fmt.Fprintf(w, "Region:   %s\n", s.Region)
	fmt.Fprintf(w, "Profile:  %s\n", s.Profile)
}
