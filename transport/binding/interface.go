package binding

import "net/http"

type Binding interface {
	Name() string
	Bind(r *http.Request, obj any) error
}

var (
	registeredBinding = map[string]Binding{
		FormBinding{}.Name():    FormBinding{},
		JsonBinding{}.Name():    JsonBinding{},
		PathVarBinding{}.Name(): PathVarBinding{},
		QueryBinding{}.Name():   QueryBinding{},
	}
)

func RegisterBinding(b Binding) {
	if b == nil {
		panic("binding is nil")
	}

	registeredBinding[b.Name()] = b
}

func GetBinding(name string) Binding {
	return registeredBinding[name]
}
