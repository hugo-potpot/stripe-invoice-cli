package cli

import (
	"log"
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

			log.Printf("Export zip created : %s", result.ZipPath)
			log.Println("Failed export listing: ", result.Failed)

			return nil
		},
	}

	// Flags
	flags := cmd.Flags()
	flags.Int64Var(&accountID, "account-id", 0, "Account ID")
	flags.StringVar(&cookie, "cookie", "", "Cookie")
	flags.StringVar(&periodStr, "period", "", "Period [YYYY-MM]")

	// Required Flags
	err := cmd.MarkFlagRequired("account-id")
	if err != nil {
		log.Fatal(err)
		return nil
	}

	err = cmd.MarkFlagRequired("cookie")
	if err != nil {
		log.Fatal(err)
		return nil
	}

	err = cmd.MarkFlagRequired("period")
	if err != nil {
		log.Fatal(err)
		return nil
	}

	return cmd
}
