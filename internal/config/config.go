// Package config reads orders-api settings from the environment.
package config

import (
	"os"
	"strings"
)

// Config is everything orders-api needs to reach its dependencies.
type Config struct {
	Addr string

	PostgresDSN string
	MySQLDSN    string
	MongoURI    string
	MongoDB     string
	RedisAddr   string
	KafkaBroker []string

	CatalogURL    string
	PricingURL    string
	AuthURL       string
	FlagsURL      string
	ShippingURL   string // HTTP/2 (h2c)
	PaymentsGRPC  string
	InventoryGRPC string

	Region string
}

// Load reads the environment, with defaults that match deploy/compose.
func Load() Config {
	return Config{
		Addr:          env("ADDR", ":8080"),
		PostgresDSN:   env("POSTGRES_DSN", "postgres://kartly:kartly@localhost:5432/kartly?sslmode=disable"),
		MySQLDSN:      env("MYSQL_DSN", "kartly:kartly@tcp(localhost:3306)/accounts?parseTime=true"),
		MongoURI:      env("MONGO_URI", "mongodb://localhost:27017"),
		MongoDB:       env("MONGO_DB", "kartly"),
		RedisAddr:     env("REDIS_ADDR", "localhost:6379"),
		KafkaBroker:   strings.Split(env("KAFKA_BROKERS", "localhost:9092"), ","),
		CatalogURL:    env("CATALOG_URL", "http://localhost:9101"),
		PricingURL:    env("PRICING_URL", "http://localhost:9102"),
		AuthURL:       env("AUTH_URL", "http://localhost:9103"),
		FlagsURL:      env("FLAGS_URL", "http://localhost:9104"),
		ShippingURL:   env("SHIPPING_URL", "http://localhost:9105"),
		PaymentsGRPC:  env("PAYMENTS_GRPC", "localhost:9201"),
		InventoryGRPC: env("INVENTORY_GRPC", "localhost:9202"),
		Region:        env("REGION", "IN"),
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
