package clients

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/http2"
)

// Shipping is the external shipping partner, spoken to over HTTP/2 (h2c in
// the cluster, TLS in production).
type Shipping struct{ c jsonClient }

// NewShipping makes an HTTP/2 shipping client.
func NewShipping(base string) *Shipping {
	c := newJSONClient("shipping-partner", base)
	c.hc = &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, network, addr)
			},
		},
	}
	return &Shipping{c: c}
}

// RateRequest asks for shipping rates.
type RateRequest struct {
	FromPincode string `json:"fromPincode"`
	ToPincode   string `json:"toPincode"`
	WeightGrams int    `json:"weightGrams"`
	// ServiceLevel is "standard", or "express" where express delivery is
	// rolled out (KART-433).
	ServiceLevel string `json:"serviceLevel"`
}

// Rate is one shipping option.
type Rate struct {
	Service  string `json:"service"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
	Days     int    `json:"days"`
}

// Rates returns shipping options for a parcel.
func (s *Shipping) Rates(ctx context.Context, req RateRequest) ([]Rate, error) {
	var out struct {
		Rates []Rate `json:"rates"`
	}
	err := s.c.do(ctx, http.MethodPost, "/v2/rates", req, &out, nil)
	return out.Rates, err
}

// LabelRequest asks for a return or shipment label.
type LabelRequest struct {
	Reference   string `json:"reference"`
	FromPincode string `json:"fromPincode"`
	ToPincode   string `json:"toPincode"`
	Kind        string `json:"kind"` // shipment | return
}

// Label creates a shipping label and returns its tracking number.
func (s *Shipping) Label(ctx context.Context, req LabelRequest) (string, error) {
	var out struct {
		Tracking string `json:"tracking"`
	}
	err := s.c.do(ctx, http.MethodPost, "/v2/labels", req, &out, nil)
	return out.Tracking, err
}
