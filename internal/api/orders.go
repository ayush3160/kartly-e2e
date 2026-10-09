package api

import (
	"net/http"
	"slices"

	kartlyv1 "github.com/ayush3160/kartly-e2e/gen/kartlyv1"
	"github.com/ayush3160/kartly-e2e/internal/store"
)

var orderStatuses = []string{"", "placed", "paid", "shipped", "delivered", "cancelled", "payment_failed"}

func (s *Server) listOrders(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if !slices.Contains(orderStatuses, status) {
		writeError(w, http.StatusBadRequest, "invalid_status", "unknown status "+status)
		return
	}
	p := principal(r)
	size := queryInt(r, "size", 20, 50)
	page := queryInt(r, "page", 1, 100)
	orders, err := s.Orders.ListOrders(r.Context(), store.ListFilter{
		Tenant: p.Tenant, CustomerID: p.UserID, Status: status, Limit: size, Offset: (page - 1) * size,
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"orders": orders, "page": page, "size": size})
}

// getOrder shows an order to its customer, or to staff auth-svc allows.
func (s *Server) getOrder(w http.ResponseWriter, r *http.Request) {
	p := principal(r)
	ctx := r.Context()
	id := r.PathValue("id")
	perms, err := s.Auth.Permissions(ctx, p.UserID, "order:"+id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	ord, err := s.Orders.OrderByID(ctx, p.Tenant, id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if ord.CustomerID != p.UserID && !slices.Contains(perms.Allowed, "read") {
		writeError(w, http.StatusForbidden, "forbidden", "this order belongs to another customer")
		return
	}
	writeJSON(w, http.StatusOK, ord)
}

// getOrderNotes serves an order's support notes to staff allowed to read
// them. Notes moved off the order detail so it no longer waits on Mongo
// (KART-430).
func (s *Server) getOrderNotes(w http.ResponseWriter, r *http.Request) {
	p := principal(r)
	ctx := r.Context()
	id := r.PathValue("id")
	perms, err := s.Auth.Permissions(ctx, p.UserID, "order:"+id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if !slices.Contains(perms.Allowed, "read_notes") {
		writeError(w, http.StatusForbidden, "forbidden", "you may not read notes on this order")
		return
	}
	notes, err := s.Docs.OrderNotes(ctx, id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"orderId": id, "notes": notes})
}

func (s *Server) cancelOrder(w http.ResponseWriter, r *http.Request) {
	p := principal(r)
	ctx := r.Context()
	id := r.PathValue("id")
	ord, err := s.Orders.OrderByID(ctx, p.Tenant, id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if ord.CustomerID != p.UserID {
		writeError(w, http.StatusForbidden, "forbidden", "this order belongs to another customer")
		return
	}
	ok, err := s.Orders.SetStatus(ctx, p.Tenant, id, "cancelled", "placed", "paid")
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if !ok {
		writeError(w, http.StatusConflict, "not_cancellable", "an order that has shipped cannot be cancelled; request a return")
		return
	}
	refundStatus := "none"
	if ord.Status == "paid" {
		capID, err := s.Orders.CaptureID(ctx, id)
		if err != nil {
			s.fail(w, r, err)
			return
		}
		if capID != "" {
			rf, err := s.GRPC.Payments.Refund(ctx, &kartlyv1.RefundRequest{CaptureId: capID, AmountCents: ord.Total.Amount, Reason: "customer_cancelled"})
			if err != nil {
				s.fail(w, r, err)
				return
			}
			refundStatus = rf.GetStatus()
		}
	}
	if _, err := s.GRPC.Inventory.Release(ctx, &kartlyv1.ReleaseRequest{ReservationId: id}); err != nil {
		s.fail(w, r, err)
		return
	}
	if err := s.Events.Publish(ctx, "order-events", id, map[string]any{"type": "order.cancelled", "orderId": id, "refund": refundStatus}); err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"orderId": id, "status": "cancelled", "refund": refundStatus})
}
