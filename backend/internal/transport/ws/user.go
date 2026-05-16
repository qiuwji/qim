package ws

import (
	"encoding/json"
	"fmt"

	"qim/internal/actor"
	"qim/internal/domain/user"
	"qim/internal/service"
)

type userRouter struct {
	svc *service.UserService
}

func (r *userRouter) dispatch(uid uint64, action string, data json.RawMessage) WsResponse {
	cmd, ref, fn, err := r.resolve(uid, action, data)
	if err != nil {
		return errReply(action, err)
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
		ref, err := r.svc.ManagerRef()
		if err != nil {
			return nil, nil, nil, err
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
		ref, err := r.svc.ManagerRef()
		if err != nil {
			return nil, nil, nil, err
		}
		return user.LoginCmd{Username: req.Username, Password: req.Password}, ref, askDispatch, nil
	case "search":
		var req struct {
			Keyword string `json:"keyword"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, nil, nil, err
		}
		ref, err := r.svc.ManagerRef()
		if err != nil {
			return nil, nil, nil, err
		}
		return user.SearchUsersCmd{Keyword: req.Keyword}, ref, askDispatch, nil
	case "get":
		var req struct {
			UID uint64 `json:"uid"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, nil, nil, err
		}
		ref, err := r.svc.ManagerRef()
		if err != nil {
			return nil, nil, nil, err
		}
		return user.GetUserCmd{UID: req.UID}, ref, askDispatch, nil
	case "profile":
		ref, err := r.svc.SessionRef(uid)
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
		ref, err := r.svc.SessionRef(uid)
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
		ref, err := r.svc.SessionRef(uid)
		if err != nil {
			return nil, nil, nil, err
		}
		return user.ChangePasswordCmd{OldPassword: req.OldPassword, NewPassword: req.NewPassword}, ref, tellDispatch, nil
	default:
		return nil, nil, nil, fmt.Errorf("unknown user action: %s", action)
	}
}
