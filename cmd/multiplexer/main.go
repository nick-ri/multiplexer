package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/NickRI/multiplexer/collector"

	"github.com/NickRI/multiplexer/api"

	"github.com/NickRI/multiplexer/limit"

	"github.com/NickRI/multiplexer/transport"
)

func main() {
	limitN := flag.Int64("limit", 100, "number of parallel actions can be processed")
	address := flag.String("address", ":8080", "listen server address")
	collectTmt := flag.Duration("timeout", time.Second, "timeout per each resource collection")

	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())

	coll := collector.NewCollector(4*int(*limitN), *collectTmt)
	coll.Start(ctx)

	srv := transport.NewServer(*address)

	srv.Post("/collect",
		http.HandlerFunc(api.Collect(coll)),
		api.RateLimitMiddleware(limit.NewLimiter(time.Second, *limitN)),
		api.LoggerMiddleware,
	)

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT)

	go func() {
		for range c {
			log.Println("Shutting down server...")

			if err := srv.Shutdown(context.Background()); err != nil {
				log.Fatal(err)
			}

			cancel()
		}
	}()

	log.Printf("limiter server starting on %s", *address)
	if err := srv.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatal("server: ", err)
	}

}
