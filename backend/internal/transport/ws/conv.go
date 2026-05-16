package ws

import (
	"encoding/json"
	"fmt"

	"qim/internal/actor"
	"qim/internal/domain/conversation"
	"qim/internal/service"
)

type convRouter struct {
	svc *service.ConvService
}

func (r *convRouter) dispatch(uid uint64, action string, data json.RawMessage) WsResponse {
	cmd, convID, fn, err := r.resolveAction(uid, action, data)
	if err != nil {
		return errReply(action, err)
	}
	ref, err := r.resolveRef(convID)
	if err != nil {
		return errReply(action, err)
	}
	return fn(ref, cmd, action)
}

func (r *convRouter) resolveAction(uid uint64, action string, data json.RawMessage) (cmd any, convID uint64, fn dispatchFn, err error) {
	switch action {
	case "list":
		return conversation.ListUserConversationsCmd{UID: uid}, 0, askDispatch, nil
	case "create_private":
		var req struct {
			UID uint64 `json:"uid"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, 0, nil, err
		}
		return conversation.CreatePrivateConvCmd{UID1: uid, UID2: req.UID}, 0, askDispatch, nil
	case "create_group":
		var req struct {
			Name    string   `json:"name"`
			Avatar  string   `json:"avatar"`
			Members []uint64 `json:"members"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, 0, nil, err
		}
		return conversation.CreateGroupConvCmd{OwnerID: uid, Name: req.Name, Avatar: req.Avatar, Members: req.Members}, 0, askDispatch, nil
	case "read_all":
		return conversation.ReadAllConvCmd{UID: uid}, 0, tellDispatch, nil
	case "update_info":
		var req struct {
			ConvID      uint64  `json:"conv_id"`
			Name        *string `json:"name"`
			Avatar      *string `json:"avatar"`
			MemberLimit *int    `json:"member_limit"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, 0, nil, err
		}
		return conversation.UpdateConvInfoCmd{
			OperatorID:  uid,
			Name:        req.Name,
			Avatar:      req.Avatar,
			MemberLimit: req.MemberLimit,
		}, req.ConvID, tellDispatch, nil
	case "pin":
		var req struct {
			ConvID uint64 `json:"conv_id"`
			Pinned bool   `json:"pinned"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, 0, nil, err
		}
		return conversation.PinConvCmd{UID: uid, Pinned: req.Pinned}, req.ConvID, tellDispatch, nil
	case "mute":
		var req struct {
			ConvID uint64 `json:"conv_id"`
			Muted  bool   `json:"muted"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, 0, nil, err
		}
		return conversation.MuteConvCmd{UID: uid, Muted: req.Muted}, req.ConvID, tellDispatch, nil
	case "read":
		var req struct {
			ConvID uint64 `json:"conv_id"`
			Seq    int64  `json:"seq"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, 0, nil, err
		}
		return conversation.ReadConvCmd{UID: uid, Seq: req.Seq}, req.ConvID, tellDispatch, nil
	case "members":
		var req struct {
			ConvID uint64 `json:"conv_id"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, 0, nil, err
		}
		return conversation.ListMembersQuery{}, req.ConvID, askDispatch, nil
	case "add_member":
		var req struct {
			ConvID uint64 `json:"conv_id"`
			UID    uint64 `json:"uid"`
			Role   int8   `json:"role"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, 0, nil, err
		}
		return conversation.AddMemberCmd{OperatorID: uid, UID: req.UID, Role: conversation.MemberRole(req.Role)}, req.ConvID, tellDispatch, nil
	case "remove_member":
		var req struct {
			ConvID uint64 `json:"conv_id"`
			UID    uint64 `json:"uid"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, 0, nil, err
		}
		return conversation.RemoveMemberCmd{OperatorID: uid, UID: req.UID}, req.ConvID, tellDispatch, nil
	case "leave":
		var req struct {
			ConvID uint64 `json:"conv_id"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, 0, nil, err
		}
		return conversation.LeaveConvCmd{UID: uid}, req.ConvID, tellDispatch, nil
	case "set_role":
		var req struct {
			ConvID uint64 `json:"conv_id"`
			UID    uint64 `json:"uid"`
			Role   int8   `json:"role"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, 0, nil, err
		}
		return conversation.SetRoleCmd{OperatorID: uid, UID: req.UID, Role: conversation.MemberRole(req.Role)}, req.ConvID, tellDispatch, nil
	case "transfer_owner":
		var req struct {
			ConvID     uint64 `json:"conv_id"`
			NewOwnerID uint64 `json:"new_owner_id"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, 0, nil, err
		}
		return conversation.TransferOwnerCmd{OperatorID: uid, NewOwnerID: req.NewOwnerID}, req.ConvID, tellDispatch, nil
	case "dissolve":
		var req struct {
			ConvID uint64 `json:"conv_id"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, 0, nil, err
		}
		return conversation.DissolveConvCmd{OperatorID: uid}, req.ConvID, tellDispatch, nil
	default:
		return nil, 0, nil, fmt.Errorf("unknown conv action: %s", action)
	}
}

func (r *convRouter) resolveRef(convID uint64) (*actor.ActorRef, error) {
	if convID == 0 {
		return r.svc.ManagerRef()
	}
	return r.svc.ConvRef(convID)
}
