package cli

import (
	"log/slog"
	"os"
	"stripe-invoice-go/internal/domain"
	"stripe-invoice-go/internal/service"
	"stripe-invoice-go/internal/stripe"

	"github.com/spf13/cobra"
)

func NewExportCmd(store service.Store, archiver service.Archiver) *cobra.Command {
	var accountID int64
	var cookie string
	var periodStr string

	var cmd = &cobra.Command{

		Use:   "export",
		Short: "Export merchant invoices for a period",
		RunE: func(cmd *cobra.Command, args []string) error {
			period, err := domain.ParsePeriod(periodStr)
			if err != nil {
				return err
			}

			stripeClient, err := stripe.NewClient(cmd.Context(), cookie)
			if err != nil {
				return err
			}
			exportService := service.NewExportService(stripeClient, store, archiver)

			var result service.ExportResult
			result, err = exportService.ExportInvoices(cmd.Context(), accountID, period)
			if err != nil {
				return err
			}

			slog.InfoContext(cmd.Context(), "export completed", "zip_path", result.ZipPath, "merchants_failed", len(result.Failed))
			if len(result.Failed) > 0 {
				failedNames := make([]string, len(result.Failed))
				for i, failure := range result.Failed {
					failedNames[i] = failure.Merchant.Name
				}
				slog.WarnContext(cmd.Context(), "some merchants failed to export", "merchants", failedNames)
			}

			return nil
		},
	}

	// Flags
	flags := cmd.Flags()
	flags.Int64Var(&accountID, "account-id", 0, "Account ID")
	flags.StringVar(&cookie, "cookie", "", "Cookie")
	flags.StringVar(&periodStr, "period", "", "Period [YYYY-MM]")

	// Required Flags
	if err := cmd.MarkFlagRequired("account-id"); err != nil {
		slog.Error("failed to mark flag required", "flag", "account-id", "error", err)
		os.Exit(1)
	}

	if err := cmd.MarkFlagRequired("cookie"); err != nil {
		slog.Error("failed to mark flag required", "flag", "cookie", "error", err)
		os.Exit(1)
	}

	if err := cmd.MarkFlagRequired("period"); err != nil {
		slog.Error("failed to mark flag required", "flag", "period", "error", err)
		os.Exit(1)
	}

	return cmd
}
