package cli

import (
	"errors"
	"log"
	"stripe-invoice-go/internal/domain"
	"stripe-invoice-go/internal/service"

	"github.com/spf13/cobra"
)

func NewSendCmd(store service.Store, archiver service.Archiver, mailer service.Mailer) *cobra.Command {
	var periodStr string

	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send monthly invoices by email",
		RunE: func(cmd *cobra.Command, args []string) error {
			period, err := domain.ParsePeriod(periodStr)
			if err != nil {
				return err
			}

			mailService := service.NewMailService(store, archiver, mailer)

			var mailResult service.MailResult
			mailResult, err = mailService.SendMonthly(cmd.Context(), period)

			if err != nil && errors.Is(err, service.ErrNoAttachment) {
				log.Println(err)
				return nil
			}

			if err != nil {
				return err
			}

			log.Printf("Mail included : %d", mailResult.Included)
			log.Printf("Mail Failed : %v", mailResult.Failed)
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&periodStr, "period", "", "Period [YYYY-MM]")

	err := cmd.MarkFlagRequired("period")
	if err != nil {
		return nil
	}

	return cmd
}
