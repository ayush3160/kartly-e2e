package store

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	"github.com/ayush3160/kartly-e2e/internal/domain"
	_ "github.com/go-sql-driver/mysql" // MySQL driver
)

// Accounts is the legacy accounts database (MySQL, text-protocol queries).
type Accounts struct{ db *sql.DB }

// NewAccounts connects to MySQL.
func NewAccounts(dsn string) (*Accounts, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(4)
	return &Accounts{db: db}, nil
}

// Account returns a customer and their addresses.
func (a *Accounts) Account(ctx context.Context, id string) (domain.Account, error) {
	var acc domain.Account
	n, err := strconv.ParseInt(id[2:], 10, 64) // ids look like "u-101"
	if err != nil {
		return acc, ErrNotFound
	}
	// The legacy schema keys by number; queries are inlined like the old monolith's.
	err = a.db.QueryRowContext(ctx,
		"SELECT id, email, full_name, tier FROM accounts WHERE id = "+strconv.FormatInt(n, 10)).
		Scan(&n, &acc.Email, &acc.Name, &acc.Tier)
	if errors.Is(err, sql.ErrNoRows) {
		return acc, ErrNotFound
	}
	if err != nil {
		return acc, err
	}
	acc.ID = id
	rows, err := a.db.QueryContext(ctx,
		"SELECT id, line1, city, pincode, is_default FROM addresses WHERE account_id = "+strconv.FormatInt(n, 10)+" ORDER BY id")
	if err != nil {
		return acc, err
	}
	defer rows.Close()
	acc.Addresses = []domain.Address{}
	for rows.Next() {
		var ad domain.Address
		if err := rows.Scan(&ad.ID, &ad.Line1, &ad.City, &ad.Pincode, &ad.Default); err != nil {
			return acc, err
		}
		acc.Addresses = append(acc.Addresses, ad)
	}
	return acc, rows.Err()
}

// SetDefaultAddress updates a customer's address and makes it the default.
func (a *Accounts) SetDefaultAddress(ctx context.Context, id string, ad domain.Address) error {
	n, err := strconv.ParseInt(id[2:], 10, 64)
	if err != nil {
		return ErrNotFound
	}
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	acct := strconv.FormatInt(n, 10)
	if _, err := tx.ExecContext(ctx, "UPDATE addresses SET is_default = 0 WHERE account_id = "+acct); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		"UPDATE addresses SET line1 = ?, city = ?, pincode = ?, is_default = 1 WHERE id = ? AND account_id = "+acct,
		ad.Line1, ad.City, ad.Pincode, ad.ID); err != nil {
		return err
	}
	return tx.Commit()
}

// Ping checks the database answers.
func (a *Accounts) Ping(ctx context.Context) error { return a.db.PingContext(ctx) }

// Close releases the pool.
func (a *Accounts) Close() { _ = a.db.Close() }
