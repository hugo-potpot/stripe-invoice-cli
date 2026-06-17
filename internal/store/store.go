package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"stripe-invoice-go/internal/domain"
	"stripe-invoice-go/internal/store/gen"
)

type Store struct {
	pool *pgxpool.Pool
	q    *gen.Queries
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, q: gen.New(pool)}
}

func (s *Store) GetAccount(ctx context.Context, id int64) (domain.Account, error) {
	row, err := s.q.GetAccount(ctx, int32(id))
	if err != nil {
		return domain.Account{}, fmt.Errorf("get account %d: %w", id, err)
	}
	return toDomainAccount(row), nil
}

func (s *Store) CreateMerchant(ctx context.Context, m domain.Merchant) (domain.Merchant, error) {
	row, err := s.q.CreateMerchant(ctx, gen.CreateMerchantParams{
		AccountID: int32(m.AccountID),
		Token:     m.Token,
		Name:      m.Name,
		Identify:  m.Identify,
	})
	if err != nil {
		return domain.Merchant{}, fmt.Errorf("create merchant: %w", err)
	}
	return toDomainMerchant(row), nil
}

func toDomainAccount(a gen.Account) domain.Account {
	return domain.Account{ID: int64(a.ID), Name: a.Name, Cookie: a.Cookie}
}

func toDomainMerchant(m gen.Merchant) domain.Merchant {
	return domain.Merchant{
		ID: int64(m.ID), AccountID: int64(m.AccountID),
		Token: m.Token, Name: m.Name, Identify: m.Identify,
	}
}
