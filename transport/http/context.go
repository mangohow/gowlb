package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mangohow/gowlb/tools/sync"
	"github.com/mangohow/gowlb/transport/binding"
)

var (
	pool = sync.Pool[*Context]{}
)

type Context struct {
	w   http.ResponseWriter
	req *http.Request
	s   *Server
}

func newContext(w http.ResponseWriter, r *http.Request, s *Server) *Context {
	c := pool.Get()
	c.w = w
	c.req = r
	c.s = s

	return c
}

func putContext(c *Context) {
	c.req = nil
	c.w = nil
	c.s = nil
	pool.Put(c)
}

func (c *Context) Request() *http.Request {
	return c.req
}

func (c *Context) ResponseWriter() http.ResponseWriter {
	return c.w
}

func (c *Context) SetHeader(key string, value string) {
	c.w.Header().Set(key, value)
}

func (c *Context) WriteContentType(contentType string) {
	c.w.Header().Set("Content-Type", contentType)
}

func (c *Context) GetContentType() string {
	return c.req.Header.Get("content-type")
}

func (c *Context) Logger() {

}

// 绑定路径中的查询参数 /api/xxx?key1=aaa&key2=bbb
func (c *Context) BindQuery(obj any) error {
	return c.s.queryBinding.Bind(c.req, obj)
}

// 绑定form表单参数 x-www-form-urlencoded
func (c *Context) BindForm(obj any) error {
	contentType := c.req.Header.Get("Content-Type")
	switch contentType {
	case "application/x-www-form-urlencoded":
		return c.BindForm(obj)
	case "application/json":
		return c.s.bodyBinding.Bind(c.req, obj)
	case "":
		return fmt.Errorf("missing Content-Type header")
	}

	_, name, found := strings.Cut(contentType, "/")
	if !found {
		name = contentType
	}
	b := binding.GetBinding(name)
	if b == nil {
		return fmt.Errorf("unsupported Content-Type: %s", contentType)
	}

	return b.Bind(c.req, obj)
}

// 绑定body中的JSON参数
func (c *Context) BindJSON(obj any) error {
	return c.s.bodyBinding.Bind(c.req, obj)
}

// 绑定路径参数 /api/user/{id}
func (c *Context) BindPathVar(obj any) error {
	return c.s.pathVarBinding.Bind(c.req, obj)
}

func (c *Context) String(status int, content string) error {
	c.w.WriteHeader(status)
	c.w.Header().Set("Content-Type", "text/plain")
	_, err := fmt.Fprintf(c.w, content)

	return err
}

func (c *Context) JSON(status int, obj any) error {
	c.w.WriteHeader(status)
	c.w.Header().Add("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(c.w).Encode(obj)
}

func (c *Context) WriteStatus(status int) {
	c.w.WriteHeader(status)
}

func FromContext(ctx context.Context) *Context {
	return ctx.Value(ctxKey).(*Context)
}
