package message

import "github.com/gin-gonic/gin"

type Handler struct {
	repo *Repo
}

func NewHandler(repo *Repo) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) List(c *gin.Context)   {}
func (h *Handler) Search(c *gin.Context) {}
