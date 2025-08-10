package gmcp

import (
	"fmt"
	"github.com/mangohow/gowlb/tools/collection"
)

type ToolHandler interface {
	ToolName() string
	Description() string
	Handle(ctx *Context) error
}

type Router struct {
	tools collection.ConcurrentMap[string, ToolHandler]
}

func (r *Router) ServeTool(toolName string, message []byte) error {
	handler, ok := r.tools.Get(toolName)
	if !ok {
		return fmt.Errorf("no such tool: %s", toolName)
	}

	ctx := newContext()

	return handler.Handle(ctx)
}
