package store

import (
	"context"
	"errors"

	"github.com/ayush3160/kartly-e2e/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a row does not exist.
var ErrNotFound = errors.New("not found")

// Orders is the orders database (Postgres).
type Orders struct{ db *pgxpool.Pool }

// NewOrders connects to Postgres.
func NewOrders(ctx context.Context, dsn string) (*Orders, error) {
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	return &Orders{db: db}, nil
}

// Coupon is a discount code.
type Coupon struct {
	Code      string
	Kind      string // percent | fixed | free_shipping
	Value     int64
	Active    bool
	MinAmount int64
}

// CouponByCode looks a coupon up; expired coupons are inactive.
func (o *Orders) CouponByCode(ctx context.Context, code string) (Coupon, error) {
	var c Coupon
	err := o.db.QueryRow(ctx,
		`SELECT code, kind, value, (expires_at > now()) AS active, min_amount FROM coupons WHERE code = $1`, code).
		Scan(&c.Code, &c.Kind, &c.Value, &c.Active, &c.MinAmount)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, ErrNotFound
	}
	return c, err
}

// CreateOrder inserts an order and its items in one transaction and returns
// the stored creation time.
func (o *Orders) CreateOrder(ctx context.Context, tenant string, ord domain.Order) (string, error) {
	tx, err := o.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var created string
	err = tx.QueryRow(ctx,
		`INSERT INTO orders (id, tenant_id, customer_id, status, total_amount, currency, coupon, shipping_fee)
		 VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), $8)
		 ON CONFLICT (id) DO UPDATE SET id = EXCLUDED.id
		 RETURNING to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')`,
		ord.ID, tenant, ord.CustomerID, ord.Status, ord.Total.Amount, ord.Total.Currency, ord.Coupon, ord.ShippingFee.Amount).
		Scan(&created)
	if err != nil {
		return "", err
	}
	for _, it := range ord.Items {
		if _, err := tx.Exec(ctx,
			`INSERT INTO order_items (order_id, sku, name, quantity, unit_amount) VALUES ($1, $2, $3, $4, $5)
			 ON CONFLICT (order_id, sku) DO NOTHING`,
			ord.ID, it.SKU, it.Name, it.Quantity, it.UnitPrice.Amount); err != nil {
			return "", err
		}
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO outbox (aggregate_id, topic, payload) VALUES ($1, 'order-events', $2)`,
		ord.ID, `{"type":"order.created","orderId":"`+ord.ID+`"}`); err != nil {
		return "", err
	}
	return created, tx.Commit(ctx)
}

// OrderByID returns one order of a tenant, with its items.
func (o *Orders) OrderByID(ctx context.Context, tenant, id string) (domain.Order, error) {
	var ord domain.Order
	var coupon *string
	err := o.db.QueryRow(ctx,
		`SELECT id, customer_id, status, total_amount, currency, coupon, shipping_fee,
		        to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		 FROM orders WHERE id = $1 AND tenant_id = $2`, id, tenant).
		Scan(&ord.ID, &ord.CustomerID, &ord.Status, &ord.Total.Amount, &ord.Total.Currency, &coupon, &ord.ShippingFee.Amount, &ord.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ord, ErrNotFound
	}
	if err != nil {
		return ord, err
	}
	if coupon != nil {
		ord.Coupon = *coupon
	}
	ord.ShippingFee.Currency = ord.Total.Currency
	ord.Items, err = o.items(ctx, ord.ID, ord.Total.Currency)
	return ord, err
}

func (o *Orders) items(ctx context.Context, orderID, currency string) ([]domain.OrderItem, error) {
	rows, err := o.db.Query(ctx,
		`SELECT sku, name, quantity, unit_amount FROM order_items WHERE order_id = $1 ORDER BY sku`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.OrderItem{}
	for rows.Next() {
		it := domain.OrderItem{UnitPrice: domain.Money{Currency: currency}}
		if err := rows.Scan(&it.SKU, &it.Name, &it.Quantity, &it.UnitPrice.Amount); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// ListFilter selects a page of orders.
type ListFilter struct {
	Tenant     string
	CustomerID string // empty for admin listings
	Status     string
	Limit      int
	Offset     int
}

// listOrdersSQL is the order-list query; kept in one place so customer and
// admin listings stay in step.
const listOrdersSQL = `SELECT id, customer_id, status, total_amount, currency,
       to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
FROM orders
WHERE tenant_id = $1 AND ($2 = '' OR customer_id = $2) AND ($3 = '' OR status = $3)
  AND status <> 'payment_failed'
ORDER BY created_at DESC, id DESC LIMIT $4 OFFSET $5`

// ListOrders returns a page of orders, newest first, with their items.
func (o *Orders) ListOrders(ctx context.Context, f ListFilter) ([]domain.Order, error) {
	rows, err := o.db.Query(ctx, listOrdersSQL, f.Tenant, f.CustomerID, f.Status, f.Limit, f.Offset)
	if err != nil {
		return nil, err
	}
	var out []domain.Order
	for rows.Next() {
		var ord domain.Order
		if err := rows.Scan(&ord.ID, &ord.CustomerID, &ord.Status, &ord.Total.Amount, &ord.Total.Currency, &ord.CreatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, ord)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		if out[i].Items, err = o.items(ctx, out[i].ID, out[i].Total.Currency); err != nil {
			return nil, err
		}
	}
	if out == nil {
		out = []domain.Order{}
	}
	return out, nil
}

// SetStatus moves an order to a new status when it is in one of from.
func (o *Orders) SetStatus(ctx context.Context, tenant, id, to string, from ...string) (bool, error) {
	tag, err := o.db.Exec(ctx,
		`UPDATE orders SET status = $1, updated_at = now() WHERE id = $2 AND tenant_id = $3 AND status = ANY($4)`,
		to, id, tenant, from)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// SetPayment records the payment authorization on an order.
func (o *Orders) SetPayment(ctx context.Context, id, authID, captureID string) error {
	_, err := o.db.Exec(ctx,
		`UPDATE orders SET authorization_id = $1, capture_id = NULLIF($2, ''), updated_at = now() WHERE id = $3`,
		authID, captureID, id)
	return err
}

// CaptureID is the payment capture of an order.
func (o *Orders) CaptureID(ctx context.Context, id string) (string, error) {
	var c *string
	err := o.db.QueryRow(ctx, `SELECT capture_id FROM orders WHERE id = $1`, id).Scan(&c)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if c == nil {
		return "", err
	}
	return *c, err
}

// CreateReturn stores a return request.
func (o *Orders) CreateReturn(ctx context.Context, r domain.Return) error {
	_, err := o.db.Exec(ctx,
		`INSERT INTO returns (id, order_id, status, refund_amount, currency, reason, label)
		 VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''))
		 ON CONFLICT (id) DO NOTHING`,
		r.ID, r.OrderID, r.Status, r.Refund.Amount, r.Refund.Currency, r.Reason, r.Label)
	return err
}

// ReturnByID returns one return.
func (o *Orders) ReturnByID(ctx context.Context, id string) (domain.Return, error) {
	var r domain.Return
	var label *string
	err := o.db.QueryRow(ctx,
		`SELECT id, order_id, status, refund_amount, currency, reason, label FROM returns WHERE id = $1`, id).
		Scan(&r.ID, &r.OrderID, &r.Status, &r.Refund.Amount, &r.Refund.Currency, &r.Reason, &label)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, ErrNotFound
	}
	if label != nil {
		r.Label = *label
	}
	return r, err
}

// DaysSinceDelivered is how long ago an order was delivered, or -1.
func (o *Orders) DaysSinceDelivered(ctx context.Context, id string) (int, error) {
	var d *int
	err := o.db.QueryRow(ctx,
		`SELECT (current_date - delivered_at::date) FROM orders WHERE id = $1`, id).Scan(&d)
	if errors.Is(err, pgx.ErrNoRows) {
		return -1, ErrNotFound
	}
	if d == nil {
		return -1, err
	}
	return *d, err
}

// RecordWebhook stores a provider event once; false when it was seen before.
func (o *Orders) RecordWebhook(ctx context.Context, provider, eventID, kind string) (bool, error) {
	tag, err := o.db.Exec(ctx,
		`INSERT INTO webhook_events (provider, event_id, kind) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
		provider, eventID, kind)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// Audit writes an admin audit row.
func (o *Orders) Audit(ctx context.Context, actor, action, target string) error {
	_, err := o.db.Exec(ctx,
		`INSERT INTO audit_log (actor, action, target) VALUES ($1, $2, $3)`, actor, action, target)
	return err
}

// Ping checks the database answers.
func (o *Orders) Ping(ctx context.Context) error { return o.db.Ping(ctx) }

// Close releases the pool.
func (o *Orders) Close() { o.db.Close() }
