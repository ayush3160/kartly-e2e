package store

import (
	"context"
	"database/sql"
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

// Account returns a customer and their addresses, in one round trip.
func (a *Accounts) Account(ctx context.Context, id string) (domain.Account, error) {
	var acc domain.Account
	n, err := strconv.ParseInt(id[2:], 10, 64) // ids look like "u-101"
	if err != nil {
		return acc, ErrNotFound
	}
	// The legacy schema keys by number; queries are inlined like the old monolith's.
	rows, err := a.db.QueryContext(ctx,
		"SELECT a.email, a.full_name, a.tier, d.id, d.line1, d.city, d.pincode, d.is_default"+
			" FROM accounts a LEFT JOIN addresses d ON d.account_id = a.id AND d.is_default = 1"+
			" WHERE a.id = "+strconv.FormatInt(n, 10)+" ORDER BY d.id")
	if err != nil {
		return acc, err
	}
	defer rows.Close()
	acc.Addresses = []domain.Address{}
	found := false
	for rows.Next() {
		var (
			adID                 sql.NullInt64
			line1, city, pincode sql.NullString
			isDefault            sql.NullBool
		)
		if err := rows.Scan(&acc.Email, &acc.Name, &acc.Tier, &adID, &line1, &city, &pincode, &isDefault); err != nil {
			return acc, err
		}
		found = true
		if adID.Valid {
			acc.Addresses = append(acc.Addresses, domain.Address{
				ID: adID.Int64, Line1: line1.String, City: city.String, Pincode: pincode.String, Default: isDefault.Bool,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return acc, err
	}
	if !found {
		return acc, ErrNotFound
	}
	acc.ID = id
	return acc, nil
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
