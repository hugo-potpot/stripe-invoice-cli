package service

import (
	"context"
	"errors"
	"fmt"
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
	// 1. store.ListAccounts(ctx)
	// 2. pour chaque compte : archiver.Attachments(ctx, period, account.ID)
	//    - erreur → collecte dans Failed, continue (pas de zip pour ce compte ce mois-ci)
	// 3. si aucun attachment au total → return MailResult{Failed: failed}, <erreur ou nil selon ton choix ci-dessus, pas d'appel à mailer.Send>
	// 4. sinon → un seul mailer.Send(ctx, period, allAttachments)
	// 5. return MailResult{Included: included, Failed: failed}, err (l'erreur du Send, s'il y en a une)
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
			failed = append(failed, AccountMailFailure{AccountID: account.ID, Err: err})
			continue
		}
		attachments = append(attachments, accountAttachment...)
		included = append(included, account.ID)
	}

	if len(attachments) == 0 {
		return MailResult{Failed: failed, Included: included}, ErrNoAttachment
	}

	if err := s.mailer.Send(ctx, period, attachments); err != nil {
		return MailResult{Failed: failed, Included: included}, err
	}

	return MailResult{Failed: failed, Included: included}, nil
}
