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
	switch action {
	case "initiate":
		return r.handleInitiate(uid, data)
	case "accept":
		return r.handleAccept(uid, data)
	case "reject":
		return r.handleReject(uid, data)
	case "cancel":
		return r.handleCancel(uid, data)
	case "end":
		return r.handleEnd(uid, data)
	case "offer":
		return r.handleOffer(uid, data)
	case "answer":
		return r.handleAnswer(uid, data)
	case "ice":
		return r.handleIce(uid, data)
	default:
		return errReply(action, fmt.Errorf("unknown call action: %s", action))
	}
}

func (r *callRouter) handleInitiate(uid uint64, data json.RawMessage) WsResponse {
	var req struct {
		CalleeUID uint64 `json:"callee_uid"`
		CallType  int8   `json:"call_type"`
	}
	if err := json.Unmarshal(data, &req); err != nil {
		return errReply("initiate", err)
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

	result, err := r.svc.AskManager(call.InitiateCallCmd{
		CallerUID:      uid,
		CalleeUID:      req.CalleeUID,
		CallType:       req.CallType,
		CallerNickname: callerNickname,
		CallerAvatar:   callerAvatar,
	})
	if err != nil {
		return errReply("initiate", err)
	}
	if result.Err != nil {
		return errReply("initiate", result.Err)
	}
	return WsResponse{Type: "ack", Action: "initiate", Data: result.Data}
}

func (r *callRouter) handleAccept(uid uint64, data json.RawMessage) WsResponse {
	var req struct {
		CallID string `json:"call_id"`
	}
	if err := json.Unmarshal(data, &req); err != nil {
		return errReply("accept", err)
	}
	result, err := r.svc.AskManager(call.AcceptCallCmd{CallID: req.CallID, UID: uid})
	if err != nil {
		return errReply("accept", err)
	}
	return domainCallResultReply("accept", result)
}

func (r *callRouter) handleReject(uid uint64, data json.RawMessage) WsResponse {
	var req struct {
		CallID string `json:"call_id"`
	}
	if err := json.Unmarshal(data, &req); err != nil {
		return errReply("reject", err)
	}
	result, err := r.svc.AskManager(call.RejectCallCmd{CallID: req.CallID, UID: uid})
	if err != nil {
		return errReply("reject", err)
	}
	return domainCallResultReply("reject", result)
}

func (r *callRouter) handleCancel(uid uint64, data json.RawMessage) WsResponse {
	var req struct {
		CallID string `json:"call_id"`
	}
	if err := json.Unmarshal(data, &req); err != nil {
		return errReply("cancel", err)
	}
	result, err := r.svc.AskManager(call.CancelCallCmd{CallID: req.CallID, UID: uid})
	if err != nil {
		return errReply("cancel", err)
	}
	return domainCallResultReply("cancel", result)
}

func (r *callRouter) handleEnd(uid uint64, data json.RawMessage) WsResponse {
	var req struct {
		CallID string `json:"call_id"`
	}
	if err := json.Unmarshal(data, &req); err != nil {
		return errReply("end", err)
	}
	result, err := r.svc.AskManager(call.EndCallCmd{CallID: req.CallID, UID: uid})
	if err != nil {
		return errReply("end", err)
	}
	return domainCallResultReply("end", result)
}

func (r *callRouter) handleOffer(uid uint64, data json.RawMessage) WsResponse {
	var req struct {
		CallID string `json:"call_id"`
		SDP    string `json:"sdp"`
	}
	if err := json.Unmarshal(data, &req); err != nil {
		return errReply("offer", err)
	}
	result, err := r.svc.AskManager(call.ForwardOfferCmd{CallID: req.CallID, UID: uid, SDP: req.SDP})
	if err != nil {
		return errReply("offer", err)
	}
	return domainCallResultReply("offer", result)
}

func (r *callRouter) handleAnswer(uid uint64, data json.RawMessage) WsResponse {
	var req struct {
		CallID string `json:"call_id"`
		SDP    string `json:"sdp"`
	}
	if err := json.Unmarshal(data, &req); err != nil {
		return errReply("answer", err)
	}
	result, err := r.svc.AskManager(call.ForwardAnswerCmd{CallID: req.CallID, UID: uid, SDP: req.SDP})
	if err != nil {
		return errReply("answer", err)
	}
	return domainCallResultReply("answer", result)
}

func (r *callRouter) handleIce(uid uint64, data json.RawMessage) WsResponse {
	var req struct {
		CallID        string `json:"call_id"`
		Candidate     string `json:"candidate"`
		SDPMid        string `json:"sdp_mid"`
		SDPMLineIndex int    `json:"sdp_m_line_index"`
	}
	if err := json.Unmarshal(data, &req); err != nil {
		return errReply("ice", err)
	}
	result, err := r.svc.AskManager(call.ForwardIceCmd{
		CallID:        req.CallID,
		UID:           uid,
		Candidate:     req.Candidate,
		SDPMid:        req.SDPMid,
		SDPMLineIndex: req.SDPMLineIndex,
	})
	if err != nil {
		return errReply("ice", err)
	}
	return domainCallResultReply("ice", result)
}

func domainCallResultReply(action string, result call.Result) WsResponse {
	if result.Err != nil {
		return errReply(action, result.Err)
	}
	return WsResponse{Type: "ack", Action: action, Data: result.Data}
}
