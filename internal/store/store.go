package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
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
	row, err := s.q.GetAccount(ctx, id)
	if err != nil {
		return domain.Account{}, fmt.Errorf("get account %d: %w", id, err)
	}
	return toDomainAccount(row), nil
}

func (s *Store) ListAccounts(ctx context.Context) ([]domain.Account, error) {
	rows, err := s.q.ListAccounts(ctx)
	if err != nil {
		return nil, err
	}

	var accounts []domain.Account
	for _, row := range rows {
		accounts = append(accounts, toDomainAccount(row))
	}
	return accounts, nil
}

func (s *Store) CreateMerchant(ctx context.Context, m domain.Merchant) (domain.Merchant, error) {
	row, err := s.q.CreateMerchant(ctx, gen.CreateMerchantParams{
		AccountID: m.AccountID,
		Token:     m.Token,
		Name:      m.Name,
		Identify:  m.Identify,
	})
	if err != nil {
		return domain.Merchant{}, fmt.Errorf("create merchant: %w", err)
	}
	return toDomainMerchant(row), nil
}

func (s *Store) ListMerchants(ctx context.Context, accountID int64) ([]domain.Merchant, error) {
	rows, err := s.q.ListMerchantsByAccount(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("list merchants by accountId: %w", err)
	}

	var results []domain.Merchant
	for _, row := range rows {
		results = append(results, toDomainMerchant(row))
	}

	return results, nil
}

func (s *Store) InsertMerchantIfNotExist(ctx context.Context, m domain.Merchant) (created bool, err error) {
	_, err = s.q.CreateMerchant(ctx, gen.CreateMerchantParams{
		AccountID: m.AccountID,
		Token:     m.Token,
		Name:      m.Name,
		Identify:  m.Identify,
	})

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == UniqueViolation {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func toDomainAccount(a gen.Account) domain.Account {
	return domain.Account{ID: a.ID, Name: a.Name}
}

func toDomainMerchant(m gen.Merchant) domain.Merchant {
	return domain.Merchant{
		ID: m.ID, AccountID: m.AccountID,
		Token: m.Token, Name: m.Name, Identify: m.Identify,
	}
}
