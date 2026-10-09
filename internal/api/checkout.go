package api

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	kartlyv1 "github.com/ayush3160/kartly-e2e/gen/kartlyv1"
	"github.com/ayush3160/kartly-e2e/internal/clients"
	"github.com/ayush3160/kartly-e2e/internal/domain"
	"github.com/ayush3160/kartly-e2e/internal/store"
)

type checkoutReq struct {
	Coupon    string `json:"coupon,omitempty"`
	CardToken string `json:"cardToken"`
	AddressID int64  `json:"addressId"`
}

// errCoupon is a coupon the customer cannot use, with the reason to show.
type errCoupon struct{ code, msg string }

func (e errCoupon) Error() string { return e.msg }

// priceCart validates the coupon and asks pricing-svc for the totals.
func (s *Server) priceCart(ctx context.Context, p clients.Principal, c domain.Cart, code string) (domain.Quote, error) {
	req := clients.QuoteRequest{
		CartID: c.ID, Customer: p.UserID, Region: s.Region, Currency: c.Currency, Lines: quoteLines(c),
	}
	if code != "" {
		cp, err := s.Orders.CouponByCode(ctx, code)
		if errors.Is(err, store.ErrNotFound) {
			return domain.Quote{}, errCoupon{"invalid_coupon", "coupon " + code + " does not exist"}
		}
		if err != nil {
			return domain.Quote{}, err
		}
		if !cp.Active {
			return domain.Quote{}, errCoupon{"coupon_expired", "coupon " + code + " has expired"}
		}
		req.Coupon = &clients.CouponRef{Code: cp.Code, Kind: cp.Kind, Value: cp.Value}
	}
	return s.Pricing.Quote(ctx, req)
}

func (s *Server) quote(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Coupon  string `json:"coupon,omitempty"`
		Pincode string `json:"pincode"`
	}
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "body must be {coupon, pincode}")
		return
	}
	p := principal(r)
	ctx := r.Context()
	c, err := s.Docs.Cart(ctx, "cart-"+p.UserID, s.currency())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if len(c.Items) == 0 {
		writeError(w, http.StatusUnprocessableEntity, "empty_cart", "add something to the cart first")
		return
	}
	q, err := s.priceCart(ctx, p, c, req.Coupon)
	var ce errCoupon
	if errors.As(err, &ce) {
		writeError(w, http.StatusUnprocessableEntity, ce.code, ce.msg)
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	rates, err := s.Shipping.Rates(ctx, clients.RateRequest{FromPincode: s.WarehousePIN, ToPincode: req.Pincode, WeightGrams: 500 * len(c.Items)})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"quote": q, "shippingOptions": rates})
}

// orderID is derived from the idempotency key, so a retried checkout lands
// on the same order.
func orderID(key string) string {
	sum := sha1.Sum([]byte(key))
	return "ord_" + hex.EncodeToString(sum[:])[:12]
}

