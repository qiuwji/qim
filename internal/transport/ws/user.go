package ws

import (
	"encoding/json"
	"fmt"

	"qim/internal/actor"
	"qim/internal/domain/user"
)

type userRouter struct {
	engine       *actor.Engine
	newSessionFn func(uid uint64) actor.Actor
}

func (r *userRouter) dispatch(uid uint64, action string, data json.RawMessage) WsResponse {
	cmd, ref, fn, err := r.resolve(uid, action, data)
	if err != nil {
		return errReply(action, err.Error())
	}
	return fn(ref, cmd, action)
}

func (r *userRouter) resolve(uid uint64, action string, data json.RawMessage) (any, *actor.ActorRef, dispatchFn, error) {
	switch action {
	case "register":
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Nickname string `json:"nickname"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, nil, nil, err
		}
		ref, ok := r.engine.Lookup("user-manager")
		if !ok {
			return nil, nil, nil, fmt.Errorf("user-manager unavailable")
		}
		return user.RegisterCmd{Username: req.Username, Password: req.Password, Nickname: req.Nickname}, ref, askDispatch, nil
	case "login":
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, nil, nil, err
		}
		ref, ok := r.engine.Lookup("user-manager")
		if !ok {
			return nil, nil, nil, fmt.Errorf("user-manager unavailable")
		}
		return user.LoginCmd{Username: req.Username, Password: req.Password}, ref, askDispatch, nil
	case "search":
		var req struct {
			Keyword string `json:"keyword"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, nil, nil, err
		}
		ref, ok := r.engine.Lookup("user-manager")
		if !ok {
			return nil, nil, nil, fmt.Errorf("user-manager unavailable")
		}
		return user.SearchUsersCmd{Keyword: req.Keyword}, ref, askDispatch, nil
	case "get":
		var req struct {
			UID uint64 `json:"uid"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, nil, nil, err
		}
		ref, ok := r.engine.Lookup("user-manager")
		if !ok {
			return nil, nil, nil, fmt.Errorf("user-manager unavailable")
		}
		return user.GetUserCmd{UID: req.UID}, ref, askDispatch, nil
	case "profile":
		ref, err := r.sessionRef(uid)
		if err != nil {
			return nil, nil, nil, err
		}
		return user.GetProfileQuery{}, ref, askDispatch, nil
	case "update_profile":
		var req struct {
			Nickname string `json:"nickname"`
			Avatar   string `json:"avatar"`
			Sign     string `json:"sign"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, nil, nil, err
		}
		ref, err := r.sessionRef(uid)
		if err != nil {
			return nil, nil, nil, err
		}
		return user.UpdateProfileCmd{Nickname: req.Nickname, Avatar: req.Avatar, Sign: req.Sign}, ref, tellDispatch, nil
	case "change_password":
		var req struct {
			OldPassword string `json:"old_password"`
			NewPassword string `json:"new_password"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, nil, nil, err
		}
		ref, err := r.sessionRef(uid)
		if err != nil {
			return nil, nil, nil, err
		}
		return user.ChangePasswordCmd{OldPassword: req.OldPassword, NewPassword: req.NewPassword}, ref, tellDispatch, nil
	default:
		return nil, nil, nil, fmt.Errorf("unknown user action: %s", action)
	}
}

func (r *userRouter) sessionRef(uid uint64) (*actor.ActorRef, error) {
	name := fmt.Sprintf("session:%d", uid)
	ref, err := r.engine.GetOrCreate(name, func() actor.Actor {
		return r.newSessionFn(uid)
	})
	if err != nil {
		return nil, fmt.Errorf("session actor unavailable: %w", err)
	}
	return ref, nil
}
