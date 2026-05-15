package conversation

import "github.com/gin-gonic/gin"

type Handler struct {
	repo *Repo
}

func NewHandler(repo *Repo) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) List(c *gin.Context)          {}
func (h *Handler) CreatePrivate(c *gin.Context) {}
func (h *Handler) Delete(c *gin.Context)        {}
func (h *Handler) Pin(c *gin.Context)           {}
func (h *Handler) Mute(c *gin.Context)          {}
func (h *Handler) Read(c *gin.Context)          {}
func (h *Handler) ReadAll(c *gin.Context)       {}
func (h *Handler) CreateGroup(c *gin.Context)   {}
func (h *Handler) Members(c *gin.Context)       {}
func (h *Handler) AddMember(c *gin.Context)     {}
func (h *Handler) RemoveMember(c *gin.Context)  {}
func (h *Handler) Leave(c *gin.Context)         {}
func (h *Handler) SetRole(c *gin.Context)       {}
func (h *Handler) TransferOwner(c *gin.Context) {}
func (h *Handler) Dissolve(c *gin.Context)      {}
func (h *Handler) UpdateInfo(c *gin.Context)    {}
