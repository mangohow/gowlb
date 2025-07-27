package binding

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type JsonBinding struct{}

func (j JsonBinding) Bind(r *http.Request, obj any) error {
	if r == nil || r.Body == nil {
		return errors.New("bind json error: invalid request body")
	}
	if obj == nil {
		return errors.New("bind json error: obj is nil")
	}

	if err := json.NewDecoder(r.Body).Decode(obj); err != nil {
		return fmt.Errorf("bind json error: %w", err)
	}

	_ = r.Body.Close()

	return nil
}

func (j JsonBinding) Name() string {
	return "json"
}
