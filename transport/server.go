package transport

import (
	"net/http"
	"time"
)

type MiddlewareFunc func(http.Handler) http.Handler

type server struct {
	*http.Server
	handlers map[string]map[string]http.Handler
}

func NewServer(address string) Server {

	handlers := make(map[string]map[string]http.Handler)

	return &server{
		Server: &http.Server{
			Addr: address,

			ReadTimeout:  time.Second,
			WriteTimeout: time.Second * 9,

			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				byMethod, ok := handlers[r.Method]
				if !ok {
					http.NotFound(w, r)
					return
				}

				handler, ok := byMethod[r.URL.String()]
				if !ok {
					http.NotFound(w, r)
					return
				}

				handler.ServeHTTP(w, r)
			}),
		},
		handlers: handlers,
	}
}

func (s *server) Start() error {
	return s.Server.ListenAndServe()
}

func (s *server) Handle(method, pattern string, handler http.Handler, middleware ...MiddlewareFunc) {

	if _, ok := s.handlers[method]; !ok {
		s.handlers[method] = make(map[string]http.Handler)
	}

	for i := len(middleware) - 1; i >= 0; i-- { //make chain call handler with all middleware on-board
		handler = middleware[i](handler)
	}

	s.handlers[method][pattern] = handler
}

func (s *server) Get(pattern string, handler http.Handler, middleware ...MiddlewareFunc) {
	s.Handle(http.MethodGet, pattern, handler, middleware...)
}

func (s *server) Post(pattern string, handler http.Handler, middleware ...MiddlewareFunc) {
	s.Handle(http.MethodPost, pattern, handler, middleware...)
}

func (s *server) Put(pattern string, handler http.Handler, middleware ...MiddlewareFunc) {
	s.Handle(http.MethodPut, pattern, handler, middleware...)
}

func (s *server) Patch(pattern string, handler http.Handler, middleware ...MiddlewareFunc) {
	s.Handle(http.MethodPatch, pattern, handler, middleware...)
}

func (s *server) Delete(pattern string, handler http.Handler, middleware ...MiddlewareFunc) {
	s.Handle(http.MethodDelete, pattern, handler, middleware...)
}
