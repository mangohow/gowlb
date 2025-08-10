package gmcp

import "github.com/mangohow/gowlb/tools/sync"

var (
	ctxPool *sync.Pool[*Context] = sync.NewPool[*Context](func() *Context {
		return &Context{}
	})
)

type Context struct {
}

func newContext() *Context {
	return ctxPool.Get()
}

func putContext(ctx *Context) {
	ctxPool.Put(ctx)
}
