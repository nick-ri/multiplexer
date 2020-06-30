package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/NickRI/multiplexer/limit"
	"github.com/NickRI/multiplexer/transport"

	"github.com/NickRI/multiplexer/api"
)

func main() {
	limitN := flag.Int64("limit", 100, "number of parallel actions can be processed")
	address := flag.String("address", ":8080", "listen server address")

	flag.Parse()

	limiter := limit.NewLimiter(time.Second, *limitN)

	srv := transport.NewServer(*address)

	srv.Get("/limit", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "limit\n")
	}), api.RateLimitMiddleware(limiter))

	log.Printf("server starting on %s", *address)
	if err := srv.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatal("server: ", err)
	}
}
