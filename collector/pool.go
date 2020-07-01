package collector

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
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
	workersCh chan chan param
	fixed     int
	overflow  int
	spawned   int32
	client    *http.Client
}

func NewCollector(fixed, overflow int, timeout time.Duration) Collector {
	return &collector{
		fixed:     fixed,
		overflow:  overflow,
		workersCh: make(chan chan param, fixed),
		client: &http.Client{
			Transport: http.DefaultTransport,
			Timeout:   timeout * 3,
		},
	}
}

func (c *collector) Start(ctx context.Context) {
	for i := 1; i <= c.fixed; i++ {
		go c.fixedWorker(i, c.workersCh)
	}

	go func() {
		select {
		case <-ctx.Done():
			close(c.workersCh)
		}
	}()
}

func (c *collector) acquireWorkers(count, buffSize int) (chan param, error) {
	if count > c.fixed+c.overflow {
		return nil, errors.New("acquired workers more that pool size")
	}

	spawned := int(atomic.LoadInt32(&c.spawned))

	if spawned >= c.overflow {
		return nil, errors.New("pool is overflowed, can't spawn more workers")
	}

	var ch = make(chan param, buffSize)
	for i := 0; i < count; i++ {
		select {
		case c.workersCh <- ch:
		default:
			go c.reader(int(atomic.AddInt32(&c.spawned, 1)), ch, false)
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
