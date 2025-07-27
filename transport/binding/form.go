package binding

import "net/http"

type FormBinding struct{}

func (f FormBinding) Bind(r *http.Request, obj any) error {
	return nil
}

func (f FormBinding) Name() string {
	return "form"
}