func (s *Server) checkout(w http.ResponseWriter, r *http.Request) {
	idem := r.Header.Get("Idempotency-Key")
	if idem == "" {
		writeError(w, http.StatusBadRequest, "missing_idempotency_key", "send an Idempotency-Key header")
		return
	}
	var req checkoutReq
	if err := decode(r, &req); err != nil || req.CardToken == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "cardToken is required")
		return
	}
	p := principal(r)
	ctx := r.Context()
	id := orderID(p.UserID + ":" + idem)

	if n, err := s.Cache.Incr(ctx, "ratelimit:checkout:"+p.UserID, time.Minute); err == nil && n > 20 {
		writeError(w, http.StatusTooManyRequests, "rate_limited", "too many checkouts, try again in a minute")
		return
	}
	claimed, err := s.Cache.Claim(ctx, "idem:checkout:"+p.UserID+":"+idem, id, 24*time.Hour)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if !claimed {
		ord, err := s.Orders.OrderByID(ctx, p.Tenant, id)
		if err != nil {
			writeError(w, http.StatusConflict, "checkout_in_progress", "this checkout is already being processed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"order": ord, "replayed": true})
		return
	}

	c, err := s.Docs.Cart(ctx, "cart-"+p.UserID, s.currency())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if len(c.Items) == 0 {
		_ = s.Cache.Del(ctx, "idem:checkout:"+p.UserID+":"+idem)
		writeError(w, http.StatusUnprocessableEntity, "empty_cart", "add something to the cart first")
		return
	}
	q, err := s.priceCart(ctx, p, c, req.Coupon)
	var ce errCoupon
	if errors.As(err, &ce) {
		_ = s.Cache.Del(ctx, "idem:checkout:"+p.UserID+":"+idem)
		writeError(w, http.StatusUnprocessableEntity, ce.code, ce.msg)
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}

	// Reserve stock before charging.
	lines := make([]*kartlyv1.ReserveLine, 0, len(c.Items))
	for _, it := range c.Items {
		lines = append(lines, &kartlyv1.ReserveLine{Sku: it.SKU, Quantity: int32(it.Quantity)})
	}
	res, err := s.GRPC.Inventory.Reserve(ctx, &kartlyv1.ReserveRequest{ReservationId: id, Lines: lines})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if res.GetStatus() != "reserved" {
		_ = s.Cache.Del(ctx, "idem:checkout:"+p.UserID+":"+idem)
		short := []string{}
		for _, l := range res.GetShort() {
			short = append(short, l.GetSku())
		}
		writeJSON(w, http.StatusConflict, map[string]any{"error": "insufficient_stock", "skus": short})
		return
	}

	auth, err := s.GRPC.Payments.Authorize(ctx, &kartlyv1.AuthorizeRequest{
		OrderId: id, CustomerId: p.UserID, AmountCents: q.Total.Amount, Currency: q.Total.Currency,
		CardToken: req.CardToken, IdempotencyKey: idem,
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if auth.GetStatus() != "approved" {
		_, _ = s.GRPC.Inventory.Release(ctx, &kartlyv1.ReleaseRequest{ReservationId: id})
		_ = s.Cache.Del(ctx, "idem:checkout:"+p.UserID+":"+idem)
		code := http.StatusPaymentRequired
		if auth.GetStatus() == "requires_action" {
			code = http.StatusAccepted
		}
		writeJSON(w, code, map[string]any{"error": "payment_" + auth.GetStatus(), "declineCode": auth.GetDeclineCode()})
		return
	}
	capture, err := s.GRPC.Payments.Capture(ctx, &kartlyv1.CaptureRequest{AuthorizationId: auth.GetAuthorizationId(), AmountCents: q.Total.Amount})
	if err != nil {
		s.fail(w, r, err)
		return
	}

	ord := domain.Order{
		ID: id, CustomerID: p.UserID, Status: "paid", Total: q.Total, Coupon: q.Coupon,
		ShippingFee: q.Shipping, Items: make([]domain.OrderItem, 0, len(c.Items)),
	}
	for _, it := range c.Items {
		ord.Items = append(ord.Items, domain.OrderItem{SKU: it.SKU, Name: it.Name, Quantity: it.Quantity, UnitPrice: it.UnitPrice})
	}
	created, err := s.Orders.CreateOrder(ctx, p.Tenant, ord)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	ord.CreatedAt = created
	if err := s.Orders.SetPayment(ctx, id, auth.GetAuthorizationId(), capture.GetCaptureId()); err != nil {
		s.fail(w, r, err)
		return
	}
	if err := s.Events.Publish(ctx, orderEventsTopic, id, map[string]any{
		"type": "order.paid", "orderId": id, "customerId": p.UserID, "total": q.Total, "items": len(ord.Items),
	}); err != nil {
		s.fail(w, r, err)
		return
	}
	if err := s.Events.Publish(ctx, "email-requests", p.UserID, map[string]any{
		"template": "order-confirmation", "orderId": id, "customerId": p.UserID,
	}); err != nil {
		s.fail(w, r, err)
		return
	}
	if err := s.Docs.DeleteCart(ctx, c.ID); err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"order": ord})
}
