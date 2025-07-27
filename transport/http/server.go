package http

import (
	"context"
	"net/http"
	"reflect"

	"github.com/mangohow/gowlb/errors"
	"github.com/mangohow/gowlb/serialize"
	"github.com/mangohow/gowlb/transport/binding"
	"github.com/sirupsen/logrus"
)

const (
	ParamKey = "param"
	FormKey  = "form"

	ctxKey = "ctx-key"
)

type Server struct {
	server *http.Server
	router *routeWrapper
	addr   string

	log            *logrus.Logger
	errorEncoder   EncodeErrorFunc
	queryBinding   binding.Binding
	formBinding    binding.Binding
	pathVarBinding binding.Binding
	bodyBinding    binding.Binding

	resultEncoder EncodeResultFunc

	middlewares []Middleware

	ctx context.Context
}

// EncodeErrorFunc 错误处理函数
type EncodeErrorFunc func(ctx *Context, err error)

// DefaultEncodeErrorFunc 默认错误处理函数
func DefaultEncodeErrorFunc(ctx *Context, err error) {
	e, ok := err.(errors.Error)
	if !ok {
		e = errors.FromError(errors.UnknownCode, errors.DefaultStatus, errors.UnknownReason, errors.UnknownMessage, err)
	}

	err = ctx.JSON(int(e.HttpStatus()), serialize.Response{
		Error: e,
	})
	if err != nil {
		ctx.WriteStatus(http.StatusInternalServerError)
	}

	return
}

type EncodeResultFunc func(ctx *Context, arg any)

type Option func(s *Server)

func WithAddr(addr string) Option {
	return func(s *Server) {
		if addr == "" {
			addr = ":8080"
		}
		s.addr = addr
	}
}

func WithEncodeErrorFunc(fn EncodeErrorFunc) Option {
	return func(s *Server) {
		s.errorEncoder = fn
	}
}

func WithQueryBinding(bind binding.Binding) Option {
	return func(s *Server) {
		s.queryBinding = bind
	}
}

func WithFormBinding(bind binding.Binding) Option {
	return func(s *Server) {
		s.formBinding = bind
	}
}

func WithPathVarBinding(bind binding.Binding) Option {
	return func(s *Server) {
		s.pathVarBinding = bind
	}
}

func WithBodyBinding(bind binding.Binding) Option {
	return func(s *Server) {
		s.bodyBinding = bind
	}
}

func WithLogger(log *logrus.Logger) Option {
	return func(s *Server) {
		s.log = log
	}
}

func WithContext(ctx context.Context) Option {
	return func(s *Server) {
		s.ctx = ctx
	}
}

func New(opts ...Option) *Server {
	s := &Server{}
	for _, opt := range opts {
		opt(s)
	}

	s.server = &http.Server{
		Handler: s.router,
	}

	if s.queryBinding == nil {
		s.queryBinding = binding.QueryBinding{Tag: "json"}
	}

	if s.formBinding == nil {
		s.formBinding = binding.FormBinding{}
	}

	if s.pathVarBinding == nil {
		s.pathVarBinding = binding.PathVarBinding{}
	}

	if s.bodyBinding == nil {
		s.bodyBinding = binding.JsonBinding{}
	}

	if s.errorEncoder == nil {
		s.errorEncoder = DefaultEncodeErrorFunc
	}

	if s.router == nil {
		s.router = newRouterWrapper(s.errorEncoder, s)
	}

	if s.log == nil {
		s.log = logrus.StandardLogger()
	}

	if s.ctx == nil {
		s.ctx = context.Background()
	}

	if s.addr == "" {
		s.addr = ":8000"
	}
	s.server.Addr = s.addr

	if s.log == nil {
		s.log = logrus.StandardLogger()
	}

	return s
}

func (s *Server) HttpServer() *http.Server {
	return s.server
}

func (s *Server) RegisterService(sd *ServiceDesc, srv interface{}) {
	if srv != nil {
		ht := reflect.TypeOf(sd.HandlerType).Elem()
		st := reflect.TypeOf(srv)
		if !st.Implements(ht) {
			s.log.Fatalf("handler type %v not implement %v", st, ht)
		}
	}

	s.register(sd, srv)
}

func (s *Server) register(sd *ServiceDesc, srv interface{}) {
	for _, d := range sd.Methods {
		handler := d.Handler
		s.handle(d.Method, d.Path, func(ctx context.Context, req interface{}) (resp interface{}, err error) {
			return handler(ctx, srv, chainHandler(s.middlewares))
		})
	}
}

func chainHandler(middlewares []Middleware) Middleware {
	if len(middlewares) == 0 {
		return func(ctx context.Context, req any, handler Handler) (any, error) {
			return handler(ctx, req)
		}
	}

	return func(ctx context.Context, req interface{}, handler Handler) (interface{}, error) {
		return middlewares[0](ctx, req, getChainMiddleware(middlewares, 0, handler))
	}
}

func getChainMiddleware(middlewares []Middleware, cur int, handler Handler) Handler {
	if cur >= len(middlewares)-1 {
		return handler
	}

	return func(ctx context.Context, req interface{}) (interface{}, error) {
		return middlewares[cur+1](ctx, req, getChainMiddleware(middlewares, cur+1, handler))
	}
}

func (s *Server) handle(method, relativePath string, handler Handler) {
	s.router.HandleFunc(method, relativePath, s.handlerConvert(handler))
}

func (s *Server) handlerConvert(handler Handler) HandlerFunc {
	return func(c *Context) error {
		ctx := context.WithValue(s.ctx, ctxKey, c)
		resp, err := handler(ctx, nil)
		if err != nil {
			return err
		}

		if s.resultEncoder != nil {
			s.resultEncoder(c, resp)
		}

		return nil
	}
}

func (s *Server) Middleware(middleware ...Middleware) {
	s.middlewares = append(s.middlewares, middleware...)
}

func (s *Server) Start() error {
	s.log.Info("server listen at ", s.addr)
	err := s.server.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}

	return err
}

func (s *Server) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
