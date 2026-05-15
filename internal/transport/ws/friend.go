package ws

import (
	"encoding/json"
	"fmt"

	"qim/internal/actor"
	"qim/internal/domain/friend"
)

type friendRouter struct {
	engine *actor.Engine
}

func (r *friendRouter) dispatch(uid uint64, action string, data json.RawMessage) WsResponse {
	cmd, fn, err := r.resolveAction(uid, action, data)
	if err != nil {
		return errReply(action, err.Error())
	}
	ref, ok := r.engine.Lookup("friend-manager")
	if !ok {
		return errReply(action, "friend-manager unavailable")
	}
	return fn(ref, cmd, action)
}

func (r *friendRouter) resolveAction(uid uint64, action string, data json.RawMessage) (any, dispatchFn, error) {
	switch action {
	case "send_request":
		var req struct {
			ToUID   uint64 `json:"to_uid"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, nil, err
		}
		return friend.SendRequestCmd{FromUID: uid, ToUID: req.ToUID, Message: req.Message}, askDispatch, nil
	case "list_incoming":
		return friend.ListIncomingCmd{UID: uid}, askDispatch, nil
	case "list_outgoing":
		return friend.ListOutgoingCmd{UID: uid}, askDispatch, nil
	case "handle_request":
		var req struct {
			ReqID  uint64 `json:"req_id"`
			Accept bool   `json:"accept"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, nil, err
		}
		return friend.HandleRequestCmd{UID: uid, ReqID: req.ReqID, Accept: req.Accept}, tellDispatch, nil
	case "delete":
		var req struct {
			FriendUID uint64 `json:"friend_uid"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, nil, err
		}
		return friend.DeleteFriendCmd{UID: uid, FriendUID: req.FriendUID}, tellDispatch, nil
	case "list":
		return friend.ListFriendsCmd{UID: uid}, askDispatch, nil
	case "update_remark":
		var req struct {
			FriendUID uint64 `json:"friend_uid"`
			Remark    string `json:"remark"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, nil, err
		}
		return friend.UpdateRemarkCmd{UID: uid, FriendUID: req.FriendUID, Remark: req.Remark}, tellDispatch, nil
	case "move_group":
		var req struct {
			FriendUID uint64 `json:"friend_uid"`
			GroupID   uint64 `json:"group_id"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, nil, err
		}
		return friend.MoveGroupCmd{UID: uid, FriendUID: req.FriendUID, GroupID: req.GroupID}, tellDispatch, nil
	case "list_groups":
		return friend.ListGroupsCmd{UID: uid}, askDispatch, nil
	case "create_group":
		var req struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, nil, err
		}
		return friend.CreateGroupCmd{UID: uid, Name: req.Name}, askDispatch, nil
	case "rename_group":
		var req struct {
			GroupID uint64 `json:"group_id"`
			Name    string `json:"name"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, nil, err
		}
		return friend.RenameGroupCmd{UID: uid, GroupID: req.GroupID, Name: req.Name}, tellDispatch, nil
	case "delete_group":
		var req struct {
			GroupID uint64 `json:"group_id"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, nil, err
		}
		return friend.DeleteGroupCmd{UID: uid, GroupID: req.GroupID}, tellDispatch, nil
	case "sort_groups":
		var req struct {
			Groups []struct {
				GroupID   uint64 `json:"group_id"`
				SortOrder int    `json:"sort_order"`
			} `json:"groups"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, nil, err
		}
		groups := make([]struct {
			GroupID   uint64
			SortOrder int
		}, len(req.Groups))
		for i, g := range req.Groups {
			groups[i].GroupID = g.GroupID
			groups[i].SortOrder = g.SortOrder
		}
		return friend.SortGroupsCmd{UID: uid, Groups: groups}, tellDispatch, nil
	default:
		return nil, nil, fmt.Errorf("unknown friend action: %s", action)
	}
}
