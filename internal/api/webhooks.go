package api

import (
	"net/http"
)

type webhookEvent struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	OrderID string `json:"orderId"`
}

var paymentTransitions = map[string]struct {
	to   string
	from []string
}{
	"payment.captured": {"paid", []string{"placed", "payment_failed"}},
	"payment.failed":   {"payment_failed", []string{"placed"}},
}

var shippingTransitions = map[string]struct {
	to   string
	from []string
}{
	"shipment.dispatched": {"shipped", []string{"paid"}},
	"shipment.delivered":  {"delivered", []string{"shipped"}},
}

func (s *Server) paymentWebhook(w http.ResponseWriter, r *http.Request) {
	s.webhook(w, r, "payments", r.Header.Get("X-Payments-Signature"), paymentTransitions)
}

func (s *Server) shippingWebhook(w http.ResponseWriter, r *http.Request) {
	s.webhook(w, r, "shipping", r.Header.Get("X-Shipping-Signature"), shippingTransitions)
}

// webhook applies a provider event once: duplicates are acknowledged and
// ignored, and an event for a status the order already left is a no-op.
func (s *Server) webhook(w http.ResponseWriter, r *http.Request, provider, signature string, transitions map[string]struct {
	to   string
	from []string
}) {
	if signature != "sig-"+provider {
		writeError(w, http.StatusUnauthorized, "bad_signature", "signature does not verify")
		return
	}
	var ev webhookEvent
	if err := decode(r, &ev); err != nil || ev.ID == "" || ev.OrderID == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "id, type and orderId are required")
		return
	}
	t, ok := transitions[ev.Type]
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{"ignored": true, "reason": "unhandled event type"})
		return
	}
	ctx := r.Context()
	first, err := s.Orders.RecordWebhook(ctx, provider, ev.ID, ev.Type)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if !first {
		writeJSON(w, http.StatusOK, map[string]any{"duplicate": true})
		return
	}
	moved, err := s.Orders.SetStatus(ctx, "acme", ev.OrderID, t.to, t.from...)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if moved {
		if err := s.Events.Publish(ctx, orderEventsTopic, ev.OrderID, map[string]any{"type": "order." + t.to, "orderId": ev.OrderID, "source": provider}); err != nil {
			s.fail(w, r, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"applied": moved, "status": t.to})
}
