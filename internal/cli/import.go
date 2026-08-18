package cli

import (
	"fmt"
	"log"
	"stripe-invoice-go/internal/service"
	"stripe-invoice-go/internal/stripe"

	"github.com/spf13/cobra"
)

func NewImportCmd(store service.Store) *cobra.Command {
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
			newMerchants, err := importService.ImportMerchants(cmd.Context(), accountID)
			if err != nil {
				return err
			}

			// Log new merchants
			for _, newMerchant := range newMerchants {
				fmt.Printf("New merchant added: %s\n", newMerchant.Name)
			}
			return nil
		},
	}

	// Flags
	flags := cmd.Flags()
	flags.Int64Var(&accountID, "account-id", 0, "Account ID")
	flags.StringVar(&cookie, "cookie", "", "Cookie")

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

	return cmd
}
