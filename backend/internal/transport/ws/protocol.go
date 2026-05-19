package ws

import (
	"encoding/json"

	"qim/internal/pkg/apperr"
	"qim/internal/pkg/pushtype"
)

type WsRequest struct {
	Type   string          `json:"type"`
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data"`
	LogID  string          `json:"-"`
}

type WsResponse struct {
	Type   string          `json:"type"`
	Action string          `json:"action,omitempty"`
	Data   any             `json:"data,omitempty"`
	Error  *apperr.Payload `json:"error,omitempty"`
	LogID  string          `json:"log_id,omitempty"`
}

type WSDisconnected struct{}

type IdleTimeout struct{}

type PushCmd = pushtype.PushCmd
