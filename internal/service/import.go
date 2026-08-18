package service

import (
	"context"
	"log"
	"stripe-invoice-go/internal/domain"
)

type ImportService struct {
	stripeClient StripeClient
	store        Store
}

func NewImportService(stripeClient StripeClient, store Store) *ImportService {
	return &ImportService{
		stripeClient: stripeClient,
		store:        store,
	}
}

func (service *ImportService) ImportMerchants(ctx context.Context, accountID int64) ([]domain.Merchant, error) {
	merchants, err := service.stripeClient.FetchMerchants(ctx)
	if err != nil {
		return nil, err
	}

	var newMerchants []domain.Merchant
	for _, merchant := range merchants {
		merchant.AccountID = accountID
		created, err := service.store.InsertMerchantIfNotExist(ctx, merchant)

		if err != nil {
			log.Printf("Error inserting merchant %v: %v", merchant, err)
			return nil, err
		}

		if created {
			log.Printf("Merchant %s created", merchant.Name)
			newMerchants = append(newMerchants, merchant)
		}
	}

	return newMerchants, nil

}
