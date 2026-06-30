package main

import (
	"context"
	"fmt"
	"os"

	"github.com/infancia03/IaC-configuration-drift-detection/internal/report"
	"github.com/infancia03/IaC-configuration-drift-detection/internal/scan"
	"github.com/spf13/cobra"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	var (
		statePath   string
		workspace   string
		awsRegion   string
		awsProfile  string
		output      string
		dryRunCloud bool
	)

	cmd := &cobra.Command{
		Use:   "driftctl",
		Short: "Detect Terraform configuration drift against live cloud infrastructure",
	}

	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Run a drift scan comparing Terraform state to cloud resources",
		RunE: func(cmd *cobra.Command, _ []string) error {
			scanner := scan.NewScanner()
			driftReport, err := scanner.Run(context.Background(), scan.Options{
				Workspace:   workspace,
				StatePath:   statePath,
				AWSRegion:   awsRegion,
				AWSProfile:  awsProfile,
				DryRunCloud: dryRunCloud,
			})
			if err != nil {
				return err
			}

			switch output {
			case "json":
				return report.WriteJSON(os.Stdout, driftReport)
			default:
				return fmt.Errorf("unsupported output format %q (supported: json)", output)
			}
		},
	}

	scanCmd.Flags().StringVar(&statePath, "state", "", "Path to terraform.tfstate file (required)")
	scanCmd.Flags().StringVar(&workspace, "workspace", "default", "Workspace name included in the report")
	scanCmd.Flags().StringVar(&awsRegion, "aws-region", "us-east-1", "AWS region for cloud fetch")
	scanCmd.Flags().StringVar(&awsProfile, "aws-profile", "", "AWS shared config profile name")
	scanCmd.Flags().StringVar(&output, "output", "json", "Output format: json")
	scanCmd.Flags().BoolVar(&dryRunCloud, "dry-run-cloud", false, "Skip cloud API calls and compare state against itself (for local testing)")

	_ = scanCmd.MarkFlagRequired("state")

	cmd.AddCommand(scanCmd)
	return cmd
}
