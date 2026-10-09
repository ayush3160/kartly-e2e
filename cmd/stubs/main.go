// Command stubs runs Kartly's downstream services with fixed, deterministic
// data: catalog, pricing, auth and flags over HTTP, the shipping partner over
// HTTP/2 (h2c), and payments and inventory over gRPC. One binary, one service
// per process: stubs -svc catalog.
package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"time"

	kartlyv1 "github.com/ayush3160/kartly-e2e/gen/kartlyv1"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
)

func main() {
	svc := flag.String("svc", "", "catalog | pricing | auth | flags | shipping | payments | inventory")
	addr := flag.String("addr", ":9000", "listen address")
	flag.Parse()

	switch *svc {
	case "catalog":
		serveHTTP(*addr, catalogMux())
	case "pricing":
		serveHTTP(*addr, pricingMux())
	case "auth":
		serveHTTP(*addr, authMux())
	case "flags":
		serveHTTP(*addr, flagsMux())
	case "shipping":
		h := h2c.NewHandler(shippingMux(), &http2.Server{})
		serveHTTP(*addr, h)
	case "payments", "inventory":
		lis, err := net.Listen("tcp", *addr)
		if err != nil {
			log.Fatal(err)
		}
		gs := grpc.NewServer()
		if *svc == "payments" {
			kartlyv1.RegisterPaymentsServer(gs, payments{})
		} else {
			kartlyv1.RegisterInventoryServer(gs, inventory{})
		}
		log.Printf("%s (gRPC) on %s", *svc, *addr)
		log.Fatal(gs.Serve(lis))
	default:
		log.Fatalf("unknown -svc %q", *svc)
	}
}

func serveHTTP(addr string, h http.Handler) {
	log.Printf("listening on %s", addr)
	s := &http.Server{Addr: addr, Handler: h, ReadHeaderTimeout: 5 * time.Second}
	log.Fatal(s.ListenAndServe())
}
