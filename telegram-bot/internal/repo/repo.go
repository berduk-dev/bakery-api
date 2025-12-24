package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"tgbot-bad-da-yo/internal/repo/errs"
	"tgbot-bad-da-yo/model"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

func sanitizeCode(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToUpper(s)
	// заменим похожие кириллические на латинские (частый баг в ТГ)
	repl := map[rune]rune{
		'А': 'A', 'В': 'B', 'Е': 'E', 'К': 'K', 'М': 'M', 'Н': 'H', 'О': 'O', 'Р': 'P', 'С': 'S', 'Т': 'T', 'У': 'Y', 'Х': 'X',
		'а': 'A', 'е': 'E', 'о': 'O', 'р': 'P', 'с': 'S', 'х': 'X',
	}
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if rr, ok := repl[r]; ok {
			out = append(out, rr)
		} else {
			// выбросим невидимые символы (ZWSP и т.п.)
			if unicode.IsSpace(r) && r != ' ' {
				continue
			}
			out = append(out, r)
		}
	}
	return string(out)
}

func (r *Repository) GetPrizeByUserID(ctx context.Context, userID int64) (*model.Prize, error) {
	var prize model.Prize
	row := r.pool.QueryRow(ctx, `
        SELECT id, code, prize, created_at, used_at
        FROM prizes
        WHERE telegram_id = $1`, userID)

	err := row.Scan(&prize.ID, &prize.Code, &prize.Prize, &prize.CreatedAt, &prize.UsedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error GetPrizeByCode: %w", err)
	}
	return &prize, nil
}

func (r *Repository) GetPrizeByCode(ctx context.Context, code string) (model.Prize, error) {
	code = sanitizeCode(code)

	var prize model.Prize
	row := r.pool.QueryRow(ctx, `
        SELECT id, code, prize, created_at, used_at
        FROM prizes
        WHERE code = $1`, code)

	err := row.Scan(&prize.ID, &prize.Code, &prize.Prize, &prize.CreatedAt, &prize.UsedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Prize{}, pgx.ErrNoRows
	}
	if err != nil {
		return model.Prize{}, fmt.Errorf("error GetPrizeByCode: %w", err)
	}
	return prize, nil
}

func (r *Repository) ActivateCode(ctx context.Context, code string) error {
	now := time.Now().UTC().Add(3 * time.Hour) // МСК
	_, err := r.pool.Exec(ctx, `
		UPDATE prizes SET used_at = $1 WHERE code = $2`,
		now, code)

	if err != nil {
		return fmt.Errorf("error ActivateCode: %w", err)
	}
	return nil
}

func (r *Repository) CreateUser(ctx context.Context, userID int64) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO users (telegram_id) 
		VALUES ($1)
	`, userID,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				return errs.ErrUserAlreadyExists
			}
		}
		return fmt.Errorf("failed to insert user: %w", err)
	}

	return nil
}

func (r *Repository) GetTelegramIDs(ctx context.Context) ([]int64, error) {
	rows, err := r.pool.Query(ctx, `SELECT telegram_id FROM users WHERE telegram_id IS NOT NULL`)
	if err != nil {
		return nil, fmt.Errorf("error query GetTelegramIDs: %w", err)
	}
	defer rows.Close()

	var telegramIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("error scan GetTelegramIDs: %w", err)
		}
		telegramIDs = append(telegramIDs, id)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error rows.Err - GetTelegramIDs: %w", err)
	}

	return telegramIDs, nil
}

func (r *Repository) AddTelegramIdIntoPrize(ctx context.Context, telegramID int64, code string) error {
	cmd, err := r.pool.Exec(ctx, `
        UPDATE prizes
        SET telegram_id = $1
        WHERE code = $2 AND telegram_id IS NULL
    `, telegramID, code)

	if err != nil {
		return fmt.Errorf("error AddTelegramIdIntoPrize: %w", err)
	}

	// Ничего не обновилось → две причины
	if cmd.RowsAffected() == 0 {
		// проверим, существует ли приз
		var exists bool
		err := r.pool.QueryRow(ctx, `
            SELECT EXISTS(SELECT 1 FROM prizes WHERE code = $1)
        `, code).Scan(&exists)

		if err != nil {
			return fmt.Errorf("error checking prize existence: %w", err)
		}

		if !exists {
			return errs.ErrPrizeNotFound
		}

		return errs.ErrTelegramIDAlreadySet
	}

	return nil
}

func (r *Repository) IsValidByCode(ctx context.Context, code string) (bool, error) {
	row := r.pool.QueryRow(ctx, `
        SELECT telegram_id
        FROM prizes
        WHERE code = $1`, code)

	var telegramID int64

	err := row.Scan(&telegramID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, pgx.ErrNoRows
	}
	if err != nil {
		return false, fmt.Errorf("error GetPrizeByCode: %w", err)
	}
	return true, nil
}

func (r *Repository) GetUsers(ctx context.Context) ([]model.User, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT telegram_id, created_at
		FROM users
		ORDER BY created_at
	`)
	if err != nil {
		return nil, fmt.Errorf("error query GetTelegramIDs: %w", err)
	}
	defer rows.Close()

	var users []model.User

	for rows.Next() {
		var user model.User
		if err := rows.Scan(&user.TelegramID, &user.CreatedAt); err != nil {
			return nil, fmt.Errorf("error scan GetUsers: %w", err)
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error rows.Err - GetUsers: %w", err)
	}

	return users, nil
}

func (r *Repository) UpdateUserPhone(ctx context.Context, userID int64, phone string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE users
		SET phone = $1
		WHERE telegram_id = $2
	`, phone, userID)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			// 23505 - unique constraint violation
			if pgErr.Code == "23505" {
				return errs.ErrPhoneAlreadyExists
			}
		}
		return fmt.Errorf("failed to update user phone: %w", err)
	}

	return nil
}
