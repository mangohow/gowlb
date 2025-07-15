package serialize

import "github.com/mangohow/gowlb/errors"

type Response struct {
	Data  interface{}  `json:"data"`
	Error errors.Error `json:"error"`
}
