package collector

import "context"

type Collector interface {
	Start(ctx context.Context)
	Collect(ctx context.Context, urls []string, limit int) ([]res, error)
}
