package clients

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/ayush3160/kartly-e2e/internal/domain"
)

// Catalog is catalog-svc.
type Catalog struct{ c jsonClient }

// NewCatalog makes a catalog client.
func NewCatalog(base string) *Catalog { return &Catalog{c: newJSONClient("catalog-svc", base)} }

// Product returns one product priced for a region.
func (k *Catalog) Product(ctx context.Context, id, region string) (domain.Product, error) {
	var p domain.Product
	err := k.c.do(ctx, http.MethodGet, "/products/"+url.PathEscape(id)+"?region="+url.QueryEscape(region), nil, &p, nil)
	return p, err
}

// ProductPage is a page of catalog search results.
type ProductPage struct {
	Items []domain.Product `json:"items"`
	Total int              `json:"total"`
	Page  int              `json:"page"`
}

// Search lists products by query, category and page.
func (k *Catalog) Search(ctx context.Context, q, category, region string, page, size int) (ProductPage, error) {
	v := url.Values{}
	v.Set("q", q)
	v.Set("category", category)
	v.Set("region", region)
	v.Set("page", strconv.Itoa(page))
	v.Set("size", strconv.Itoa(size))
	var out ProductPage
	err := k.c.do(ctx, http.MethodGet, "/products?"+v.Encode(), nil, &out, nil)
	return out, err
}

// Pricing is pricing-svc.
type Pricing struct{ c jsonClient }

// NewPricing makes a pricing client.
func NewPricing(base string) *Pricing { return &Pricing{c: newJSONClient("pricing-svc", base)} }

// QuoteLine is one line priced by pricing-svc.
type QuoteLine struct {
	SKU       string `json:"sku"`
	Quantity  int    `json:"quantity"`
	UnitPrice int64  `json:"unitPrice"`
}

// QuoteRequest asks pricing-svc for a cart total.
type QuoteRequest struct {
	CartID   string      `json:"cartId"`
	Customer string      `json:"customerId"`
	Region   string      `json:"region"`
	Currency string      `json:"currency"`
	Lines    []QuoteLine `json:"lines"`
	Coupon   *CouponRef  `json:"coupon,omitempty"`
	// GiftWrap asks pricing-svc to add the gift-wrap fee (KART-402).
	GiftWrap bool `json:"giftWrap"`
}

// CouponRef is a validated coupon passed to pricing.
type CouponRef struct {
	Code  string `json:"code"`
	Kind  string `json:"kind"`
	Value int64  `json:"value"`
}

// Quote prices a cart.
func (p *Pricing) Quote(ctx context.Context, req QuoteRequest) (domain.Quote, error) {
	var q domain.Quote
	err := p.c.do(ctx, http.MethodPost, "/quote", req, &q, nil)
	return q, err
}

// Auth is auth-svc.
type Auth struct{ c jsonClient }

// NewAuth makes an auth client.
func NewAuth(base string) *Auth { return &Auth{c: newJSONClient("auth-svc", base)} }

// Principal is the caller a token belongs to.
type Principal struct {
	UserID string `json:"userId"`
	Tenant string `json:"tenant"`
	Role   string `json:"role"` // customer | admin | support
	Active bool   `json:"active"`
}

// Introspect validates a bearer token.
func (a *Auth) Introspect(ctx context.Context, token string) (Principal, error) {
	var p Principal
	err := a.c.do(ctx, http.MethodPost, "/introspect", map[string]string{"token": token}, &p, nil)
	return p, err
}

// Perms is what a user may do on a resource.
type Perms struct {
	Allowed []string `json:"allowed"`
}

// Permissions asks auth-svc what userID may do on resource.
func (a *Auth) Permissions(ctx context.Context, userID, resource string) (Perms, error) {
	var p Perms
	err := a.c.do(ctx, http.MethodGet, "/perms/"+url.PathEscape(userID)+"?resource="+url.QueryEscape(resource), nil, &p, nil)
	return p, err
}

// Flags is flags-svc.
type Flags struct{ c jsonClient }

// NewFlags makes a flags client.
func NewFlags(base string) *Flags { return &Flags{c: newJSONClient("flags-svc", base)} }

// Enabled reports whether a flag is on for a tenant.
func (f *Flags) Enabled(ctx context.Context, flag, tenant string) (bool, error) {
	var out struct {
		Enabled bool `json:"enabled"`
	}
	err := f.c.do(ctx, http.MethodGet, "/flags/"+url.PathEscape(flag)+"?tenant="+url.QueryEscape(tenant), nil, &out, nil)
	return out.Enabled, err
}
