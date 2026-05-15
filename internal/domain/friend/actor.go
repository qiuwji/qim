package friend

import (
	"fmt"
	"time"

	"qim/internal/actor"
	"qim/internal/dal"
)

type ManagerActor struct {
	store  dal.FriendStore
	engine *actor.Engine
}

func NewManagerActor(store dal.FriendStore, engine *actor.Engine) *ManagerActor {
	return &ManagerActor{store: store, engine: engine}
}

func (a *ManagerActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case SendRequestCmd:
		a.handleSendRequest(ctx, msg)
	case ListIncomingCmd:
		a.handleListIncoming(ctx, msg)
	case ListOutgoingCmd:
		a.handleListOutgoing(ctx, msg)
	case HandleRequestCmd:
		a.handleRequest(ctx, msg)
	case DeleteFriendCmd:
		a.handleDeleteFriend(ctx, msg)
	case ListFriendsCmd:
		a.handleListFriends(ctx, msg)
	case UpdateRemarkCmd:
		a.handleUpdateRemark(ctx, msg)
	case MoveGroupCmd:
		a.handleMoveGroup(ctx, msg)
	case ListGroupsCmd:
		a.handleListGroups(ctx, msg)
	case CreateGroupCmd:
		a.handleCreateGroup(ctx, msg)
	case RenameGroupCmd:
		a.handleRenameGroup(ctx, msg)
	case DeleteGroupCmd:
		a.handleDeleteGroup(ctx, msg)
	case SortGroupsCmd:
		a.handleSortGroups(ctx, msg)
	}
}

