package http

import (
	"net/http"

	"github.com/gorilla/mux"
)

type routeWrapper struct {
	mu           *mux.Router
	s            *Server
	errorEncoder EncodeErrorFunc
}

func newRouterWrapper(errorEncoder EncodeErrorFunc, s *Server) *routeWrapper {
	return &routeWrapper{
		mu:           mux.NewRouter(),
		s:            s,
		errorEncoder: errorEncoder,
	}
}

type HandlerFunc func(ctx *Context) error

func (r *routeWrapper) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mu.ServeHTTP(w, req)
}

func (r *routeWrapper) HandleFunc(method string, path string, handler HandlerFunc) {
	r.mu.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
		ctx := newContext(w, req, r.s)
		defer putContext(ctx)
		if err := handler(ctx); err != nil {
			r.errorEncoder(ctx, err)
		}

	}).Methods(method)
}

func (r *routeWrapper) GET(path string, handler HandlerFunc) {
	r.HandleFunc(http.MethodGet, path, handler)
}

func (r *routeWrapper) POST(path string, handler HandlerFunc) {
	r.HandleFunc(http.MethodPost, path, handler)
}

func (r *routeWrapper) PUT(path string, handler HandlerFunc) {
	r.HandleFunc(http.MethodPut, path, handler)
}

func (r *routeWrapper) PATCH(path string, handler HandlerFunc) {
	r.HandleFunc(http.MethodPatch, path, handler)
}

func (r *routeWrapper) DELETE(path string, handler HandlerFunc) {
	r.HandleFunc(http.MethodDelete, path, handler)
}
