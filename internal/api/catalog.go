package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	kartlyv1 "github.com/ayush3160/kartly-e2e/gen/kartlyv1"
	"github.com/ayush3160/kartly-e2e/internal/domain"
)

const productTTL = 10 * time.Minute

// product reads a product through the Redis cache.
func (s *Server) product(ctx context.Context, id string) (domain.Product, error) {
	key := "product:" + s.Region + ":" + id
	if v, ok, err := s.Cache.Get(ctx, key); err == nil && ok {
		var p domain.Product
		if json.Unmarshal([]byte(v), &p) == nil {
			return p, nil
		}
	}
	p, err := s.Catalog.Product(ctx, id, s.Region)
	if err != nil {
		return p, err
	}
	b, _ := json.Marshal(p)
	_ = s.Cache.Set(ctx, key, string(b), productTTL)
	return p, nil
}

func (s *Server) listProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page := queryInt(r, "page", 1, 50)
	size := queryInt(r, "size", 20, 50)
	res, err := s.Catalog.Search(r.Context(), q.Get("q"), q.Get("category"), s.Region, page, size)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if res.Items == nil {
		res.Items = []domain.Product{}
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) getProduct(w http.ResponseWriter, r *http.Request) {
	p, err := s.product(r.Context(), r.PathValue("id"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	rating, err := s.Docs.ProductRating(r.Context(), p.ID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"product": p, "rating": rating})
}

func (s *Server) getStock(w http.ResponseWriter, r *http.Request) {
	p, err := s.product(r.Context(), r.PathValue("id"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	st, err := s.GRPC.Inventory.Get(r.Context(), &kartlyv1.GetStockRequest{Sku: p.SKU, Warehouse: "BLR-1"})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"sku": st.GetSku(), "available": st.GetAvailable(), "warehouse": st.GetWarehouse(),
		"lowStock": st.GetAvailable() > 0 && st.GetAvailable() < 5,
	})
}
