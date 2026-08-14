package stripe

import (
	"context"
	"os"
	"stripe-invoice-go/internal/domain"
	"testing"
	"time"
)

func TestNewClient_Bootstrap(t *testing.T) {
	cookie := os.Getenv("STRIPE_TEST_COOKIE")
	if cookie == "" {
		t.Skip("STRIPE_TEST_COOKIE non défini, test d'intégration ignoré")
	}

	client, err := NewClient(context.Background(), cookie)
	if err != nil {
		t.Fatalf("NewClient a échoué: %v", err)
	}
	if client.bearerToken == "" {
		t.Fatal("bearerToken vide après bootstrap")
	}
	t.Logf("bearer obtenu: %s...", client.bearerToken[:10])
}

func TestFetchMerchants(t *testing.T) {
	cookie := os.Getenv("STRIPE_TEST_COOKIE")
	if cookie == "" {
		t.Skip("STRIPE_TEST_COOKIE non défini, test d'intégration ignoré")
	}

	client, err := NewClient(context.Background(), cookie)
	if err != nil {
		t.Fatalf("NewClient a échoué: %v", err)
	}

	merchants, err := client.FetchMerchants(context.Background())
	if err != nil {
		t.Fatalf("FetchMerchants a échoué: %v", err)
	}
	if len(merchants) == 0 {
		t.Fatal("aucun marchand retourné")
	}
	t.Logf("%d marchand(s) récupéré(s), premier: %+v", len(merchants), merchants[0])
}

func TestListInvoiceDocuments(t *testing.T) {
	cookie := os.Getenv("STRIPE_TEST_COOKIE")
	merchantToken := os.Getenv("STRIPE_TEST_MERCHANT_TOKEN")
	if cookie == "" {
		t.Skip("STRIPE_TEST_COOKIE non défini, test d'intégration ignoré")
	}

	if merchantToken == "" {
		t.Skip("STRIPE_TEST_MERCHANT_TOKEN non défini, test d'intégration ignoré")
	}

	client, err := NewClient(context.Background(), cookie)
	if err != nil {
		t.Fatalf("NewClient a échoué: %v", err)
	}

	invoiceDocument, err := client.ListInvoiceDocuments(context.Background(), merchantToken, domain.Period{
		Year:  2026,
		Month: time.August,
	})
	if err != nil {
		t.Fatalf("ListInvoiceDocuments a échoué: %v", err)
	}

	t.Logf("Document récupéré: %+v", invoiceDocument)
}
