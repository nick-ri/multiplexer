package collector

import (
	"context"
	"errors"
	"net/http"
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
	poolSize  int
	client    *http.Client
}

func NewCollector(poolSize int, timeout time.Duration) Collector {
	return &collector{
		poolSize:  poolSize,
		workersCh: make(chan chan param, 1),
		client: &http.Client{
			Transport: http.DefaultTransport,
			Timeout:   timeout,
		},
	}
}

func (c *collector) Start(ctx context.Context) {
	for i := 1; i <= c.poolSize; i++ {
		go c.worker(i, c.workersCh)
	}

	go func() {
		select {
		case <-ctx.Done():
			close(c.workersCh)
		}
	}()
}

func (c *collector) acquireWorkers(count, buffSize int) (chan param, error) {
	if count > c.poolSize {
		return nil, errors.New("acquired workers more that pool size")
	}
	var ch = make(chan param, buffSize)
	for i := 0; i < count; i++ {
		c.workersCh <- ch
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
