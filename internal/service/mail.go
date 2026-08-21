package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"stripe-invoice-go/internal/domain"
)

type AccountMailFailure struct {
	AccountID int64
	Err       error
}

type MailResult struct {
	Included []int64
	Failed   []AccountMailFailure
}

type MailService struct {
	store    Store
	archiver Archiver
	mailer   Mailer
}

var ErrNoAttachment = errors.New("no attachment")

func NewMailService(store Store, archiver Archiver, mailer Mailer) *MailService {
	return &MailService{store: store, archiver: archiver, mailer: mailer}
}

func (s *MailService) SendMonthly(ctx context.Context, period domain.Period) (MailResult, error) {
	accounts, err := s.store.ListAccounts(ctx)
	if err != nil {
		return MailResult{}, fmt.Errorf("listing accounts: %w", err)
	}

	var attachments []string
	var failed []AccountMailFailure
	var included []int64

	for _, account := range accounts {
		accountAttachment, err := s.archiver.Attachments(ctx, period, account.ID)
		if err != nil {
			slog.WarnContext(ctx, "no export attachments for account, skipping", "account_id", account.ID, "period", period.String(), "error", err)
			failed = append(failed, AccountMailFailure{AccountID: account.ID, Err: err})
			continue
		}
		attachments = append(attachments, accountAttachment...)
		included = append(included, account.ID)
	}

	if len(attachments) == 0 {
		slog.WarnContext(ctx, "no attachments found for any account, skipping mail", "period", period.String())
		return MailResult{Failed: failed, Included: included}, ErrNoAttachment
	}

	slog.InfoContext(ctx, "sending monthly mail", "period", period.String(), "accounts_included", len(included), "accounts_failed", len(failed))
	if err := s.mailer.Send(ctx, period, attachments); err != nil {
		slog.ErrorContext(ctx, "sending monthly mail failed", "period", period.String(), "error", err)
		return MailResult{Failed: failed, Included: included}, err
	}

	return MailResult{Failed: failed, Included: included}, nil
}
