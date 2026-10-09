// Command orders-api is Kartly's orders service.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ayush3160/kartly-e2e/internal/api"
	"github.com/ayush3160/kartly-e2e/internal/clients"
	"github.com/ayush3160/kartly-e2e/internal/config"
	"github.com/ayush3160/kartly-e2e/internal/store"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("orders-api stopped", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	orders, err := store.NewOrders(ctx, cfg.PostgresDSN)
	if err != nil {
		return err
	}
	defer orders.Close()
	accounts, err := store.NewAccounts(cfg.MySQLDSN)
	if err != nil {
		return err
	}
	defer accounts.Close()
	docs, err := store.NewDocs(cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		return err
	}
	defer docs.Close(context.Background())
	cache := store.NewCache(cfg.RedisAddr)
	defer cache.Close()
	events := store.NewEvents(cfg.KafkaBroker)
	defer events.Close()
	grpcClients, err := clients.NewGRPC(cfg.PaymentsGRPC, cfg.InventoryGRPC)
	if err != nil {
		return err
	}
	defer grpcClients.Close()

	// Wait for the databases and Kafka: in a fresh namespace they start
	// alongside orders-api.
	if err := waitFor(ctx, log, "postgres", orders.Ping); err != nil {
		return err
	}
	if err := waitFor(ctx, log, "mysql", accounts.Ping); err != nil {
		return err
	}
	if err := waitFor(ctx, log, "kafka", func(ctx context.Context) error {
		return store.EnsureTopics(ctx, cfg.KafkaBroker, "order-events", "email-requests")
	}); err != nil {
		return err
	}

	srv := &api.Server{
		Orders: orders, Accounts: accounts, Docs: docs, Cache: cache, Events: events,
		Catalog: clients.NewCatalog(cfg.CatalogURL), Pricing: clients.NewPricing(cfg.PricingURL),
		Auth: clients.NewAuth(cfg.AuthURL), Flags: clients.NewFlags(cfg.FlagsURL),
		Shipping: clients.NewShipping(cfg.ShippingURL), GRPC: grpcClients,
		Region: cfg.Region, WarehousePIN: "560001", Log: log,
	}
	hs := &http.Server{Addr: cfg.Addr, Handler: srv.Routes(), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = hs.Shutdown(shutdown)
	}()
	log.Info("orders-api listening", "addr", cfg.Addr)
	if err := hs.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// waitFor retries check every two seconds for up to two minutes.
func waitFor(ctx context.Context, log *slog.Logger, name string, check func(context.Context) error) error {
	deadline := time.Now().Add(2 * time.Minute)
	for {
		c, cancel := context.WithTimeout(ctx, 3*time.Second)
		err := check(c)
		cancel()
		if err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%s is not reachable: %w", name, err)
		}
		log.Info("waiting for dependency", "name", name, "err", err.Error())
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}
