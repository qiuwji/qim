package user

import (
	"fmt"
	"time"

	"qim/internal/actor"
	"qim/internal/dal"
)

type ManagerActor struct {
	store  dal.UserStore
	engine *actor.Engine
}

func NewManagerActor(store dal.UserStore, engine *actor.Engine) *ManagerActor {
	return &ManagerActor{store: store, engine: engine}
}

func (a *ManagerActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case RegisterCmd:
		a.handleRegister(ctx, msg)
	case LoginCmd:
		a.handleLogin(ctx, msg)
	case SearchUsersCmd:
		a.handleSearchUsers(ctx, msg)
	case GetUserCmd:
		a.handleGetUser(ctx, msg)
	}
}

func (a *ManagerActor) handleRegister(ctx actor.Context, msg RegisterCmd) {
	now := time.Now().Unix()
	u := &dal.User{
		Username:  msg.Username,
		Password:  msg.Password,
		Nickname:  msg.Nickname,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := a.store.CreateUser(u); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	ctx.Reply(Result{Data: UserDTO{
		ID:        u.ID,
		Username:  u.Username,
		Nickname:  u.Nickname,
		CreatedAt: u.CreatedAt,
	}})
}

func (a *ManagerActor) handleLogin(ctx actor.Context, msg LoginCmd) {
	u, err := a.store.Authenticate(msg.Username, msg.Password)
	if err != nil {
		ctx.Reply(Result{Err: fmt.Errorf("invalid credentials")})
		return
	}

	name := fmt.Sprintf("session:%d", u.ID)
	a.engine.GetOrCreate(name, func() actor.Actor {
		return NewSessionActor(u.ID, a.store, a.engine)
	})

	ctx.Reply(Result{Data: UserDTO{
		ID:        u.ID,
		Username:  u.Username,
		Nickname:  u.Nickname,
		Avatar:    u.Avatar,
		Sign:      u.Sign,
		Status:    UserStatus(u.Status),
		CreatedAt: u.CreatedAt,
	}})
}

func (a *ManagerActor) handleSearchUsers(ctx actor.Context, msg SearchUsersCmd) {
	users, err := a.store.SearchUsers(msg.Keyword, 20)
	if err != nil {
		ctx.Reply(Result{Err: err})
		return
	}

	dtos := make([]UserDTO, 0, len(users))
	for _, u := range users {
		dtos = append(dtos, UserDTO{
			ID:       u.ID,
			Username: u.Username,
			Nickname: u.Nickname,
			Avatar:   u.Avatar,
			Sign:     u.Sign,
		})
	}
	ctx.Reply(Result{Data: dtos})
}

func (a *ManagerActor) handleGetUser(ctx actor.Context, msg GetUserCmd) {
	u, err := a.store.GetUser(msg.UID)
	if err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	ctx.Reply(Result{Data: UserDTO{
		ID:        u.ID,
		Username:  u.Username,
		Nickname:  u.Nickname,
		Avatar:    u.Avatar,
		Sign:      u.Sign,
		Status:    UserStatus(u.Status),
		CreatedAt: u.CreatedAt,
	}})
}
