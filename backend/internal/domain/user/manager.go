package user

import (
	"fmt"
	"regexp"
	"time"

	"qim/internal/actor"
	"qim/internal/dal"
	"qim/internal/eventbus"
)

var usernamePattern = regexp.MustCompile(`^\d+$`)

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
	case GetUserByUsernameCmd:
		a.handleGetUserByUsername(ctx, msg)
	case UpdateLastOnlineCmd:
		a.handleUpdateLastOnline(msg)
	case eventbus.EventEnvelope:
		a.handleEvent(msg.Event)
	}
}

func (a *ManagerActor) handleRegister(ctx actor.Context, msg RegisterCmd) {
	if !usernamePattern.MatchString(msg.Username) {
		ctx.Reply(Result{Err: ErrInvalidUsername})
		return
	}
	if !validatePassword(msg.Password) {
		ctx.Reply(Result{Err: ErrWeakPassword})
		return
	}
	now := time.Now().Unix()
	passwordHash, err := hashPassword(msg.Password)
	if err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	u := &dal.User{
		Username:     msg.Username,
		Password:     passwordHash,
		Nickname:     msg.Nickname,
		CreatedAt:    now,
		UpdatedAt:    now,
		LastOnlineAt: now,
	}
	if err := a.store.CreateUser(u); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	ctx.Reply(Result{Data: UserDTO{
		ID:           u.ID,
		Username:     u.Username,
		Nickname:     u.Nickname,
		CreatedAt:    u.CreatedAt,
		LastOnlineAt: u.LastOnlineAt,
	}})
}

func (a *ManagerActor) handleLogin(ctx actor.Context, msg LoginCmd) {
	u, err := a.store.Authenticate(msg.Username, msg.Password)
	if err != nil {
		ctx.Reply(Result{Err: ErrInvalidCredentials})
		return
	}

	name := fmt.Sprintf("session:%d", u.ID)
	a.engine.GetOrCreate(name, func() actor.Actor {
		return NewSessionActor(u.ID, a.store, a.engine)
	})

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

func (a *ManagerActor) handleSearchUsers(ctx actor.Context, msg SearchUsersCmd) {
	users, err := a.store.SearchUsers(msg.Keyword, 20)
	if err != nil {
		ctx.Reply(Result{Err: err})
		return
	}

	dtos := make([]UserDTO, 0, len(users))
	for _, u := range users {
		dtos = append(dtos, UserDTO{
			ID:           u.ID,
			Username:     u.Username,
			Nickname:     u.Nickname,
			Avatar:       u.Avatar,
			Sign:         u.Sign,
			LastOnlineAt: u.LastOnlineAt,
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

func (a *ManagerActor) handleGetUserByUsername(ctx actor.Context, msg GetUserByUsernameCmd) {
	u, err := a.store.GetUserByUsername(msg.Username)
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

func (a *ManagerActor) handleUpdateLastOnline(msg UpdateLastOnlineCmd) {
	if msg.UID == 0 || msg.At == 0 {
		return
	}
	_ = a.store.UpdateUser(msg.UID, map[string]any{"last_online_at": msg.At})
}

func (a *ManagerActor) handleEvent(event eventbus.Event) {
	switch e := event.(type) {
	case UserOnlineEvent:
		a.handleUpdateLastOnline(UpdateLastOnlineCmd{UID: e.UID, At: e.At})
	}
}
