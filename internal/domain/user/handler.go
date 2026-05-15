package user

import "github.com/gin-gonic/gin"

type Handler struct {
	repo *Repo
}

func NewHandler(repo *Repo) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) Register(c *gin.Context)  {}
func (h *Handler) Login(c *gin.Context)     {}
func (h *Handler) Profile(c *gin.Context)   {}
func (h *Handler) UpdateProfile(c *gin.Context)  {}
func (h *Handler) ChangePassword(c *gin.Context) {}
func (h *Handler) Search(c *gin.Context)    {}
func (h *Handler) GetUser(c *gin.Context)   {}
