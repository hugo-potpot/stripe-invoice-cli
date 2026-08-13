package stripe

import (
	"context"
	"os"
	"testing"
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
