package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/blontic/awsc/internal/aws"
	"github.com/spf13/cobra"
)

var consoleCmd = &cobra.Command{
	Use:   "console",
	Short: "Open AWS console in web browser",
	Long:  `Open the AWS Management Console in your default web browser for the currently logged-in account`,
	Run:   runConsole,
}

var consoleService string
var consoleSwitchAccount bool

func init() {
	rootCmd.AddCommand(consoleCmd)
	consoleCmd.Flags().StringVar(&consoleService, "service", "", "AWS service to open (e.g., ec2, s3, rds)")
	consoleCmd.Flags().BoolVarP(&consoleSwitchAccount, "switch-account", "s", false, "Switch AWS account before opening console")
}

func runConsole(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	// Track if we just authenticated
	justAuthenticated := false

	// Handle account switching FIRST, before creating the console manager
	if consoleSwitchAccount {
		if err := handleAccountSwitch(ctx); err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
		justAuthenticated = true
	}

	// Now create console manager with the correct (potentially switched) account
	consoleManager, err := aws.NewConsoleManager(ctx)
	if err != nil {
		if aws.IsAuthError(err) && !justAuthenticated {
			shouldReauth, reAuthErr := aws.PromptForReauth(ctx)
			if reAuthErr != nil {
				fmt.Printf("Error during re-authentication: %v\n", reAuthErr)
				os.Exit(1)
			}
			if !shouldReauth {
				fmt.Printf("Authentication cancelled\n")
				os.Exit(1)
			}
			// Retry after auth
			consoleManager, err = aws.NewConsoleManager(ctx)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	}

	// If we switched accounts, logout of console first to avoid session conflicts
	if consoleSwitchAccount {
		consoleManager.SetLogoutFirst(true)
	}

	if err := consoleManager.OpenConsole(ctx, consoleService); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
