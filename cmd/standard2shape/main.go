package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/IndependentImpact/standard2shape/internal/tracer"
)

func main() {
	address := flag.String("addr", "127.0.0.1:8090", "local address for the tracer")
	fixture := flag.String("fixture", "fixtures/tracer", "path to the synthetic tracer bundle")
	webDir := flag.String("web-dir", "web/dist", "path to built web assets")
	flag.Parse()

	session, err := tracer.NewSession(*fixture)
	if err != nil {
		log.Fatalf("open tracer fixture: %v", err)
	}
	defer session.Close()

	server := &http.Server{
		Addr:              *address,
		Handler:           tracer.Handler(session, *webDir),
		ReadHeaderTimeout: 5 * time.Second,
	}
	fmt.Fprintf(os.Stdout, "standard2shape tracer: http://%s\n", *address)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("serve tracer: %v", err)
	}
}
