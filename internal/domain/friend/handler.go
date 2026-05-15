package friend

import "github.com/gin-gonic/gin"

type Handler struct {
	repo *Repo
}

func NewHandler(repo *Repo) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) SendRequest(c *gin.Context)      {}
func (h *Handler) ListIncoming(c *gin.Context)     {}
func (h *Handler) ListOutgoing(c *gin.Context)     {}
func (h *Handler) HandleRequest(c *gin.Context)    {}
func (h *Handler) DeleteFriend(c *gin.Context)     {}
func (h *Handler) ListFriends(c *gin.Context)      {}
func (h *Handler) UpdateRemark(c *gin.Context)     {}
func (h *Handler) MoveGroup(c *gin.Context)        {}
func (h *Handler) ListGroups(c *gin.Context)       {}
func (h *Handler) CreateGroup(c *gin.Context)      {}
func (h *Handler) RenameGroup(c *gin.Context)      {}
func (h *Handler) DeleteGroup(c *gin.Context)      {}
func (h *Handler) SortGroups(c *gin.Context)       {}
