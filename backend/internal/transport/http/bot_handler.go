package http

import (
	"encoding/json"
	"fmt"

	"qim/internal/actor"
	"qim/internal/dal"
	"qim/internal/domain/conversation"
	"qim/internal/pkg/resp"
	"qim/internal/service"
	agenttransport "qim/internal/transport/agent"

	"github.com/gin-gonic/gin"
)

type BotHTTPHandler struct {
	userSvc   *service.UserService
	convSvc   *service.ConvService
	friendSvc *service.FriendService
	botStore  dal.BotStore
	userStore dal.UserStore
	agentHub  *actor.ActorRef
}

func NewBotHTTPHandler(
	userSvc *service.UserService,
	convSvc *service.ConvService,
	friendSvc *service.FriendService,
	botStore dal.BotStore,
	userStore dal.UserStore,
) *BotHTTPHandler {
	return &BotHTTPHandler{
		userSvc:   userSvc,
		convSvc:   convSvc,
		friendSvc: friendSvc,
		botStore:  botStore,
		userStore: userStore,
	}
}

func (h *BotHTTPHandler) SetAgentHubRef(ref *actor.ActorRef) {
	h.agentHub = ref
}

func (h *BotHTTPHandler) Activate(c *gin.Context) {
	creatorUID := c.GetUint64("uid")
	result, err := h.botStore.ActivateBotTx(creatorUID)
	if err != nil {
		resp.Fail(c, internalError(err))
		return
	}

	resp.OK(c, map[string]any{
		"bot_user": map[string]any{
			"id":       result.BotUser.ID,
			"username": result.BotUser.Username,
			"nickname": result.BotUser.Nickname,
			"avatar":   result.BotUser.Avatar,
		},
		"conversation_id": result.ConversationID,
	})
}

func (h *BotHTTPHandler) GetConfig(c *gin.Context) {
	creatorUID := c.GetUint64("uid")
	botUID, err := h.findBotUID(creatorUID)
	if err != nil {
		resp.Fail(c, badRequest(err))
		return
	}

	cfg, err := h.botStore.GetConfig(botUID)
	if err != nil {
		resp.Fail(c, internalError(err))
		return
	}

	resp.OK(c, map[string]any{
		"uid":         cfg.UID,
		"permissions": parsePermissions(cfg.Permissions),
	})
}

func (h *BotHTTPHandler) UpdateConfig(c *gin.Context) {
	creatorUID := c.GetUint64("uid")

	var req struct {
		Permissions []string `json:"permissions"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, badRequest(err))
		return
	}

	botUID, err := h.findBotUID(creatorUID)
	if err != nil {
		resp.Fail(c, badRequest(err))
		return
	}

	if err := h.botStore.UpdatePermissions(botUID, req.Permissions); err != nil {
		resp.Fail(c, internalError(err))
		return
	}
	if h.agentHub != nil {
		_ = h.agentHub.Tell(agenttransport.RefreshPermissionsCmd{BotUID: botUID})
	}

	resp.OK(c, map[string]any{"status": "ok"})
}

func (h *BotHTTPHandler) NewSession(c *gin.Context) {
	creatorUID := c.GetUint64("uid")
	botUID, err := h.findBotUID(creatorUID)
	if err != nil {
		resp.Fail(c, badRequest(err))
		return
	}

	convResult, err := h.convSvc.AskManager(conversation.CreateBotSessionCmd{
		OwnerUID: creatorUID,
		BotUID:   botUID,
	})
	if err != nil || convResult.Err != nil {
		resp.Fail(c, internalError(err))
		return
	}

	convID := uint64(0)
	if dto, ok := convResult.Data.(conversation.ConversationDTO); ok {
		convID = dto.ID
	}

	resp.OK(c, map[string]any{"conversation_id": convID})
}

func (h *BotHTTPHandler) findBotUID(creatorUID uint64) (uint64, error) {
	users, err := h.userStore.SearchUsers("", 1000)
	if err != nil {
		return 0, err
	}
	for _, u := range users {
		if u.UserType == 1 && u.CreatorUID == creatorUID {
			return u.ID, nil
		}
	}
	return 0, fmt.Errorf("bot not found")
}

func parsePermissions(data string) []string {
	if data == "" || data == "[]" {
		return nil
	}
	var perms []string
	if err := json.Unmarshal([]byte(data), &perms); err != nil {
		return nil
	}
	return perms
}
