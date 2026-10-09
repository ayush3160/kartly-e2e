package main

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

type money struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

type product struct {
	ID       string `json:"id"`
	SKU      string `json:"sku"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Price    money  `json:"price"`
	InStock  bool   `json:"inStock"`
	priceIN  int64
	priceUS  int64
}

// products is the catalog: prices in paise (IN) and cents (US).
var products = []product{
	{ID: "p-1001", SKU: "SKU-TEE-BLK-M", Name: "Classic Tee, Black, M", Category: "apparel", InStock: true, priceIN: 79900, priceUS: 1599},
	{ID: "p-1002", SKU: "SKU-TEE-WHT-L", Name: "Classic Tee, White, L", Category: "apparel", InStock: true, priceIN: 79900, priceUS: 1599},
	{ID: "p-1003", SKU: "SKU-HOOD-GRY-M", Name: "Zip Hoodie, Grey, M", Category: "apparel", InStock: true, priceIN: 249900, priceUS: 4999},
	{ID: "p-1004", SKU: "SKU-JEAN-IND-32", Name: "Slim Jeans, Indigo, 32", Category: "apparel", InStock: true, priceIN: 199900, priceUS: 3999},
	{ID: "p-1005", SKU: "SKU-CAP-NVY", Name: "Baseball Cap, Navy", Category: "accessories", InStock: false, priceIN: 59900, priceUS: 1299},
	{ID: "p-1006", SKU: "SKU-SOCK-3PK", Name: "Ankle Socks, 3-pack", Category: "accessories", InStock: true, priceIN: 39900, priceUS: 899},
	{ID: "p-2001", SKU: "SKU-BUDS-PRO", Name: "Wireless Earbuds Pro", Category: "electronics", InStock: true, priceIN: 899900, priceUS: 12999},
	{ID: "p-2002", SKU: "SKU-CHGR-65W", Name: "65W USB-C Charger", Category: "electronics", InStock: true, priceIN: 249900, priceUS: 3999},
	{ID: "p-2003", SKU: "SKU-CABLE-2M", Name: "Braided USB-C Cable, 2m", Category: "electronics", InStock: true, priceIN: 69900, priceUS: 1499},
	{ID: "p-2004", SKU: "SKU-SPKR-MINI", Name: "Mini Bluetooth Speaker", Category: "electronics", InStock: true, priceIN: 349900, priceUS: 5999},
	{ID: "p-2005", SKU: "SKU-WATCH-FIT", Name: "Fitness Watch", Category: "electronics", InStock: true, priceIN: 1299900, priceUS: 17999},
	{ID: "p-2006", SKU: "SKU-LOW-STOCK", Name: "Limited Edition Keyboard", Category: "electronics", InStock: true, priceIN: 1499900, priceUS: 19999},
	{ID: "p-3001", SKU: "SKU-MUG-CER", Name: "Ceramic Mug, 350ml", Category: "home", InStock: true, priceIN: 49900, priceUS: 999},
	{ID: "p-3002", SKU: "SKU-LAMP-DESK", Name: "LED Desk Lamp", Category: "home", InStock: true, priceIN: 189900, priceUS: 3499},
	{ID: "p-3003", SKU: "SKU-PLANT-POT", Name: "Terracotta Planter", Category: "home", InStock: true, priceIN: 89900, priceUS: 1799},
	{ID: "p-3004", SKU: "SKU-THROW-BLK", Name: "Knit Throw Blanket", Category: "home", InStock: true, priceIN: 299900, priceUS: 5499},
	{ID: "p-4001", SKU: "SKU-BOOK-GO", Name: "Learning Go, 2nd ed.", Category: "books", InStock: true, priceIN: 349900, priceUS: 4499},
	{ID: "p-4002", SKU: "SKU-BOOK-DDIA", Name: "Designing Data-Intensive Applications", Category: "books", InStock: true, priceIN: 399900, priceUS: 4999},
	{ID: "p-4003", SKU: "SKU-NOTE-A5", Name: "Dot-grid Notebook, A5", Category: "books", InStock: true, priceIN: 29900, priceUS: 799},
	{ID: "p-4004", SKU: "SKU-PEN-SET", Name: "Gel Pen Set", Category: "books", InStock: true, priceIN: 19900, priceUS: 599},
}

func priced(p product, region string) product {
	if region == "US" {
		p.Price = money{Amount: p.priceUS, Currency: "USD"}
	} else {
		p.Price = money{Amount: p.priceIN, Currency: "INR"}
	}
	return p
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func catalogMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /products/{id}", func(w http.ResponseWriter, r *http.Request) {
		for _, p := range products {
			if p.ID == r.PathValue("id") {
				writeJSON(w, 200, priced(p, r.URL.Query().Get("region")))
				return
			}
		}
		writeJSON(w, 404, map[string]string{"error": "product not found"})
	})
	mux.HandleFunc("GET /products", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		term, cat, region := strings.ToLower(q.Get("q")), q.Get("category"), q.Get("region")
		page, _ := strconv.Atoi(q.Get("page"))
		size, _ := strconv.Atoi(q.Get("size"))
		if page < 1 {
			page = 1
		}
		if size < 1 {
			size = 20
		}
		var hits []product
		for _, p := range products {
			if cat != "" && p.Category != cat {
				continue
			}
			if term != "" && !strings.Contains(strings.ToLower(p.Name), term) {
				continue
			}
			hits = append(hits, priced(p, region))
		}
		sort.SliceStable(hits, func(i, j int) bool { return hits[i].ID < hits[j].ID })
		total := len(hits)
		start := min((page-1)*size, total)
		end := min(start+size, total)
		page_ := hits[start:end]
		if page_ == nil {
			page_ = []product{}
		}
		writeJSON(w, 200, map[string]any{"items": page_, "total": total, "page": page})
	})
	return mux
}

type quoteLine struct {
	SKU       string `json:"sku"`
	Quantity  int    `json:"quantity"`
	UnitPrice int64  `json:"unitPrice"`
}

type quoteReq struct {
	CartID   string      `json:"cartId"`
	Customer string      `json:"customerId"`
	Region   string      `json:"region"`
	Currency string      `json:"currency"`
	Lines    []quoteLine `json:"lines"`
	GiftWrap bool        `json:"giftWrap"`
	Coupon   *struct {
		Code  string `json:"code"`
		Kind  string `json:"kind"`
		Value int64  `json:"value"`
	} `json:"coupon,omitempty"`
}

// pricingMux prices a cart: 18% GST (IN) or 8% sales tax (US) on the
// discounted subtotal, free shipping above 99900 paise / 5000 cents.
func pricingMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /quote", func(w http.ResponseWriter, r *http.Request) {
		var q quoteReq
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields() // a field pricing-svc does not know is a contract break
		if err := dec.Decode(&q); err != nil {
			writeJSON(w, 400, map[string]string{"error": "bad quote request: " + err.Error()})
			return
		}
		var sub int64
		for _, l := range q.Lines {
			sub += int64(l.Quantity) * l.UnitPrice
		}
		freeAbove, ship, taxPct := int64(99900), int64(4900), int64(18)
		if q.Region == "US" {
			freeAbove, ship, taxPct = 5000, 599, 8
		}
		if sub >= freeAbove {
			ship = 0
		}
		var disc int64
		code := ""
		if q.Coupon != nil {
			code = q.Coupon.Code
			switch q.Coupon.Kind {
			case "percent":
				disc = sub * q.Coupon.Value / 100
			case "fixed":
				disc = min(q.Coupon.Value, sub)
			case "free_shipping":
				ship = 0
			}
		}
		var wrap int64
		if q.GiftWrap {
			wrap = 4900
			if q.Region == "US" {
				wrap = 499
			}
		}
		tax := (sub - disc + wrap) * taxPct / 100
		m := func(a int64) money { return money{Amount: a, Currency: q.Currency} }
		writeJSON(w, 200, map[string]any{
			"subtotal": m(sub), "discount": m(disc), "shipping": m(ship), "tax": m(tax),
			"giftWrapFee": m(wrap), "total": m(sub - disc + ship + wrap + tax), "coupon": code,
		})
	})
	return mux
}

type principal struct {
	UserID string `json:"userId"`
	Tenant string `json:"tenant"`
	Role   string `json:"role"`
	Active bool   `json:"active"`
}

// tokens maps bearer tokens to users. tok-expired is a revoked session.
var tokens = map[string]principal{
	"tok-u101":    {"u-101", "acme", "customer", true},
	"tok-u102":    {"u-102", "acme", "customer", true},
	"tok-u103":    {"u-103", "acme", "customer", true},
	"tok-u104":    {"u-104", "acme", "customer", true},
	"tok-support": {"u-900", "acme", "support", true},
	"tok-admin":   {"u-901", "acme", "admin", true},
	"tok-expired": {"u-101", "acme", "customer", false},
}

func authMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /introspect", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Token string `json:"token"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		p, ok := tokens[req.Token]
		if !ok {
			writeJSON(w, 200, principal{Active: false})
			return
		}
		writeJSON(w, 200, p)
	})
	mux.HandleFunc("GET /perms/{user}", func(w http.ResponseWriter, r *http.Request) {
		allowed := []string{}
		switch r.PathValue("user") {
		case "u-900": // support
			allowed = []string{"read", "read_notes"}
		case "u-901": // admin
			allowed = []string{"read", "read_notes", "write"}
		}
		writeJSON(w, 200, map[string]any{"allowed": allowed, "resource": r.URL.Query().Get("resource")})
	})
	return mux
}

