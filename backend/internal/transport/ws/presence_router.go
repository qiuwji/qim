package ws

import (
	"encoding/json"
	"fmt"

	"qim/internal/actor"
	"qim/internal/domain/friend"
	"qim/internal/domain/presence"
)

type presenceRouter struct {
	presenceRef *actor.ActorRef
	friendRef   *actor.ActorRef
}

func (r *presenceRouter) dispatch(uid uint64, action string, data json.RawMessage) WsResponse {
	switch action {
	case "online_friends":
		return r.onlineFriends(uid)
	default:
		return errReply(action, fmt.Errorf("unknown presence action: %s", action))
	}
}

func (r *presenceRouter) onlineFriends(uid uint64) WsResponse {
	if r.friendRef == nil || r.presenceRef == nil {
		return WsResponse{Type: "ack", Action: "online_friends", Data: map[uint64]bool{}}
	}
	raw, err := r.friendRef.Ask(friend.ListFriendsCmd{UID: uid}, askTimeout)
	if err != nil {
		return errReply("online_friends", err)
	}
	fr, ok := raw.(friend.Result)
	if !ok || fr.Err != nil || fr.Data == nil {
		return WsResponse{Type: "ack", Action: "online_friends", Data: map[uint64]bool{}}
	}
	dtos, ok := fr.Data.([]friend.FriendDTO)
	if !ok {
		return WsResponse{Type: "ack", Action: "online_friends", Data: map[uint64]bool{}}
	}
	uids := make([]uint64, 0, len(dtos))
	for _, f := range dtos {
		uids = append(uids, f.FriendUID)
	}
	raw2, err := r.presenceRef.Ask(presence.BatchOnlineQuery{UIDs: uids}, askTimeout)
	if err != nil {
		return errReply("online_friends", err)
	}
	result, ok := raw2.(presence.BatchOnlineResult)
	if !ok {
		return WsResponse{Type: "ack", Action: "online_friends", Data: map[uint64]bool{}}
	}
	return WsResponse{Type: "ack", Action: "online_friends", Data: result.OnlineMap}
}
