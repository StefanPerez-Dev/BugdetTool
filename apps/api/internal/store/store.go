package store

import (
	"context"
	"embed"
	"fmt"
	"strings"

	"github.com/StefanPerez-Dev/BugdetTool/apps/api/internal/domain"
	"github.com/StefanPerez-Dev/BugdetTool/apps/api/internal/money"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

type Store struct {
	pool *pgxpool.Pool
}

func Connect(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	s := &Store{pool: pool}
	if err := s.Migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() { s.pool.Close() }

func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

func (s *Store) Migrate(ctx context.Context) error {
	sqlBytes, err := migrationFS.ReadFile("migrations/001_init.sql")
	if err != nil {
		return fmt.Errorf("read migration: %w", err)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
		return fmt.Errorf("apply migration: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO schema_migrations(version) VALUES ('001_init')
		ON CONFLICT (version) DO NOTHING
	`); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) ListAccounts(ctx context.Context) ([]domain.Account, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name, short_name FROM accounts ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Account
	for rows.Next() {
		var a domain.Account
		if err := rows.Scan(&a.ID, &a.Name, &a.ShortName); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) ListCategories(ctx context.Context) ([]domain.Category, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, hue, is_income, is_savings
		FROM categories ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Category
	for rows.Next() {
		var c domain.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Hue, &c.IsIncome, &c.IsSavings); err != nil {
			return nil, err
		}
		c.Color = money.CategoryColor(c.Hue)
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) CategoryByName(ctx context.Context, name string) (*domain.Category, error) {
	var c domain.Category
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, hue, is_income, is_savings FROM categories WHERE name = $1`, name).
		Scan(&c.ID, &c.Name, &c.Hue, &c.IsIncome, &c.IsSavings)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.Color = money.CategoryColor(c.Hue)
	return &c, nil
}

func (s *Store) ListBudgets(ctx context.Context) ([]domain.Budget, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT c.id, c.name, COALESCE(b.amount_cents, 0), c.hue
		FROM categories c
		LEFT JOIN budgets b ON b.category_id = c.id
		WHERE c.is_income = false AND c.is_savings = false
		ORDER BY c.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Budget
	for rows.Next() {
		var b domain.Budget
		if err := rows.Scan(&b.CategoryID, &b.CategoryName, &b.AmountCents, &b.Hue); err != nil {
			return nil, err
		}
		b.Amount = money.FromCents(b.AmountCents)
		b.Color = money.CategoryColor(b.Hue)
		out = append(out, b)
	}
	return out, rows.Err()
}

func (s *Store) UpdateBudget(ctx context.Context, categoryID int64, amountCents int) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO budgets(category_id, amount_cents) VALUES ($1, $2)
		ON CONFLICT (category_id) DO UPDATE SET amount_cents = EXCLUDED.amount_cents`,
		categoryID, amountCents)
	return err
}

type ListTxFilter struct {
	YearMonth *string
	Reviewed  *bool
	Query     string
}

func (s *Store) ListTransactions(ctx context.Context, f ListTxFilter) ([]domain.Transaction, error) {
	var (
		conds []string
		args  []any
	)
	if f.YearMonth != nil && *f.YearMonth != "" {
		args = append(args, *f.YearMonth)
		conds = append(conds, fmt.Sprintf(`to_char(t.booked_on, 'YYYY-MM') = $%d`, len(args)))
	}
	if f.Reviewed != nil {
		args = append(args, *f.Reviewed)
		conds = append(conds, fmt.Sprintf(`t.reviewed = $%d`, len(args)))
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		args = append(args, "%"+strings.ToLower(q)+"%")
		conds = append(conds, fmt.Sprintf(`(lower(t.description) LIKE $%d OR lower(c.name) LIKE $%d)`, len(args), len(args)))
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}
	q := fmt.Sprintf(`
		SELECT t.id, to_char(t.booked_on, 'YYYY-MM-DD'), t.description, t.amount_cents,
		       t.account_id, a.name, a.short_name, t.category_id, c.name, c.hue,
		       t.reviewed, t.recurring, t.flag, t.created_at, t.updated_at
		FROM transactions t
		JOIN accounts a ON a.id = t.account_id
		JOIN categories c ON c.id = t.category_id
		%s
		ORDER BY t.booked_on DESC, t.id DESC`, where)
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTransactions(rows)
}

func scanTransactions(rows pgx.Rows) ([]domain.Transaction, error) {
	var out []domain.Transaction
	for rows.Next() {
		var t domain.Transaction
		if err := rows.Scan(
			&t.ID, &t.Date, &t.Description, &t.AmountCents,
			&t.AccountID, &t.Account, &t.AccountShort, &t.CategoryID, &t.Category, &t.CategoryHue,
			&t.Reviewed, &t.Recurring, &t.Flag, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		t.Amount = money.FromCents(t.AmountCents)
		t.CategoryColor = money.CategoryColor(t.CategoryHue)
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) GetTransaction(ctx context.Context, id int64) (*domain.Transaction, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT t.id, to_char(t.booked_on, 'YYYY-MM-DD'), t.description, t.amount_cents,
		       t.account_id, a.name, a.short_name, t.category_id, c.name, c.hue,
		       t.reviewed, t.recurring, t.flag, t.created_at, t.updated_at
		FROM transactions t
		JOIN accounts a ON a.id = t.account_id
		JOIN categories c ON c.id = t.category_id
		WHERE t.id = $1`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items, err := scanTransactions(rows)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	return &items[0], nil
}

func (s *Store) CreateTransaction(ctx context.Context, in domain.CreateTransactionInput, amountCents int, categoryID int64, reviewed bool) (*domain.Transaction, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO transactions(booked_on, description, amount_cents, account_id, category_id, reviewed, recurring, flag)
		VALUES ($1::date, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`,
		in.Date, in.Description, amountCents, in.AccountID, categoryID, reviewed, in.Recurring, in.Flag,
	).Scan(&id)
	if err != nil {
		return nil, err
	}
	return s.GetTransaction(ctx, id)
}

func (s *Store) UpdateTransaction(ctx context.Context, id int64, in domain.UpdateTransactionInput) (*domain.Transaction, error) {
	cur, err := s.GetTransaction(ctx, id)
	if err != nil || cur == nil {
		return cur, err
	}
	date := cur.Date
	desc := cur.Description
	amountCents := cur.AmountCents
	accountID := cur.AccountID
	categoryID := cur.CategoryID
	reviewed := cur.Reviewed
	recurring := cur.Recurring
	flag := cur.Flag

	if in.Date != nil {
		date = *in.Date
	}
	if in.Description != nil {
		desc = *in.Description
	}
	if in.Amount != nil {
		amountCents = money.ToCents(*in.Amount)
	}
	if in.AccountID != nil {
		accountID = *in.AccountID
	}
	if in.CategoryID != nil {
		categoryID = *in.CategoryID
	}
	if in.Reviewed != nil {
		reviewed = *in.Reviewed
	}
	if in.Recurring != nil {
		recurring = *in.Recurring
	}
	if in.ClearFlag {
		flag = nil
	} else if in.Flag != nil {
		flag = in.Flag
	}

	_, err = s.pool.Exec(ctx, `
		UPDATE transactions SET
		  booked_on = $2::date,
		  description = $3,
		  amount_cents = $4,
		  account_id = $5,
		  category_id = $6,
		  reviewed = $7,
		  recurring = $8,
		  flag = $9,
		  updated_at = now()
		WHERE id = $1`,
		id, date, desc, amountCents, accountID, categoryID, reviewed, recurring, flag,
	)
	if err != nil {
		return nil, err
	}
	return s.GetTransaction(ctx, id)
}

func (s *Store) DeleteTransaction(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM transactions WHERE id = $1`, id)
	return err
}

func (s *Store) ConfirmAllInbox(ctx context.Context) (int64, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE transactions SET reviewed = true, flag = NULL, updated_at = now()
		WHERE reviewed = false`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (s *Store) ClearAll(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM transactions`)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO app_meta(key, value) VALUES ('demo', 'false')
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`)
	return err
}

func (s *Store) DemoEnabled(ctx context.Context) (bool, error) {
	var v string
	err := s.pool.QueryRow(ctx, `SELECT value FROM app_meta WHERE key = 'demo'`).Scan(&v)
	if err == pgx.ErrNoRows {
		return true, nil
	}
	if err != nil {
		return true, err
	}
	return v != "false", nil
}

func (s *Store) ExistsSimilar(ctx context.Context, date string, amountCents int, description string) (bool, error) {
	var n int
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM transactions
		WHERE booked_on = $1::date AND amount_cents = $2 AND description = $3`,
		date, amountCents, description).Scan(&n)
	return n > 0, err
}

func (s *Store) ExistsSameDateAmount(ctx context.Context, date string, amountCents int) (bool, error) {
	var n int
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM transactions
		WHERE booked_on = $1::date AND amount_cents = $2`, date, amountCents).Scan(&n)
	return n > 0, err
}

func (s *Store) CountTransactions(ctx context.Context) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM transactions`).Scan(&n)
	return n, err
}
