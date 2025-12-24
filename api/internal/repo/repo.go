package repo

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) Repository {
	return Repository{
		pool: pool,
	}
}

func (r *Repository) CreatePrize(ctx context.Context, prizeName string, code string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO prizes (code, prize)
		VALUES ($1, $2)`,
		code, prizeName,
	)
	if err != nil {
		return fmt.Errorf("CreatePrize INSERT: %w", err)
	}

	return nil
}
