package http

import (
	"context"
)

type Handler func(ctx context.Context, req any) (resp any, err error)

type Middleware func(ctx context.Context, req any, handler Handler) (any, error)

type methodHandler func(ctx context.Context, srv any, middleware Middleware) (any, error)

type ServiceDesc struct {
	HandlerType interface{}
	Methods     []MethodDesc
}

type MethodDesc struct {
	Method  string
	Path    string
	Handler methodHandler
}
