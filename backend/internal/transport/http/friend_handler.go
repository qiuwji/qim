package http

import (
	"qim/internal/domain/friend"
	"qim/internal/pkg/resp"
	"qim/internal/service"

	"github.com/gin-gonic/gin"
)

type FriendHandler struct {
	svc     *service.FriendService
	userSvc *service.UserService
}

func NewFriendHandler(svc *service.FriendService, userSvc *service.UserService) *FriendHandler {
	return &FriendHandler{svc: svc, userSvc: userSvc}
}

func handleFriendResult(c *gin.Context, r friend.Result, err error) {
	if err != nil {
		logHTTPError(c, "friend request failed", err)
		resp.Fail(c, internalError(err))
		return
	}
	if r.Err != nil {
		logHTTPError(c, "friend domain error", r.Err)
		resp.Fail(c, r.Err)
		return
	}
	resp.OK(c, r.Data)
}

func (h *FriendHandler) SendRequest(c *gin.Context) {
	uid := c.GetUint64("uid")
	var req struct {
		ToUID    uint64 `json:"to_uid"`
		Username string `json:"username"`
		Message  string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, badRequest(err))
		return
	}
	toUID := req.ToUID
	if req.Username != "" {
		resolvedUID, ok := resolveUsername(c, h.userSvc, req.Username)
		if !ok {
			return
		}
		toUID = resolvedUID
	}
	r, err := h.svc.Ask(friend.SendRequestCmd{FromUID: uid, ToUID: toUID, Message: req.Message})
	handleFriendResult(c, r, err)
}

func (h *FriendHandler) ListIncoming(c *gin.Context) {
	uid := c.GetUint64("uid")
	r, err := h.svc.Ask(friend.ListIncomingCmd{UID: uid})
	handleFriendResult(c, r, err)
}

func (h *FriendHandler) ListOutgoing(c *gin.Context) {
	uid := c.GetUint64("uid")
	r, err := h.svc.Ask(friend.ListOutgoingCmd{UID: uid})
	handleFriendResult(c, r, err)
}

func (h *FriendHandler) HandleRequest(c *gin.Context) {
	uid := c.GetUint64("uid")
	reqID, ok := paramUint(c, "req_id")
	if !ok {
		return
	}
	var req struct {
		Action string `json:"action"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, badRequest(err))
		return
	}
	r, err := h.svc.Ask(friend.HandleRequestCmd{UID: uid, ReqID: reqID, Accept: req.Action == "accept"})
	handleFriendResult(c, r, err)
}

func (h *FriendHandler) DeleteFriend(c *gin.Context) {
	uid := c.GetUint64("uid")
	friendUID, ok := paramUint(c, "friend_uid")
	if !ok {
		return
	}
	r, err := h.svc.Ask(friend.DeleteFriendCmd{UID: uid, FriendUID: friendUID})
	handleFriendResult(c, r, err)
}

func (h *FriendHandler) ListFriends(c *gin.Context) {
	uid := c.GetUint64("uid")
	r, err := h.svc.Ask(friend.ListFriendsCmd{UID: uid})
	handleFriendResult(c, r, err)
}

func (h *FriendHandler) UpdateRemark(c *gin.Context) {
	uid := c.GetUint64("uid")
	friendUID, ok := paramUint(c, "friend_uid")
	if !ok {
		return
	}
	var req struct {
		Remark string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, badRequest(err))
		return
	}
	r, err := h.svc.Ask(friend.UpdateRemarkCmd{UID: uid, FriendUID: friendUID, Remark: req.Remark})
	handleFriendResult(c, r, err)
}

func (h *FriendHandler) MoveGroup(c *gin.Context) {
	uid := c.GetUint64("uid")
	friendUID, ok := paramUint(c, "friend_uid")
	if !ok {
		return
	}
	var req struct {
		GroupID uint64 `json:"group_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, badRequest(err))
		return
	}
	r, err := h.svc.Ask(friend.MoveGroupCmd{UID: uid, FriendUID: friendUID, GroupID: req.GroupID})
	handleFriendResult(c, r, err)
}
