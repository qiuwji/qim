package gateway

type WsRequest struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type WsResponse struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}
