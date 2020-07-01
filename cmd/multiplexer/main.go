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

const (
	incomingLimit        = 100         // number of incoming requests that can be processed in second
	outgoingLimit        = 4           // number of outbound requests per second per collection
	maxCountOfUrls       = 20          // maximum number of incoming urls
	maxCollectionTmt     = time.Second // timeout per each resource collection
	fixedWorkersCount    = incomingLimit * outgoingLimit
	overflowWorkersCount = fixedWorkersCount*(maxCountOfUrls/outgoingLimit) - fixedWorkersCount
)

func main() {
	address := flag.String("address", ":8080", "listen server address")

	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())

	coll := collector.NewCollector(fixedWorkersCount, overflowWorkersCount, maxCollectionTmt)
	coll.Start(ctx)

	srv := transport.NewServer(*address)

	srv.Post("/collect",
		http.HandlerFunc(api.Collect(coll, maxCountOfUrls, outgoingLimit)),
		api.RateLimitMiddleware(limit.NewLimiter(time.Second, incomingLimit)),
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
