package cli

import (
	"log/slog"
	"os"
	"stripe-invoice-go/internal/domain"
	"stripe-invoice-go/internal/service"
	"stripe-invoice-go/internal/stripe"
	"time"

	"github.com/spf13/cobra"
)

func NewImportCmd(store service.Store, archiver service.Archiver) *cobra.Command {
	var accountID int64
	var cookie string

	var cmd = &cobra.Command{
		Use:   "import",
		Short: "Import listing account merchants",
		RunE: func(cmd *cobra.Command, args []string) error {
			stripeClient, err := stripe.NewClient(cmd.Context(), cookie)
			if err != nil {
				return err
			}
			importService := service.NewImportService(stripeClient, store)

			var merchants []domain.Merchant
			merchants, err = importService.ImportMerchants(cmd.Context(), accountID)
			if err != nil {
				return err
			}

			now := time.Now()
			period := domain.Period{
				Year:  now.Year(),
				Month: now.Month(),
			}
			csvPath, err := archiver.WriteNewMerchantsCSV(cmd.Context(), period, accountID, merchants)
			if err != nil {
				return err
			}

			slog.InfoContext(cmd.Context(), "import completed", "account_id", accountID, "new_merchants", len(merchants), "csv_path", csvPath)
			return nil
		},
	}

	// Flags
	flags := cmd.Flags()
	flags.Int64Var(&accountID, "account-id", 0, "Account ID")
	flags.StringVar(&cookie, "cookie", "", "Cookie")

	// Required Flags
	if err := cmd.MarkFlagRequired("account-id"); err != nil {
		slog.Error("failed to mark flag required", "flag", "account-id", "error", err)
		os.Exit(1)
	}

	if err := cmd.MarkFlagRequired("cookie"); err != nil {
		slog.Error("failed to mark flag required", "flag", "cookie", "error", err)
		os.Exit(1)
	}

	return cmd
}
