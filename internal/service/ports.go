package service

import (
	"context"
	"stripe-invoice-go/internal/domain"
)

type Store interface {
	GetAccount(ctx context.Context, accountID int64) (domain.Account, error)
	ListAccounts(ctx context.Context) ([]domain.Account, error)
	InsertMerchantIfNotExist(ctx context.Context, m domain.Merchant) (created bool, err error)
	ListMerchants(ctx context.Context, accountID int64) ([]domain.Merchant, error)
}

type StripeClient interface {
	FetchMerchants(ctx context.Context) ([]domain.Merchant, error)
	ListInvoiceDocuments(ctx context.Context, accountToken string, p domain.Period) (domain.Invoice, error)
	DownloadPDF(ctx context.Context, accountToken, url string) ([]byte, error)
}

type Archiver interface {
	WriteNewMerchantsCSV(ctx context.Context, period domain.Period, accountID int64, merchants []domain.Merchant) (string, error)
	WriteInvoice(ctx context.Context, period domain.Period, accountID int64, merchant domain.Merchant, pdf []byte) error
	ZipAccount(ctx context.Context, period domain.Period, accountID int64) (string, error)
	Attachments(ctx context.Context, period domain.Period, accountID int64) ([]string, error)
}

type Mailer interface {
	Send(ctx context.Context, period domain.Period, attachments []string) error
}
