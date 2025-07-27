package binding

import (
	"net/http"
)

type PathVarBinding struct{}

func (p PathVarBinding) Bind(r *http.Request, obj any) error {
	//TODO implement me
	panic("implement me")
}

func (p PathVarBinding) Name() string {
	return "pathVar"
}
