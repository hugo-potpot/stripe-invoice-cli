package service

import (
	"context"
	"stripe-invoice-go/internal/domain"
	"sync"

	"golang.org/x/sync/errgroup"
)

type ExportService struct {
	stripeClient StripeClient
	store        Store
	archiver     Archiver
}

func NewExportService(stripeClient StripeClient, store Store, archiver Archiver) *ExportService {
	return &ExportService{stripeClient: stripeClient, store: store, archiver: archiver}
}

type MerchantExportFailure struct {
	Merchant domain.Merchant
	Err      error
}

type ExportResult struct {
	ZipPath string
	Failed  []MerchantExportFailure
}

const maxConcurrentExports = 10

func (s *ExportService) ExportInvoices(ctx context.Context, accountID int64, period domain.Period) (ExportResult, error) {
	merchants, err := s.store.ListMerchants(ctx, accountID)
	if err != nil {
		return ExportResult{}, err
	}

	var g errgroup.Group
	var mu sync.Mutex
	var exportResult ExportResult

	g.SetLimit(maxConcurrentExports)

	for _, merchant := range merchants {
		g.Go(func() error {
			if err := s.exportMerchantInvoice(ctx, accountID, period, merchant); err != nil {
				mu.Lock()
				exportResult.Failed = append(exportResult.Failed, MerchantExportFailure{
					Merchant: merchant,
					Err:      err,
				})
				mu.Unlock()
			}
			return nil
		})
	}

	g.Wait()

	zipPath, err := s.archiver.ZipAccount(
		ctx,
		period,
		accountID,
	)

	if err != nil {
		return ExportResult{}, err
	}

	exportResult.ZipPath = zipPath
	return exportResult, nil
}

func (s *ExportService) exportMerchantInvoice(ctx context.Context, accountID int64, period domain.Period, merchant domain.Merchant) error {
	var invoice domain.Invoice
	var err error
	invoice, err = s.stripeClient.ListInvoiceDocuments(ctx, merchant.Token, period)
	if err != nil {
		return err
	}

	var pdf []byte
	pdf, err = s.stripeClient.DownloadPDF(ctx, merchant.Token, invoice.Link)

	if err != nil {
		return err
	}

	if err := s.archiver.WriteInvoice(ctx, period, accountID, merchant, pdf); err != nil {
		return err
	}

	return nil
}
