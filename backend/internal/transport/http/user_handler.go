package http

import (
	"qim/internal/domain/user"
	jwtpkg "qim/internal/pkg/jwt"
	"qim/internal/pkg/resp"
	"qim/internal/service"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	svc *service.UserService
	jwt *jwtpkg.Manager
}

func NewUserHandler(svc *service.UserService, jwt *jwtpkg.Manager) *UserHandler {
	return &UserHandler{svc: svc, jwt: jwt}
}

func handleUserResult(c *gin.Context, r user.Result, err error) {
	if err != nil {
		logHTTPError(c, "user request failed", err)
		resp.Fail(c, internalError(err))
		return
	}
	if r.Err != nil {
		logHTTPError(c, "user domain error", r.Err)
		resp.Fail(c, r.Err)
		return
	}
	resp.OK(c, r.Data)
}

func (h *UserHandler) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Nickname string `json:"nickname"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, badRequest(err))
		return
	}
	r, err := h.svc.AskManager(user.RegisterCmd{
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
		resp.Fail(c, badRequest(err))
		return
	}
	r, err := h.svc.AskManager(user.LoginCmd{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil || r.Err != nil {
		handleUserResult(c, r, err)
		return
	}
	u, ok := r.Data.(user.UserDTO)
	if !ok {
		resp.Fail(c, internalError(nil))
		return
	}
	token, err := h.jwt.Generate(u.ID, u.Username)
	if err != nil {
		resp.Fail(c, internalError(err))
		return
	}
	resp.OK(c, user.LoginResult{Token: token, User: u})
}

func (h *UserHandler) Profile(c *gin.Context) {
	uid := c.GetUint64("uid")
	r, err := h.svc.AskSession(uid, user.GetProfileQuery{})
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
		resp.Fail(c, badRequest(err))
		return
	}
	r, err := h.svc.AskSession(uid, user.UpdateProfileCmd{
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
		resp.Fail(c, badRequest(err))
		return
	}
	r, err := h.svc.AskSession(uid, user.ChangePasswordCmd{
		OldPassword: req.OldPassword,
		NewPassword: req.NewPassword,
	})
	handleUserResult(c, r, err)
}

func (h *UserHandler) Search(c *gin.Context) {
	keyword := c.Query("keyword")
	r, err := h.svc.AskManager(user.SearchUsersCmd{Keyword: keyword})
	handleUserResult(c, r, err)
}

func (h *UserHandler) GetUser(c *gin.Context) {
	uid, ok := paramUint(c, "id")
	if !ok {
		return
	}
	r, err := h.svc.AskManager(user.GetUserCmd{UID: uid})
	handleUserResult(c, r, err)
}
