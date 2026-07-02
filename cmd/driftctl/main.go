package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/infancia03/IaC-configuration-drift-detection/internal/api"
	"github.com/infancia03/IaC-configuration-drift-detection/internal/config"
	"github.com/infancia03/IaC-configuration-drift-detection/internal/report"
	"github.com/infancia03/IaC-configuration-drift-detection/internal/scan"
	"github.com/infancia03/IaC-configuration-drift-detection/internal/storage"
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
		scanHistoryFile    string
		historyHistoryFile string
		serveHistoryFile   string
		listLimit     int
		addr          string
		scheduleEvery time.Duration
		failOnDrift   bool
		dryRunCloud   bool
	)

	cmd := &cobra.Command{
		Use:   "driftctl",
		Short: "Detect Terraform configuration drift against live cloud infrastructure",
	}

	buildScanOptions := func() (scan.Options, error) {
		driftOptions, err := config.LoadIgnoreConfig(ignoreConfig)
		if err != nil {
			return scan.Options{}, err
		}
		return scan.Options{
			Workspace:     workspace,
			StatePath:     statePath,
			StateS3Bucket: stateS3Bucket,
			StateS3Key:    stateS3Key,
			StateS3Region: stateS3Region,
			AWSRegion:     awsRegion,
			AWSProfile:    awsProfile,
			Drift:         driftOptions,
			DryRunCloud:   dryRunCloud,
		}, nil
	}

	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Run a drift scan comparing Terraform state to cloud resources",
		RunE: func(cmd *cobra.Command, _ []string) error {
			scanOptions, err := buildScanOptions()
			if err != nil {
				return err
			}

			scanner := scan.NewScanner()
			driftReport, err := scanner.Run(context.Background(), scanOptions)
			if err != nil {
				return err
			}
			if scanHistoryFile != "" {
				if err := storage.NewStore(scanHistoryFile).Save(driftReport); err != nil {
					return err
				}
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
	scanCmd.Flags().StringVar(&scanHistoryFile, "history-file", "", "Path to JSON scan history file")
	scanCmd.Flags().BoolVar(&failOnDrift, "fail-on-drift", false, "Exit with a non-zero status when drift is detected")
	scanCmd.Flags().BoolVar(&dryRunCloud, "dry-run-cloud", false, "Skip cloud API calls and compare state against itself (for local testing)")

	historyCmd := &cobra.Command{
		Use:   "history",
		Short: "List persisted drift scan history",
		RunE: func(cmd *cobra.Command, _ []string) error {
			reports, err := storage.NewStore(historyHistoryFile).List()
			if err != nil {
				return err
			}
			if listLimit > 0 && len(reports) > listLimit {
				reports = reports[:listLimit]
			}
			return report.WriteJSON(os.Stdout, reports)
		},
	}
	historyCmd.Flags().StringVar(&historyHistoryFile, "history-file", "drift-history.json", "Path to JSON scan history file")
	historyCmd.Flags().IntVar(&listLimit, "limit", 20, "Maximum number of scans to return")

	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the REST API and dashboard",
		RunE: func(cmd *cobra.Command, _ []string) error {
			scanOptions, err := buildScanOptions()
			if err != nil {
				return err
			}
			store := storage.NewStore(serveHistoryFile)
			server := api.NewServer(api.Options{
				Store:         store,
				DefaultScan:   scanOptions,
				ScheduleEvery: scheduleEvery,
			})
			fmt.Fprintf(os.Stderr, "dashboard listening on http://%s\n", addr)
			return http.ListenAndServe(addr, server.Handler(context.Background()))
		},
	}
	serveCmd.Flags().StringVar(&statePath, "state", "", "Path to local terraform.tfstate file")
	serveCmd.Flags().StringVar(&stateS3Bucket, "state-s3-bucket", "", "S3 bucket containing remote terraform.tfstate")
	serveCmd.Flags().StringVar(&stateS3Key, "state-s3-key", "", "S3 key for remote terraform.tfstate")
	serveCmd.Flags().StringVar(&stateS3Region, "state-s3-region", "", "AWS region for S3 state backend; defaults to --aws-region")
	serveCmd.Flags().StringVar(&workspace, "workspace", "default", "Workspace name included in the report")
	serveCmd.Flags().StringVar(&awsRegion, "aws-region", "us-east-1", "AWS region for cloud fetch")
	serveCmd.Flags().StringVar(&awsProfile, "aws-profile", "", "AWS shared config profile name")
	serveCmd.Flags().StringVar(&ignoreConfig, "ignore-config", "", "Path to JSON ignore rules config")
	serveCmd.Flags().StringVar(&serveHistoryFile, "history-file", "drift-history.json", "Path to JSON scan history file")
	serveCmd.Flags().StringVar(&addr, "addr", "127.0.0.1:8080", "HTTP listen address")
	serveCmd.Flags().DurationVar(&scheduleEvery, "schedule-every", 0, "Run scheduled scans at this interval, for example 15m or 1h")
	serveCmd.Flags().BoolVar(&dryRunCloud, "dry-run-cloud", false, "Skip cloud API calls and compare state against itself")

	cmd.AddCommand(scanCmd)
	cmd.AddCommand(historyCmd)
	cmd.AddCommand(serveCmd)
	return cmd
}
