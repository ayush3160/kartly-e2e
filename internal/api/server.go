// Package api is orders-api's HTTP surface.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ayush3160/kartly-e2e/internal/clients"
	"github.com/ayush3160/kartly-e2e/internal/store"
)

// orderEventsTopic carries order lifecycle events. v2 is keyed by order id
// with a schema version on every event (KART-408); v1 consumers were moved
// before this release.
const orderEventsTopic = "order-events-v2"

// Server wires the handlers to their dependencies.
type Server struct {
	Orders   *store.Orders
	Accounts *store.Accounts
	Docs     *store.Docs
	Cache    *store.Cache
	Events   *store.Events

	Catalog  *clients.Catalog
	Pricing  *clients.Pricing
	Auth     *clients.Auth
	Flags    *clients.Flags
	Shipping *clients.Shipping
	GRPC     *clients.GRPC

	Region       string
	WarehousePIN string
	Log          *slog.Logger
}

// Routes is the HTTP router.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)

	// account
	mux.HandleFunc("GET /me", s.authed(s.getMe))
	mux.HandleFunc("PUT /me/address", s.authed(s.putAddress))
	mux.HandleFunc("GET /me/orders", s.authed(s.myOrders))

	// catalog
	mux.HandleFunc("GET /products", s.listProducts)
	mux.HandleFunc("GET /products/{id}", s.getProduct)
	mux.HandleFunc("GET /products/{id}/stock", s.getStock)

	// cart
	mux.HandleFunc("GET /cart", s.cartOwner(s.getCart))
	mux.HandleFunc("POST /cart/items", s.cartOwner(s.addItem))
	mux.HandleFunc("PUT /cart/items/{sku}", s.cartOwner(s.updateItem))
	mux.HandleFunc("DELETE /cart/items/{sku}", s.cartOwner(s.removeItem))
	mux.HandleFunc("POST /cart/merge", s.authed(s.mergeCart))

	// checkout
	mux.HandleFunc("POST /checkout/quote", s.authed(s.quote))
	mux.HandleFunc("POST /checkout", s.authed(s.checkout))

	// orders
	mux.HandleFunc("GET /orders", s.authed(s.listOrders))
	mux.HandleFunc("GET /orders/{id}", s.authed(s.getOrder))
	mux.HandleFunc("POST /orders/{id}/cancel", s.authed(s.cancelOrder))

	// returns
	mux.HandleFunc("POST /returns", s.authed(s.createReturn))
	mux.HandleFunc("GET /returns/{id}", s.authed(s.getReturn))

	// admin
	mux.HandleFunc("GET /admin/orders", s.authed(s.adminOnly(s.adminListOrders)))
	mux.HandleFunc("PATCH /admin/orders/{id}", s.authed(s.adminOnly(s.adminUpdateOrder)))

	// webhooks
	mux.HandleFunc("POST /webhooks/payments", s.paymentWebhook)
	mux.HandleFunc("POST /webhooks/shipping", s.shippingWebhook)

	return s.logged(mux)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "up"})
}

type ctxKey struct{}

// principal is the authenticated caller of a request.
func principal(r *http.Request) clients.Principal {
	p, _ := r.Context().Value(ctxKey{}).(clients.Principal)
	return p
}

// authed checks the bearer token. A token's principal is cached in Redis for
// five minutes, so most requests skip auth-svc.
func (s *Server) authed(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if token == "" || token == r.Header.Get("Authorization") {
			writeError(w, http.StatusUnauthorized, "missing_token", "send Authorization: Bearer <token>")
			return
		}
		p, err := s.session(r.Context(), token)
		if err != nil {
			s.fail(w, r, err)
			return
		}
		if !p.Active {
			writeError(w, http.StatusUnauthorized, "token_expired", "sign in again")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, p)))
	}
}

func (s *Server) session(ctx context.Context, token string) (clients.Principal, error) {
	key := "session:" + token
	if v, ok, err := s.Cache.Get(ctx, key); err == nil && ok {
		var p clients.Principal
		if json.Unmarshal([]byte(v), &p) == nil {
			return p, nil
		}
	}
	p, err := s.Auth.Introspect(ctx, token)
	if err != nil {
		return p, err
	}
	if p.Active {
		b, _ := json.Marshal(p)
		_ = s.Cache.Set(ctx, key, string(b), 5*time.Minute)
	}
	return p, nil
}

func (s *Server) adminOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if principal(r).Role != "admin" {
			writeError(w, http.StatusForbidden, "forbidden", "admin role required")
			return
		}
		next(w, r)
	}
}

func (s *Server) logged(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := &statusWriter{ResponseWriter: w, status: 200}
		h.ServeHTTP(sw, r)
		s.Log.Info("request", "method", r.Method, "path", r.URL.Path, "status", sw.status)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// fail maps an error to a response.
func (s *Server) fail(w http.ResponseWriter, r *http.Request, err error) {
	var se *clients.StatusError
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "no such resource")
	case errors.As(err, &se) && se.Status == http.StatusNotFound:
		writeError(w, http.StatusNotFound, "not_found", se.Service+" has no such resource")
	case errors.As(err, &se):
		s.Log.Error("downstream failed", "path", r.URL.Path, "err", err)
		writeError(w, http.StatusBadGateway, "upstream_error", se.Service+" failed")
	default:
		s.Log.Error("request failed", "path", r.URL.Path, "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "something went wrong")
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]string{"error": code, "message": msg})
}

func decode(r *http.Request, v any) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func queryInt(r *http.Request, key string, def, max int) int {
	n, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil || n <= 0 {
		return def
	}
	if n > max {
		return max
	}
	return n
}
