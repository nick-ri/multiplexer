package transport

import (
	"context"
	"net/http"
)

type Server interface {
	Start() error
	Shutdown(ctx context.Context) error

	Handle(method, pattern string, handler http.Handler, middleware ...MiddlewareFunc)

	Get(pattern string, handler http.Handler, middleware ...MiddlewareFunc)
	Post(pattern string, handler http.Handler, middleware ...MiddlewareFunc)
	Put(pattern string, handler http.Handler, middleware ...MiddlewareFunc)
	Patch(pattern string, handler http.Handler, middleware ...MiddlewareFunc)
	Delete(pattern string, handler http.Handler, middleware ...MiddlewareFunc)
}
