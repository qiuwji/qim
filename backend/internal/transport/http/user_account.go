package http

import (
	"fmt"
	"strings"

	"qim/internal/domain/user"
	"qim/internal/pkg/resp"
	"qim/internal/service"

	"github.com/gin-gonic/gin"
)

func resolveUsername(c *gin.Context, svc *service.UserService, username string) (uint64, bool) {
	username = strings.TrimSpace(username)
	if username == "" {
		resp.Fail(c, badRequest(fmt.Errorf("username is required")))
		return 0, false
	}
	r, err := svc.AskManager(user.GetUserByUsernameCmd{Username: username})
	if err != nil {
		logHTTPError(c, "resolve username failed", err)
		resp.Fail(c, internalError(err))
		return 0, false
	}
	if r.Err != nil {
		logHTTPError(c, "username not found", r.Err)
		resp.Fail(c, badRequest(fmt.Errorf("user not found")))
		return 0, false
	}
	u, ok := r.Data.(user.UserDTO)
	if !ok || u.ID == 0 {
		resp.Fail(c, internalError(nil))
		return 0, false
	}
	return u.ID, true
}
