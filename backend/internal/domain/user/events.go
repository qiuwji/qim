package user

const EventUserOnline = "user.online"

type UserOnlineEvent struct {
	UID uint64
	At  int64
}

func (UserOnlineEvent) Name() string {
	return EventUserOnline
}
