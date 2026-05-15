package http

import (
	"fmt"
	"time"

	"qim/internal/actor"
	"qim/internal/domain/user"
	"qim/internal/pkg/resp"

	"github.com/gin-gonic/gin"
)

const userAskTimeout = 5 * time.Second

type UserHandler struct {
	engine       *actor.Engine
	newSessionFn func(uid uint64) actor.Actor
}

func NewUserHandler(engine *actor.Engine, newSessionFn func(uid uint64) actor.Actor) *UserHandler {
	return &UserHandler{engine: engine, newSessionFn: newSessionFn}
}

func (h *UserHandler) managerRef() (*actor.ActorRef, bool) {
	return h.engine.Lookup("user-manager")
}

func (h *UserHandler) sessionRef(uid uint64) (*actor.ActorRef, error) {
	name := fmt.Sprintf("session:%d", uid)
	ref, err := h.engine.GetOrCreate(name, func() actor.Actor {
		return h.newSessionFn(uid)
	})
	if err != nil {
		return nil, fmt.Errorf("session actor unavailable: %w", err)
	}
	return ref, nil
}

func (h *UserHandler) askManager(cmd any) (user.Result, error) {
	ref, ok := h.managerRef()
	if !ok {
		return user.Result{}, fmt.Errorf("user manager unavailable")
	}
	raw, err := ref.Ask(cmd, userAskTimeout)
	if err != nil {
		return user.Result{}, err
	}
	r, ok := raw.(user.Result)
	if !ok {
		return user.Result{}, fmt.Errorf("unexpected result type")
	}
	return r, nil
}

func (h *UserHandler) askSession(uid uint64, cmd any) (user.Result, error) {
	ref, err := h.sessionRef(uid)
	if err != nil {
		return user.Result{}, err
	}
	raw, err := ref.Ask(cmd, userAskTimeout)
	if err != nil {
		return user.Result{}, err
	}
	r, ok := raw.(user.Result)
	if !ok {
		return user.Result{}, fmt.Errorf("unexpected result type")
	}
	return r, nil
}

func (h *UserHandler) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Nickname string `json:"nickname"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, err.Error())
		return
	}
	r, err := h.askManager(user.RegisterCmd{
		Username: req.Username,
		Password: req.Password,
		Nickname: req.Nickname,
	})
	handleUserResult(c, r, err)
}

func (h *UserHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, err.Error())
		return
	}
	r, err := h.askManager(user.LoginCmd{
		Username: req.Username,
		Password: req.Password,
	})
	handleUserResult(c, r, err)
}

func (h *UserHandler) Profile(c *gin.Context) {
	uid := c.GetUint64("uid")
	r, err := h.askSession(uid, user.GetProfileQuery{})
	handleUserResult(c, r, err)
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	uid := c.GetUint64("uid")
	var req struct {
		Nickname string `json:"nickname"`
		Avatar   string `json:"avatar"`
		Sign     string `json:"sign"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, err.Error())
		return
	}
	r, err := h.askSession(uid, user.UpdateProfileCmd{
		Nickname: req.Nickname,
		Avatar:   req.Avatar,
		Sign:     req.Sign,
	})
	handleUserResult(c, r, err)
}

func (h *UserHandler) ChangePassword(c *gin.Context) {
	uid := c.GetUint64("uid")
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, err.Error())
		return
	}
	r, err := h.askSession(uid, user.ChangePasswordCmd{
		OldPassword: req.OldPassword,
		NewPassword: req.NewPassword,
	})
	handleUserResult(c, r, err)
}

func (h *UserHandler) Search(c *gin.Context) {
	keyword := c.Query("keyword")
	r, err := h.askManager(user.SearchUsersCmd{Keyword: keyword})
	handleUserResult(c, r, err)
}

func (h *UserHandler) GetUser(c *gin.Context) {
	uid := c.GetUint64("id")
	r, err := h.askManager(user.GetUserCmd{UID: uid})
	handleUserResult(c, r, err)
}

func handleUserResult(c *gin.Context, r user.Result, err error) {
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
