package cli

import (
	"errors"
	"log/slog"
	"os"
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
				slog.WarnContext(cmd.Context(), "nothing to send", "error", err)
				return nil
			}

			if err != nil {
				return err
			}

			slog.InfoContext(cmd.Context(), "send completed", "accounts_included", mailResult.Included, "accounts_failed", len(mailResult.Failed))
			if len(mailResult.Failed) > 0 {
				slog.WarnContext(cmd.Context(), "some accounts had no export to send", "failures", mailResult.Failed)
			}
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&periodStr, "period", "", "Period [YYYY-MM]")

	if err := cmd.MarkFlagRequired("period"); err != nil {
		slog.Error("failed to mark flag required", "flag", "period", "error", err)
		os.Exit(1)
	}

	return cmd
}
