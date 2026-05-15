package http

import (
	"qim/internal/domain/friend"
	"qim/internal/pkg/resp"
	"qim/internal/service"

	"github.com/gin-gonic/gin"
)

type FriendHandler struct {
	svc *service.FriendService
}

func NewFriendHandler(svc *service.FriendService) *FriendHandler {
	return &FriendHandler{svc: svc}
}

func handleFriendResult(c *gin.Context, r friend.Result, err error) {
	if err != nil {
		resp.Fail(c, 500, err.Error())
		return
	}
	if r.Err != nil {
		resp.Fail(c, 400, r.Err.Error())
		return
	}
	resp.OK(c, r.Data)
}

func (h *FriendHandler) SendRequest(c *gin.Context) {
	uid := c.GetUint64("uid")
	var req struct {
		ToUID   uint64 `json:"to_uid"`
		Message string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, err.Error())
		return
	}
	r, err := h.svc.Ask(friend.SendRequestCmd{FromUID: uid, ToUID: req.ToUID, Message: req.Message})
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
	reqID := c.GetUint64("req_id")
	var req struct {
		Action string `json:"action"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, err.Error())
		return
	}
	r, err := h.svc.Ask(friend.HandleRequestCmd{UID: uid, ReqID: reqID, Accept: req.Action == "accept"})
	handleFriendResult(c, r, err)
}

func (h *FriendHandler) DeleteFriend(c *gin.Context) {
	uid := c.GetUint64("uid")
	friendUID := c.GetUint64("friend_uid")
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
	friendUID := c.GetUint64("friend_uid")
	var req struct {
		Remark string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, err.Error())
		return
	}
	r, err := h.svc.Ask(friend.UpdateRemarkCmd{UID: uid, FriendUID: friendUID, Remark: req.Remark})
	handleFriendResult(c, r, err)
}

func (h *FriendHandler) MoveGroup(c *gin.Context) {
	uid := c.GetUint64("uid")
	friendUID := c.GetUint64("friend_uid")
	var req struct {
		GroupID uint64 `json:"group_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, err.Error())
		return
	}
	r, err := h.svc.Ask(friend.MoveGroupCmd{UID: uid, FriendUID: friendUID, GroupID: req.GroupID})
	handleFriendResult(c, r, err)
}
