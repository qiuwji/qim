package user

import (
	"time"

	"qim/internal/actor"
	"qim/internal/dal"
)

type SessionActor struct {
	uid    uint64
	engine *actor.Engine
	store  dal.UserStore
}

func NewSessionActor(uid uint64, store dal.UserStore, engine *actor.Engine) *SessionActor {
	return &SessionActor{
		uid:    uid,
		engine: engine,
		store:  store,
	}
}

func (a *SessionActor) OnStart(ctx actor.Context) {}

func (a *SessionActor) OnStop(ctx actor.Context) {}

func (a *SessionActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case GetProfileQuery:
		a.handleGetProfile(ctx)
	case UpdateProfileCmd:
		a.handleUpdateProfile(ctx, msg)
	case ChangePasswordCmd:
		a.handleChangePassword(ctx, msg)
	}
}

func (a *SessionActor) handleGetProfile(ctx actor.Context) {
	u, err := a.store.GetUser(a.uid)
	if err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	ctx.Reply(Result{Data: UserDTO{
		ID:           u.ID,
		Username:     u.Username,
		Nickname:     u.Nickname,
		Avatar:       u.Avatar,
		Sign:         u.Sign,
		Status:       UserStatus(u.Status),
		CreatedAt:    u.CreatedAt,
		LastOnlineAt: u.LastOnlineAt,
	}})
}

func (a *SessionActor) handleUpdateProfile(ctx actor.Context, msg UpdateProfileCmd) {
	now := time.Now().Unix()
	updates := map[string]any{"updated_at": now}
	if msg.Nickname != "" {
		updates["nickname"] = msg.Nickname
	}
	if msg.Avatar != "" {
		updates["avatar"] = msg.Avatar
	}
	if msg.Sign != "" {
		updates["sign"] = msg.Sign
	}
	if err := a.store.UpdateUser(a.uid, updates); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	ctx.Reply(Result{Data: true})
}

func (a *SessionActor) handleChangePassword(ctx actor.Context, msg ChangePasswordCmd) {
	u, err := a.store.GetUser(a.uid)
	if err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	if !checkPassword(u.Password, msg.OldPassword) {
		ctx.Reply(Result{Err: ErrIncorrectPassword})
		return
	}
	if !validatePassword(msg.NewPassword) {
		ctx.Reply(Result{Err: ErrWeakPassword})
		return
	}
	passwordHash, err := hashPassword(msg.NewPassword)
	if err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	now := time.Now().Unix()
	if err := a.store.UpdateUser(a.uid, map[string]any{"password": passwordHash, "updated_at": now}); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	ctx.Reply(Result{Data: true})
}
