package http

import (
	"encoding/json"
	"fmt"
	"time"

	"qim/internal/dal"
	"qim/internal/domain/conversation"
	"qim/internal/domain/friend"
	"qim/internal/pkg/resp"
	"qim/internal/service"

	"github.com/gin-gonic/gin"
)

type BotHTTPHandler struct {
	userSvc   *service.UserService
	convSvc   *service.ConvService
	friendSvc *service.FriendService
	botStore  dal.BotStore
	userStore dal.UserStore
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

func (h *BotHTTPHandler) Activate(c *gin.Context) {
	creatorUID := c.GetUint64("uid")

	botUser := &dal.User{
		Username:   fmt.Sprintf("bot_%d", creatorUID),
		Password:   "",
		Nickname:   "AI Bot",
		Avatar:     "",
		UserType:   1,
		CreatorUID: creatorUID,
		Status:     0,
		CreatedAt:  time.Now().Unix(),
		UpdatedAt:  time.Now().Unix(),
		LastOnlineAt: time.Now().Unix(),
	}
	if err := h.userStore.CreateUser(botUser); err != nil {
		resp.Fail(c, internalError(err))
		return
	}

	cfg := &dal.BotConfig{
		UID:       botUser.ID,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}
	if err := h.botStore.CreateConfig(cfg); err != nil {
		resp.Fail(c, internalError(err))
		return
	}

	_, err := h.friendSvc.Ask(friend.SendRequestCmd{
		FromUID: creatorUID,
		ToUID:   botUser.ID,
	})
	if err != nil {
		resp.Fail(c, internalError(err))
		return
	}

	reqRef, err := h.friendSvc.Ref()
	if err != nil {
		resp.Fail(c, internalError(err))
		return
	}
	raw, err := reqRef.Ask(friend.ListIncomingCmd{UID: botUser.ID}, 5*time.Second)
	if err == nil {
		if result, ok := raw.(friend.Result); ok && result.Err == nil {
			if reqs, ok := result.Data.([]friend.FriendRequestDTO); ok && len(reqs) > 0 {
				reqRef.Tell(friend.HandleRequestCmd{
					UID:    botUser.ID,
					ReqID:  reqs[0].ID,
					Accept: true,
				})
			}
		}
	}

	convResult, err := h.convSvc.AskManager(conversation.CreatePrivateConvCmd{
		UID1: creatorUID,
		UID2: botUser.ID,
	})
	if err != nil || convResult.Err != nil {
		resp.Fail(c, internalError(err))
		return
	}

	convID := uint64(0)
	if dto, ok := convResult.Data.(conversation.ConversationDTO); ok {
		convID = dto.ID
	}

	resp.OK(c, map[string]any{
		"bot_user": map[string]any{
			"id":       botUser.ID,
			"nickname": botUser.Nickname,
			"avatar":   botUser.Avatar,
		},
		"conversation_id": convID,
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

	resp.OK(c, map[string]any{"status": "ok"})
}

func (h *BotHTTPHandler) NewSession(c *gin.Context) {
	creatorUID := c.GetUint64("uid")
	botUID, err := h.findBotUID(creatorUID)
	if err != nil {
		resp.Fail(c, badRequest(err))
		return
	}

	convResult, err := h.convSvc.AskManager(conversation.CreatePrivateConvCmd{
		UID1: creatorUID,
		UID2: botUID,
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
