package main

import (
	"context"
	"strings"

	kartlyv1 "github.com/ayush3160/kartly-e2e/gen/kartlyv1"
)

// payments decides by card token: tok_visa approves, tok_decline declines,
// tok_3ds needs customer action.
type payments struct {
	kartlyv1.UnimplementedPaymentsServer
}

func (payments) Authorize(_ context.Context, r *kartlyv1.AuthorizeRequest) (*kartlyv1.AuthorizeResponse, error) {
	switch r.GetCardToken() {
	case "tok_decline":
		return &kartlyv1.AuthorizeResponse{AuthorizationId: "auth_" + r.GetOrderId(), Status: "declined", DeclineCode: "insufficient_funds"}, nil
	case "tok_3ds":
		return &kartlyv1.AuthorizeResponse{AuthorizationId: "auth_" + r.GetOrderId(), Status: "requires_action"}, nil
	}
	return &kartlyv1.AuthorizeResponse{AuthorizationId: "auth_" + r.GetOrderId(), Status: "approved"}, nil
}

func (payments) Capture(_ context.Context, r *kartlyv1.CaptureRequest) (*kartlyv1.CaptureResponse, error) {
	return &kartlyv1.CaptureResponse{CaptureId: "cap_" + strings.TrimPrefix(r.GetAuthorizationId(), "auth_"), Status: "succeeded"}, nil
}

func (payments) Refund(_ context.Context, r *kartlyv1.RefundRequest) (*kartlyv1.RefundResponse, error) {
	return &kartlyv1.RefundResponse{RefundId: "ref_" + strings.TrimPrefix(r.GetCaptureId(), "cap_"), Status: "succeeded"}, nil
}

// inventory has fixed stock per SKU; SKU-LOW-STOCK has 2 left and
// SKU-CAP-NVY none.
type inventory struct {
	kartlyv1.UnimplementedInventoryServer
}

func stockOf(sku string) int32 {
	switch sku {
	case "SKU-LOW-STOCK":
		return 2
	case "SKU-CAP-NVY":
		return 0
	}
	return 120
}

func (inventory) Get(_ context.Context, r *kartlyv1.GetStockRequest) (*kartlyv1.Stock, error) {
	return &kartlyv1.Stock{Sku: r.GetSku(), Available: stockOf(r.GetSku()), Warehouse: r.GetWarehouse()}, nil
}

func (inventory) Reserve(_ context.Context, r *kartlyv1.ReserveRequest) (*kartlyv1.ReserveResponse, error) {
	resp := &kartlyv1.ReserveResponse{ReservationId: r.GetReservationId(), Status: "reserved"}
	for _, l := range r.GetLines() {
		if l.GetQuantity() > stockOf(l.GetSku()) {
			resp.Short = append(resp.Short, &kartlyv1.ReserveLine{Sku: l.GetSku(), Quantity: l.GetQuantity() - stockOf(l.GetSku())})
		}
	}
	if len(resp.Short) > 0 {
		resp.Status = "partial"
	}
	return resp, nil
}

func (inventory) Release(context.Context, *kartlyv1.ReleaseRequest) (*kartlyv1.ReleaseResponse, error) {
	return &kartlyv1.ReleaseResponse{Status: "released"}, nil
}
