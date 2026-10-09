package clients

import (
	kartlyv1 "github.com/ayush3160/kartly-e2e/gen/kartlyv1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// GRPC holds the payments and inventory clients.
type GRPC struct {
	Payments  kartlyv1.PaymentsClient
	Inventory kartlyv1.InventoryClient
	conns     []*grpc.ClientConn
}

// NewGRPC dials payments and inventory.
func NewGRPC(paymentsAddr, inventoryAddr string) (*GRPC, error) {
	pc, err := grpc.NewClient(paymentsAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	ic, err := grpc.NewClient(inventoryAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		_ = pc.Close()
		return nil, err
	}
	return &GRPC{
		Payments:  kartlyv1.NewPaymentsClient(pc),
		Inventory: kartlyv1.NewInventoryClient(ic),
		conns:     []*grpc.ClientConn{pc, ic},
	}, nil
}

// Close closes the connections.
func (g *GRPC) Close() {
	for _, c := range g.conns {
		_ = c.Close()
	}
}
