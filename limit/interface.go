package limit

type Limiter interface {
	Allow() bool
}
