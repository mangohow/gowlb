package gmcp

import (
	"encoding/json"
)

const (
	JSONRPCVersion = "2.0"
)

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	ID      any             `json:"id"`
	Params  json.RawMessage `json:"params"`
}

type ErrorInfo struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type JSONRPCError struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id"`
	Err     ErrorInfo `json:"error"`
}

func (e *JSONRPCError) Error() string {
	jsonData, err := json.Marshal(e)
	if err != nil {
		return err.Error()
	}

	return string(jsonData)
}

type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   JSONRPCError    `json:"error"`
}