func (a *ManagerActor) handleSendRequest(ctx actor.Context, msg SendRequestCmd) {
	now := time.Now().Unix()
	req := &dal.FriendRequest{
		FromUID:   msg.FromUID,
		ToUID:     msg.ToUID,
		Message:   msg.Message,
		Status:    int8(FriendRequestPending),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := a.store.CreateRequest(req); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	ctx.Reply(Result{Data: FriendRequestDTO{
		ID:        req.ID,
		FromUID:   req.FromUID,
		ToUID:     req.ToUID,
		Message:   req.Message,
		Status:    FriendRequestStatus(req.Status),
		CreatedAt: req.CreatedAt,
	}})
}

func (a *ManagerActor) handleListIncoming(ctx actor.Context, msg ListIncomingCmd) {
	reqs, err := a.store.ListIncomingRequests(msg.UID)
	if err != nil {
		ctx.Reply(Result{Err: err})
		return
	}

	dtos := make([]FriendRequestDTO, 0, len(reqs))
	for _, r := range reqs {
		dtos = append(dtos, FriendRequestDTO{
			ID:        r.ID,
			FromUID:   r.FromUID,
			ToUID:     r.ToUID,
			Message:   r.Message,
			Status:    FriendRequestStatus(r.Status),
			CreatedAt: r.CreatedAt,
		})
	}
	ctx.Reply(Result{Data: dtos})
}

func (a *ManagerActor) handleListOutgoing(ctx actor.Context, msg ListOutgoingCmd) {
	reqs, err := a.store.ListOutgoingRequests(msg.UID)
	if err != nil {
		ctx.Reply(Result{Err: err})
		return
	}

	dtos := make([]FriendRequestDTO, 0, len(reqs))
	for _, r := range reqs {
		dtos = append(dtos, FriendRequestDTO{
			ID:        r.ID,
			FromUID:   r.FromUID,
			ToUID:     r.ToUID,
			Message:   r.Message,
			Status:    FriendRequestStatus(r.Status),
			CreatedAt: r.CreatedAt,
		})
	}
	ctx.Reply(Result{Data: dtos})
}

func (a *ManagerActor) handleRequest(ctx actor.Context, msg HandleRequestCmd) {
	req, err := a.store.GetRequest(msg.ReqID)
	if err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	if req.ToUID != msg.UID {
		ctx.Reply(Result{Err: fmt.Errorf("not your request")})
		return
	}

	if msg.Accept {
		if err := a.store.AcceptFriendRequest(msg.ReqID, req.FromUID, req.ToUID); err != nil {
			ctx.Reply(Result{Err: err})
			return
		}
	} else {
		if err := a.store.RejectFriendRequest(msg.ReqID); err != nil {
			ctx.Reply(Result{Err: err})
			return
		}
	}
	ctx.Reply(Result{Data: true})
}

func (a *ManagerActor) handleDeleteFriend(ctx actor.Context, msg DeleteFriendCmd) {
	if err := a.store.DeleteFriendBidirectional(msg.UID, msg.FriendUID); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	ctx.Reply(Result{Data: true})
}

func (a *ManagerActor) handleListFriends(ctx actor.Context, msg ListFriendsCmd) {
	friends, err := a.store.ListFriends(msg.UID)
	if err != nil {
		ctx.Reply(Result{Err: err})
		return
	}

	dtos := make([]FriendDTO, 0, len(friends))
	for _, f := range friends {
		dtos = append(dtos, FriendDTO{
			ID:        f.ID,
			FriendUID: f.FriendUID,
			Remark:    f.Remark,
			GroupID:   f.GroupID,
			CreatedAt: f.CreatedAt,
		})
	}
	ctx.Reply(Result{Data: dtos})
}

func (a *ManagerActor) handleUpdateRemark(ctx actor.Context, msg UpdateRemarkCmd) {
	if err := a.store.UpdateFriend(msg.UID, msg.FriendUID, map[string]any{"remark": msg.Remark}); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	ctx.Reply(Result{Data: true})
}

func (a *ManagerActor) handleMoveGroup(ctx actor.Context, msg MoveGroupCmd) {
	if err := a.store.UpdateFriend(msg.UID, msg.FriendUID, map[string]any{"group_id": msg.GroupID}); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	ctx.Reply(Result{Data: true})
}

func (a *ManagerActor) handleListGroups(ctx actor.Context, msg ListGroupsCmd) {
	groups, err := a.store.ListGroups(msg.UID)
	if err != nil {
		ctx.Reply(Result{Err: err})
		return
	}

	dtos := make([]FriendGroupDTO, 0, len(groups))
	for _, g := range groups {
		dtos = append(dtos, FriendGroupDTO{
			ID:        g.ID,
			Name:      g.Name,
			SortOrder: g.SortOrder,
		})
	}
	ctx.Reply(Result{Data: dtos})
}

func (a *ManagerActor) handleCreateGroup(ctx actor.Context, msg CreateGroupCmd) {
	group := &dal.FriendGroup{
		UserID: msg.UID,
		Name:   msg.Name,
	}
	if err := a.store.CreateGroup(group); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	ctx.Reply(Result{Data: FriendGroupDTO{
		ID:   group.ID,
		Name: group.Name,
	}})
}

func (a *ManagerActor) handleRenameGroup(ctx actor.Context, msg RenameGroupCmd) {
	if err := a.store.UpdateGroup(msg.GroupID, msg.UID, map[string]any{"name": msg.Name}); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	ctx.Reply(Result{Data: true})
}

func (a *ManagerActor) handleDeleteGroup(ctx actor.Context, msg DeleteGroupCmd) {
	if err := a.store.UpdateFriend(msg.UID, 0, map[string]any{"group_id": 0}); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	if err := a.store.DeleteGroup(msg.GroupID, msg.UID); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	ctx.Reply(Result{Data: true})
}

func (a *ManagerActor) handleSortGroups(ctx actor.Context, msg SortGroupsCmd) {
	for _, g := range msg.Groups {
		a.store.UpdateGroup(g.GroupID, msg.UID, map[string]any{"sort_order": g.SortOrder})
	}
	ctx.Reply(Result{Data: true})
}
