package main

import (
	"context"
	"fmt"
	"os"

	"github.com/infancia03/IaC-configuration-drift-detection/internal/config"
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
		statePath     string
		stateS3Bucket string
		stateS3Key    string
		stateS3Region string
		workspace     string
		awsRegion     string
		awsProfile    string
		output        string
		ignoreConfig  string
		failOnDrift   bool
		dryRunCloud   bool
	)

	cmd := &cobra.Command{
		Use:   "driftctl",
		Short: "Detect Terraform configuration drift against live cloud infrastructure",
	}

	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Run a drift scan comparing Terraform state to cloud resources",
		RunE: func(cmd *cobra.Command, _ []string) error {
			driftOptions, err := config.LoadIgnoreConfig(ignoreConfig)
			if err != nil {
				return err
			}

			scanner := scan.NewScanner()
			driftReport, err := scanner.Run(context.Background(), scan.Options{
				Workspace:     workspace,
				StatePath:     statePath,
				StateS3Bucket: stateS3Bucket,
				StateS3Key:    stateS3Key,
				StateS3Region: stateS3Region,
				AWSRegion:     awsRegion,
				AWSProfile:    awsProfile,
				Drift:         driftOptions,
				DryRunCloud:   dryRunCloud,
			})
			if err != nil {
				return err
			}

			var writeErr error
			switch output {
			case "json":
				writeErr = report.WriteJSON(os.Stdout, driftReport)
			default:
				return fmt.Errorf("unsupported output format %q (supported: json)", output)
			}
			if writeErr != nil {
				return writeErr
			}
			if failOnDrift && len(driftReport.Drifts) > 0 {
				return fmt.Errorf("drift detected: %d drift item(s)", len(driftReport.Drifts))
			}
			return nil
		},
	}

	scanCmd.Flags().StringVar(&statePath, "state", "", "Path to local terraform.tfstate file")
	scanCmd.Flags().StringVar(&stateS3Bucket, "state-s3-bucket", "", "S3 bucket containing remote terraform.tfstate")
	scanCmd.Flags().StringVar(&stateS3Key, "state-s3-key", "", "S3 key for remote terraform.tfstate")
	scanCmd.Flags().StringVar(&stateS3Region, "state-s3-region", "", "AWS region for S3 state backend; defaults to --aws-region")
	scanCmd.Flags().StringVar(&workspace, "workspace", "default", "Workspace name included in the report")
	scanCmd.Flags().StringVar(&awsRegion, "aws-region", "us-east-1", "AWS region for cloud fetch")
	scanCmd.Flags().StringVar(&awsProfile, "aws-profile", "", "AWS shared config profile name")
	scanCmd.Flags().StringVar(&output, "output", "json", "Output format: json")
	scanCmd.Flags().StringVar(&ignoreConfig, "ignore-config", "", "Path to JSON ignore rules config")
	scanCmd.Flags().BoolVar(&failOnDrift, "fail-on-drift", false, "Exit with a non-zero status when drift is detected")
	scanCmd.Flags().BoolVar(&dryRunCloud, "dry-run-cloud", false, "Skip cloud API calls and compare state against itself (for local testing)")

	cmd.AddCommand(scanCmd)
	return cmd
}
