package collector

import (
	"context"
	"errors"
	"log"
	"net/http"
	"sync"
	"time"
)

type res struct {
	Url  string
	Body string
}

type param struct {
	url   string
	ctx   context.Context
	resCh chan res
	errCh chan error
}

type collector struct {
	sync.RWMutex
	workersCh chan chan param
	fixed     int
	overflow  int
	spawned   int
	closed    bool
	client    *http.Client
}

func NewCollector(fixed, overflow int, timeout time.Duration) Collector {
	return &collector{
		fixed:     fixed,
		overflow:  overflow,
		workersCh: make(chan chan param, fixed),
		client: &http.Client{
			Transport: http.DefaultTransport,
			Timeout:   timeout,
		},
	}
}

func (c *collector) Start(ctx context.Context) {
	defer log.Println("workers pool was started")

	for i := 1; i <= c.fixed; i++ {
		go c.fixedWorker(i, c.workersCh)
	}

	go func() {
		select {
		case <-ctx.Done():
			c.stop()
		}
	}()
}

func (c *collector) stop() {
	defer log.Println("workers pool was stopped")
	c.Lock()
	c.closed = true
	c.Unlock()
	close(c.workersCh)
}

func (c *collector) isStopped() bool {
	c.RLock()
	defer c.RUnlock()
	return c.closed
}

func (c *collector) getSpawnedCount() int {
	c.RLock()
	defer c.RUnlock()
	return c.spawned
}

func (c *collector) incSpawnedCount(n int) int {
	c.Lock()
	defer c.Unlock()
	c.spawned += n
	return c.spawned
}

func (c *collector) acquireWorkers(count, buffSize int) (chan param, error) {
	if c.isStopped() {
		return nil, errors.New("can't acquire workers from stopped pool")
	}

	if count > c.fixed+c.overflow {
		return nil, errors.New("acquired workers more that pool size")
	}

	if c.overflow > 0 && c.getSpawnedCount() >= c.overflow {
		return nil, errors.New("pool is full, can't spawn more workers")
	}

	var ch = make(chan param, buffSize)
	for i := 0; i < count; i++ {
		select {
		case c.workersCh <- ch:
		default:
			id := c.incSpawnedCount(1) + c.fixed
			log.Printf("spawn overflow worker id:%d", id)
			go c.reader(id, ch, false)
		}
	}
	return ch, nil
}

func (c *collector) Collect(ctx context.Context, urls []string, limit int) ([]res, error) {
	var data = make([]res, 0, len(urls))

	var errCh = make(chan error, limit)
	var resCh = make(chan res, len(urls))

	paramsCh, err := c.acquireWorkers(limit, len(urls))
	if err != nil {
		return nil, err
	}
	defer close(paramsCh)

	innerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, url := range urls {
		paramsCh <- param{ctx: innerCtx, url: url, errCh: errCh, resCh: resCh}
	}

	for {
		select {
		case err := <-errCh:
			return nil, err
		case <-ctx.Done():
			return nil, ctx.Err()
		case result := <-resCh:
			data = append(data, result)

			if len(data) == len(urls) {
				return data, nil
			}
		}
	}
}
