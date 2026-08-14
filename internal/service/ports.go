package service

import (
	"context"
	"stripe-invoice-go/internal/domain"
)

type Store interface {
	UpsertAccountCookie(ctx context.Context, accountID int64, cookie string) (domain.Account, error)
	GetAccount(ctx context.Context, accountID int64) (domain.Account, error)
	InsertMerchantIfNotExist(ctx context.Context, m domain.Merchant) (created bool, err error)
	ListMerchants(ctx context.Context, accountID int64) ([]domain.Merchant, error)
}

type StripeClient interface {
	FetchMerchants(ctx context.Context) ([]domain.Merchant, error)
	ListInvoiceDocuments(ctx context.Context, accountToken string, p domain.Period) (domain.Invoice, error)
	DownloadPDF(ctx context.Context, accountToken, url string) ([]byte, error)
}

type Mailer interface {
	Send(ctx context.Context, attachments []string) error
}