var flags = map[string]bool{"admin-status-audit": true, "express-shipping": false}

func flagsMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /flags/{name}", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"flag": r.PathValue("name"), "enabled": flags[r.PathValue("name")]})
	})
	return mux
}

// shippingMux is the partner API (served over h2c).
func shippingMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v2/rates", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			FromPincode string `json:"fromPincode"`
			ToPincode   string `json:"toPincode"`
			WeightGrams int    `json:"weightGrams"`
		}
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil || len(req.ToPincode) != 6 {
			writeJSON(w, 400, map[string]string{"error": "invalid rate request"})
			return
		}
		base := int64(4900 + req.WeightGrams/500*1000)
		days := 4
		if req.ToPincode[0] == req.FromPincode[0] {
			days = 2
		}
		writeJSON(w, 200, map[string]any{"rates": []map[string]any{
			{"service": "standard", "amount": base, "currency": "INR", "days": days},
			{"service": "express", "amount": base * 2, "currency": "INR", "days": 1},
		}})
	})
	mux.HandleFunc("POST /v2/labels", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Reference string `json:"reference"`
			Kind      string `json:"kind"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		writeJSON(w, 201, map[string]any{"tracking": "KRT" + strings.ToUpper(strings.ReplaceAll(req.Reference, "_", "")), "kind": req.Kind})
	})
	return mux
}
