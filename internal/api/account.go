package api

import (
	"net/http"
	"regexp"

	"github.com/ayush3160/kartly-e2e/internal/domain"
	"github.com/ayush3160/kartly-e2e/internal/store"
)

var pincodeRE = regexp.MustCompile(`^[1-9][0-9]{5}$`)

func (s *Server) getMe(w http.ResponseWriter, r *http.Request) {
	acc, err := s.Accounts.Account(r.Context(), principal(r).UserID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, acc)
}

func (s *Server) putAddress(w http.ResponseWriter, r *http.Request) {
	var ad domain.Address
	if err := decode(r, &ad); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "body must be an address")
		return
	}
	switch {
	case ad.ID == 0:
		writeError(w, http.StatusBadRequest, "invalid_address", "id is required")
		return
	case ad.Line1 == "" || ad.City == "":
		writeError(w, http.StatusBadRequest, "invalid_address", "line1 and city are required")
		return
	case !pincodeRE.MatchString(ad.Pincode):
		writeError(w, http.StatusBadRequest, "invalid_pincode", "pincode must be 6 digits")
		return
	}
	p := principal(r)
	if err := s.Accounts.SetDefaultAddress(r.Context(), p.UserID, ad); err != nil {
		s.fail(w, r, err)
		return
	}
	acc, err := s.Accounts.Account(r.Context(), p.UserID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, acc)
}

func (s *Server) myOrders(w http.ResponseWriter, r *http.Request) {
	p := principal(r)
	orders, err := s.Orders.ListOrders(r.Context(), store.ListFilter{
		Tenant: p.Tenant, CustomerID: p.UserID, Limit: 5,
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"orders": orders, "count": len(orders)})
}
