package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/ayush3160/kartly-e2e/internal/clients"
	"github.com/ayush3160/kartly-e2e/internal/domain"
)

const maxQuantity = 10

type cartKey struct{}

// cartOwner resolves whose cart a request is about: the signed-in user's, or
// a guest cart named by the X-Cart-Id header.
func (s *Server) cartOwner(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			s.authed(func(w http.ResponseWriter, r *http.Request) {
				id := "cart-" + principal(r).UserID
				next(w, r.WithContext(context.WithValue(r.Context(), cartKey{}, id)))
			})(w, r)
			return
		}
		guest := r.Header.Get("X-Cart-Id")
		if !strings.HasPrefix(guest, "guest-") {
			writeError(w, http.StatusBadRequest, "no_cart", "sign in or send X-Cart-Id: guest-<id>")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), cartKey{}, guest)))
	}
}

func cartID(r *http.Request) string { id, _ := r.Context().Value(cartKey{}).(string); return id }

func (s *Server) currency() string {
	if s.Region == "US" {
		return "USD"
	}
	return "INR"
}

func (s *Server) getCart(w http.ResponseWriter, r *http.Request) {
	c, err := s.Docs.Cart(r.Context(), cartID(r), s.currency())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, withTotals(c))
}

type addItemReq struct {
	ProductID string `json:"productId"`
	Quantity  int    `json:"quantity"`
}

func (s *Server) addItem(w http.ResponseWriter, r *http.Request) {
	var req addItemReq
	if err := decode(r, &req); err != nil || req.ProductID == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "productId and quantity are required")
		return
	}
	if req.Quantity < 1 || req.Quantity > maxQuantity {
		writeError(w, http.StatusUnprocessableEntity, "invalid_quantity", "quantity must be between 1 and 10")
		return
	}
	p, err := s.product(r.Context(), req.ProductID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if !p.InStock {
		writeError(w, http.StatusConflict, "out_of_stock", p.Name+" is out of stock")
		return
	}
	c, err := s.Docs.Cart(r.Context(), cartID(r), s.currency())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	found := false
	for i := range c.Items {
		if c.Items[i].SKU == p.SKU {
			c.Items[i].Quantity += req.Quantity
			if c.Items[i].Quantity > maxQuantity {
				writeError(w, http.StatusUnprocessableEntity, "invalid_quantity", "at most 10 of one item")
				return
			}
			c.Items[i].UnitPrice = p.Price
			found = true
		}
	}
	if !found {
		c.Items = append(c.Items, domain.CartItem{ProductID: p.ID, SKU: p.SKU, Name: p.Name, Quantity: req.Quantity, UnitPrice: p.Price})
	}
	if err := s.Docs.SaveCart(r.Context(), c); err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, withTotals(c))
}

func (s *Server) updateItem(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Quantity int `json:"quantity"`
	}
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "quantity is required")
		return
	}
	if req.Quantity < 0 || req.Quantity > maxQuantity {
		writeError(w, http.StatusUnprocessableEntity, "invalid_quantity", "quantity must be between 0 and 10")
		return
	}
	s.changeItem(w, r, r.PathValue("sku"), req.Quantity)
}

func (s *Server) removeItem(w http.ResponseWriter, r *http.Request) {
	s.changeItem(w, r, r.PathValue("sku"), 0)
}

func (s *Server) changeItem(w http.ResponseWriter, r *http.Request, sku string, qty int) {
	c, err := s.Docs.Cart(r.Context(), cartID(r), s.currency())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	kept := c.Items[:0]
	found := false
	for _, it := range c.Items {
		if it.SKU == sku {
			found = true
			if qty == 0 {
				continue
			}
			it.Quantity = qty
		}
		kept = append(kept, it)
	}
	if !found {
		writeError(w, http.StatusNotFound, "not_in_cart", sku+" is not in the cart")
		return
	}
	c.Items = kept
	if err := s.Docs.SaveCart(r.Context(), c); err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, withTotals(c))
}

// mergeCart moves a guest cart into the signed-in user's cart.
func (s *Server) mergeCart(w http.ResponseWriter, r *http.Request) {
	guestID := r.Header.Get("X-Cart-Id")
	if !strings.HasPrefix(guestID, "guest-") {
		writeError(w, http.StatusBadRequest, "no_cart", "send X-Cart-Id: guest-<id>")
		return
	}
	ctx := r.Context()
	guest, err := s.Docs.Cart(ctx, guestID, s.currency())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	userCart, err := s.Docs.Cart(ctx, "cart-"+principal(r).UserID, s.currency())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	for _, g := range guest.Items {
		merged := false
		for i := range userCart.Items {
			if userCart.Items[i].SKU == g.SKU {
				userCart.Items[i].Quantity = min(userCart.Items[i].Quantity+g.Quantity, maxQuantity)
				merged = true
			}
		}
		if !merged {
			userCart.Items = append(userCart.Items, g)
		}
	}
	userCart.CustomerID = principal(r).UserID
	if err := s.Docs.SaveCart(ctx, userCart); err != nil {
		s.fail(w, r, err)
		return
	}
	if err := s.Docs.DeleteCart(ctx, guestID); err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, withTotals(userCart))
}

type cartView struct {
	domain.Cart
	ItemCount int          `json:"itemCount"`
	Subtotal  domain.Money `json:"subtotal"`
}

func withTotals(c domain.Cart) cartView {
	v := cartView{Cart: c, Subtotal: domain.Money{Currency: c.Currency}}
	for _, it := range c.Items {
		v.ItemCount += it.Quantity
		v.Subtotal.Amount += int64(it.Quantity) * it.UnitPrice.Amount
	}
	return v
}

func quoteLines(c domain.Cart) []clients.QuoteLine {
	lines := make([]clients.QuoteLine, 0, len(c.Items))
	for _, it := range c.Items {
		lines = append(lines, clients.QuoteLine{SKU: it.SKU, Quantity: it.Quantity, UnitPrice: it.UnitPrice.Amount})
	}
	return lines
}
