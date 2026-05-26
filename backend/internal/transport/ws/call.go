package ws

import (
	"encoding/json"
	"fmt"

	"qim/internal/domain/call"
	"qim/internal/domain/user"
	"qim/internal/service"
)

type callRouter struct {
	svc     *service.CallService
	userSvc *service.UserService
}

func (r *callRouter) dispatch(uid uint64, action string, data json.RawMessage) WsResponse {
	cmd, fn, err := r.resolveAction(uid, action, data)
	if err != nil {
		return errReply(action, err)
	}
	ref, err := r.svc.ManagerRef()
	if err != nil {
		return errReply(action, err)
	}
	return fn(ref, cmd, action)
}

func (r *callRouter) resolveAction(uid uint64, action string, data json.RawMessage) (any, dispatchFn, error) {
	switch action {
	case "initiate":
		var req struct {
			CalleeUID uint64 `json:"callee_uid"`
			CallType  int8   `json:"call_type"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, nil, err
		}
		callerNickname := ""
		callerAvatar := ""
		if r.userSvc != nil {
			if result, err := r.userSvc.AskManager(user.GetUserCmd{UID: uid}); err == nil && result.Err == nil {
				if dto, ok := result.Data.(user.UserDTO); ok {
					callerNickname = dto.Nickname
					callerAvatar = dto.Avatar
				}
			}
		}
		return call.InitiateCallCmd{
			CallerUID:      uid,
			CalleeUID:      req.CalleeUID,
			CallType:       req.CallType,
			CallerNickname: callerNickname,
			CallerAvatar:   callerAvatar,
		}, askDispatch, nil
	case "accept":
		var req struct {
			CallID string `json:"call_id"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, nil, err
		}
		return call.AcceptCallCmd{CallID: req.CallID, UID: uid}, askDispatch, nil
	case "reject":
		var req struct {
			CallID string `json:"call_id"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, nil, err
		}
		return call.RejectCallCmd{CallID: req.CallID, UID: uid}, askDispatch, nil
	case "cancel":
		var req struct {
			CallID string `json:"call_id"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, nil, err
		}
		return call.CancelCallCmd{CallID: req.CallID, UID: uid}, askDispatch, nil
	case "end":
		var req struct {
			CallID string `json:"call_id"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, nil, err
		}
		return call.EndCallCmd{CallID: req.CallID, UID: uid}, askDispatch, nil
	case "offer":
		var req struct {
			CallID string `json:"call_id"`
			SDP    string `json:"sdp"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, nil, err
		}
		return call.ForwardOfferCmd{CallID: req.CallID, UID: uid, SDP: req.SDP}, askDispatch, nil
	case "answer":
		var req struct {
			CallID string `json:"call_id"`
			SDP    string `json:"sdp"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, nil, err
		}
		return call.ForwardAnswerCmd{CallID: req.CallID, UID: uid, SDP: req.SDP}, askDispatch, nil
	case "ice":
		var req struct {
			CallID        string `json:"call_id"`
			Candidate     string `json:"candidate"`
			SDPMid        string `json:"sdp_mid"`
			SDPMLineIndex int    `json:"sdp_m_line_index"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, nil, err
		}
		return call.ForwardIceCmd{
			CallID:        req.CallID,
			UID:           uid,
			Candidate:     req.Candidate,
			SDPMid:        req.SDPMid,
			SDPMLineIndex: req.SDPMLineIndex,
		}, askDispatch, nil
	default:
		return nil, nil, fmt.Errorf("unknown call action: %s", action)
	}
}
