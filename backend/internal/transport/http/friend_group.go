package http

import (
	"qim/internal/domain/friend"
	"qim/internal/pkg/resp"

	"github.com/gin-gonic/gin"
)

func (h *FriendHandler) ListGroups(c *gin.Context) {
	uid := c.GetUint64("uid")
	r, err := h.svc.Ask(friend.ListGroupsCmd{UID: uid})
	handleFriendResult(c, r, err)
}

func (h *FriendHandler) CreateGroup(c *gin.Context) {
	uid := c.GetUint64("uid")
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, badRequest(err))
		return
	}
	r, err := h.svc.Ask(friend.CreateGroupCmd{UID: uid, Name: req.Name})
	handleFriendResult(c, r, err)
}

func (h *FriendHandler) RenameGroup(c *gin.Context) {
	uid := c.GetUint64("uid")
	groupID, ok := paramUint(c, "group_id")
	if !ok {
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, badRequest(err))
		return
	}
	r, err := h.svc.Ask(friend.RenameGroupCmd{UID: uid, GroupID: groupID, Name: req.Name})
	handleFriendResult(c, r, err)
}

func (h *FriendHandler) DeleteGroup(c *gin.Context) {
	uid := c.GetUint64("uid")
	groupID, ok := paramUint(c, "group_id")
	if !ok {
		return
	}
	r, err := h.svc.Ask(friend.DeleteGroupCmd{UID: uid, GroupID: groupID})
	handleFriendResult(c, r, err)
}

func (h *FriendHandler) SortGroups(c *gin.Context) {
	uid := c.GetUint64("uid")
	var req struct {
		Groups []struct {
			GroupID   uint64 `json:"group_id"`
			SortOrder int    `json:"sort_order"`
		} `json:"groups"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, badRequest(err))
		return
	}
	groups := make([]struct {
		GroupID   uint64
		SortOrder int
	}, len(req.Groups))
	for i, g := range req.Groups {
		groups[i].GroupID = g.GroupID
		groups[i].SortOrder = g.SortOrder
	}
	r, err := h.svc.Ask(friend.SortGroupsCmd{UID: uid, Groups: groups})
	handleFriendResult(c, r, err)
}
