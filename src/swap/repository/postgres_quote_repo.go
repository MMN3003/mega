package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/MMN3003/mega/src/logger"
	"github.com/MMN3003/mega/src/swap/domain"
	"github.com/shopspring/decimal"
)

type PostgresQuoteRepo struct {
	db  *sql.DB
	log *logger.Logger
}

func NewPostgresQuoteRepo(db *sql.DB, log *logger.Logger) *PostgresQuoteRepo {
	return &PostgresQuoteRepo{db: db, log: log}
}

func (r *PostgresQuoteRepo) Save(ctx context.Context, q *domain.Quote) error {
	query := `
	INSERT INTO quotes (
		id, from_network, from_token, to_network, to_token,
		amount_in, amount_out, expires_at, created_at, used, user_address
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	`

	_, err := r.db.ExecContext(ctx, query,
		q.ID,
		q.FromNetwork,
		q.FromToken,
		q.ToNetwork,
		q.ToToken,
		q.AmountIn.String(),
		q.AmountOut.String(),
		q.ExpiresAt,
		q.CreatedAt,
		q.Used,
		q.UserAddress,
	)
	if err != nil {
		r.log.Errorf("failed to save quote: %v", err)
	}
	return err
}

func (r *PostgresQuoteRepo) GetByID(ctx context.Context, id string) (*domain.Quote, error) {
	query := `SELECT id, from_network, from_token, to_network, to_token, amount_in, amount_out, expires_at, created_at, used, user_address FROM quotes WHERE id=$1`
	row := r.db.QueryRowContext(ctx, query, id)

	var q domain.Quote
	var amountInStr, amountOutStr string

	err := row.Scan(
		&q.ID,
		&q.FromNetwork,
		&q.FromToken,
		&q.ToNetwork,
		&q.ToToken,
		&amountInStr,
		&amountOutStr,
		&q.ExpiresAt,
		&q.CreatedAt,
		&q.Used,
		&q.UserAddress,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // or custom NotFound error
		}
		r.log.Errorf("failed to get quote by id: %v", err)
		return nil, err
	}

	// Parse decimal strings into decimal.Decimal
	q.AmountIn, err = decimal.NewFromString(amountInStr)
	if err != nil {
		return nil, err
	}
	q.AmountOut, err = decimal.NewFromString(amountOutStr)
	if err != nil {
		return nil, err
	}

	return &q, nil
}

func (r *PostgresQuoteRepo) MarkUsed(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE quotes SET used=true WHERE id=$1", id)
	if err != nil {
		r.log.Errorf("failed to mark quote used: %v", err)
	}
	return err
}

func (r *PostgresQuoteRepo) ListActive(ctx context.Context) ([]*domain.Quote, error) {
	query := `SELECT id, from_network, from_token, to_network, to_token, amount_in, amount_out, expires_at, created_at, used, user_address FROM quotes WHERE used=false AND expires_at > now()`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		r.log.Errorf("failed to list active quotes: %v", err)
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Quote

	for rows.Next() {
		var q domain.Quote
		var amountInStr, amountOutStr string

		err := rows.Scan(
			&q.ID,
			&q.FromNetwork,
			&q.FromToken,
			&q.ToNetwork,
			&q.ToToken,
			&amountInStr,
			&amountOutStr,
			&q.ExpiresAt,
			&q.CreatedAt,
			&q.Used,
			&q.UserAddress,
		)
		if err != nil {
			r.log.Errorf("failed to scan quote row: %v", err)
			return nil, err
		}

		q.AmountIn, err = decimal.NewFromString(amountInStr)
		if err != nil {
			return nil, err
		}
		q.AmountOut, err = decimal.NewFromString(amountOutStr)
		if err != nil {
			return nil, err
		}

		out = append(out, &q)
	}

	if err := rows.Err(); err != nil {
		r.log.Errorf("rows iteration error: %v", err)
		return nil, err
	}

	return out, nil
}
