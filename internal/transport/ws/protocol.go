package ws

import "encoding/json"

type WsRequest struct {
	Type   string          `json:"type"`
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data"`
}

type WsResponse struct {
	Type   string `json:"type"`
	Action string `json:"action,omitempty"`
	Data   any    `json:"data"`
}

type WSDisconnected struct{}

type IdleTimeout struct{}

type PushCmd struct {
	Type string
	Data any
}
