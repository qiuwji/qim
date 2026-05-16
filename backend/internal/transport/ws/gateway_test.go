package ws

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"qim/internal/actor"
	"qim/internal/domain/presence"
	"qim/internal/eventbus"

	"github.com/gorilla/websocket"
)

func TestGatewayActorWebSocketFlow_BitsUT(t *testing.T) {
	engine := actor.NewEngine()
	presenceEvents := make(chan any, 4)
	if _, err := engine.Spawn("presence", gatewayPresenceActor{events: presenceEvents}); err != nil {
		t.Fatalf("spawn presence: %v", err)
	}
	bus := &gatewayEventBus{}

	var serverConn *websocket.Conn
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		serverConn = conn
	}))
	defer server.Close()

	client, _, err := websocket.DefaultDialer.Dial("ws"+server.URL[len("http"):], nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer client.Close()
	deadline := time.Now().Add(time.Second)
	for serverConn == nil && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond * 10)
	}
	if serverConn == nil {
		t.Fatalf("server conn was not established")
	}

	ref, err := engine.Spawn("gateway-flow-test", NewGatewayActor(1001, serverConn, engine, bus, gatewayDispatcher{}, "conn-log"))
	if err != nil {
		t.Fatalf("spawn gateway: %v", err)
	}
	select {
	case event := <-presenceEvents:
		if _, ok := event.(presence.UserConnected); !ok {
			t.Fatalf("presence event = %T", event)
		}
	case <-time.After(time.Second):
		t.Fatalf("presence connect event missing")
	}
	if bus.count != 1 {
		t.Fatalf("user online event count = %d", bus.count)
	}

	if err := client.WriteJSON(WsRequest{Type: "ping", Action: "echo"}); err != nil {
		t.Fatalf("write request: %v", err)
	}
	var resp WsResponse
	if err := client.ReadJSON(&resp); err != nil {
		t.Fatalf("read response: %v", err)
	}
	if resp.Type != "ack" || resp.Action != "echo" || resp.LogID == "" {
		t.Fatalf("response = %+v", resp)
	}

	if err := ref.Tell(PushCmd{Type: "notice", Action: "server", Data: map[string]string{"ok": "true"}}); err != nil {
		t.Fatalf("tell push: %v", err)
	}
	if err := client.ReadJSON(&resp); err != nil {
		t.Fatalf("read push: %v", err)
	}
	if resp.Type != "notice" || resp.Action != "server" || resp.LogID == "" {
		t.Fatalf("push response = %+v", resp)
	}

	if err := ref.Tell(WSDisconnected{}); err != nil {
		t.Fatalf("tell disconnect: %v", err)
	}
	select {
	case event := <-presenceEvents:
		if _, ok := event.(presence.UserDisconnected); !ok {
			t.Fatalf("presence event = %T", event)
		}
	case <-time.After(time.Second):
		t.Fatalf("presence disconnect event missing")
	}
}

type gatewayDispatcher struct{}

func (gatewayDispatcher) Dispatch(uid uint64, req WsRequest) WsResponse {
	return WsResponse{Type: "ack", Action: req.Action, Data: map[string]uint64{"uid": uid}}
}

type gatewayPresenceActor struct {
	events chan any
}

func (a gatewayPresenceActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case presence.UserConnected:
		a.events <- msg
	case presence.UserDisconnected:
		a.events <- msg
	}
}

type gatewayEventBus struct {
	count int
}

func (b *gatewayEventBus) Publish(event eventbus.Event) error {
	b.count++
	return nil
}

func (b *gatewayEventBus) Subscribe(eventName string, subscriber *actor.ActorRef) error {
	return nil
}

func (b *gatewayEventBus) Unsubscribe(eventName string, subscriber *actor.ActorRef) error {
	return nil
}
