package api

import (
	"net/http"
	"slices"

	"github.com/ayush3160/kartly-e2e/internal/store"
)

var adminTransitions = map[string][]string{
	"shipped":   {"paid"},
	"delivered": {"shipped"},
	"cancelled": {"placed", "paid"},
}

func (s *Server) adminListOrders(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if !slices.Contains(orderStatuses, status) {
		writeError(w, http.StatusBadRequest, "invalid_status", "unknown status "+status)
		return
	}
	p := principal(r)
	size := queryInt(r, "size", 25, 100)
	page := queryInt(r, "page", 1, 100)
	orders, err := s.Orders.ListOrders(r.Context(), store.ListFilter{
		Tenant: p.Tenant, CustomerID: r.URL.Query().Get("customer"), Status: status, Limit: size, Offset: (page - 1) * size,
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"orders": orders, "page": page, "size": size})
}

func (s *Server) adminUpdateOrder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Status string `json:"status"`
	}
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "body must be {status}")
		return
	}
	from, ok := adminTransitions[req.Status]
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_status", "status must be shipped, delivered or cancelled")
		return
	}
	p := principal(r)
	ctx := r.Context()
	id := r.PathValue("id")
	bulk, err := s.Flags.Enabled(ctx, "admin-status-audit", p.Tenant)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	moved, err := s.Orders.SetStatus(ctx, p.Tenant, id, req.Status, from...)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if !moved {
		writeError(w, http.StatusConflict, "invalid_transition", "order cannot move to "+req.Status+" from its current status")
		return
	}
	if bulk {
		if err := s.Orders.Audit(ctx, p.UserID, "order.status."+req.Status, id); err != nil {
			s.fail(w, r, err)
			return
		}
	}
	if err := s.Events.Publish(ctx, orderEventsTopic, id, map[string]any{"type": "order." + req.Status, "orderId": id, "by": p.UserID}); err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"orderId": id, "status": req.Status})
}
