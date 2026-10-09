package api

import (
	"net/http"

	kartlyv1 "github.com/ayush3160/kartly-e2e/gen/kartlyv1"
	"github.com/ayush3160/kartly-e2e/internal/clients"
	"github.com/ayush3160/kartly-e2e/internal/domain"
)

const returnWindowDays = 30

type returnReq struct {
	OrderID string   `json:"orderId"`
	SKUs    []string `json:"skus"`
	Reason  string   `json:"reason"`
	Pincode string   `json:"pincode"`
}

func (s *Server) createReturn(w http.ResponseWriter, r *http.Request) {
	var req returnReq
	if err := decode(r, &req); err != nil || req.OrderID == "" || req.Reason == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "orderId and reason are required")
		return
	}
	p := principal(r)
	ctx := r.Context()
	ord, err := s.Orders.OrderByID(ctx, p.Tenant, req.OrderID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if ord.CustomerID != p.UserID {
		writeError(w, http.StatusForbidden, "forbidden", "this order belongs to another customer")
		return
	}
	if ord.Status != "delivered" {
		writeError(w, http.StatusConflict, "not_delivered", "only delivered orders can be returned")
		return
	}
	days, err := s.Orders.DaysSinceDelivered(ctx, ord.ID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if days > returnWindowDays {
		writeError(w, http.StatusUnprocessableEntity, "return_window_closed", "returns are accepted within 30 days of delivery")
		return
	}
	refund := domain.Money{Currency: ord.Total.Currency}
	partial := len(req.SKUs) > 0
	for _, it := range ord.Items {
		if !partial || contains(req.SKUs, it.SKU) {
			refund.Amount += int64(it.Quantity) * it.UnitPrice.Amount
		}
	}
	if refund.Amount == 0 {
		writeError(w, http.StatusUnprocessableEntity, "nothing_to_return", "none of those items are on the order")
		return
	}
	retID := "ret_" + ord.ID[4:]
	label, err := s.Shipping.Label(ctx, clients.LabelRequest{Reference: retID, FromPincode: req.Pincode, ToPincode: s.WarehousePIN, Kind: "return"})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	capID, err := s.Orders.CaptureID(ctx, ord.ID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	status := "label_created"
	if capID != "" {
		rf, err := s.GRPC.Payments.Refund(ctx, &kartlyv1.RefundRequest{CaptureId: capID, AmountCents: refund.Amount, Reason: req.Reason})
		if err != nil {
			s.fail(w, r, err)
			return
		}
		if rf.GetStatus() == "succeeded" {
			status = "refunded"
		}
	}
	ret := domain.Return{ID: retID, OrderID: ord.ID, Status: status, Refund: refund, Label: label, Reason: req.Reason}
	if err := s.Orders.CreateReturn(ctx, ret); err != nil {
		s.fail(w, r, err)
		return
	}
	if err := s.Events.Publish(ctx, "order-events", ord.ID, map[string]any{"type": "order.return_requested", "orderId": ord.ID, "returnId": retID, "refund": refund}); err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, ret)
}

func (s *Server) getReturn(w http.ResponseWriter, r *http.Request) {
	ret, err := s.Orders.ReturnByID(r.Context(), r.PathValue("id"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, ret)
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
