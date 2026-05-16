package friend

const (
	EventFriendRequestCreated = "friend.request_created"
	EventFriendRequestHandled = "friend.request_handled"
)

type FriendRequestCreatedEvent struct {
	RequestID uint64 `json:"request_id"`
	FromUID   uint64 `json:"from_uid"`
	ToUID     uint64 `json:"to_uid"`
	Message   string `json:"message"`
	CreatedAt int64  `json:"created_at"`
}

func (FriendRequestCreatedEvent) Name() string {
	return EventFriendRequestCreated
}

type FriendRequestHandledEvent struct {
	RequestID uint64 `json:"request_id"`
	FromUID   uint64 `json:"from_uid"`
	ToUID     uint64 `json:"to_uid"`
	Accepted  bool   `json:"accepted"`
}

func (FriendRequestHandledEvent) Name() string {
	return EventFriendRequestHandled
}
