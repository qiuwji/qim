package http

import (
	"fmt"
	"time"

	"qim/internal/actor"
	"qim/internal/domain/conversation"
	"qim/internal/pkg/resp"

	"github.com/gin-gonic/gin"
)

const convAskTimeout = 5 * time.Second

type ConversationHandler struct {
	engine    *actor.Engine
	newConvFn func(convID uint64) actor.Actor
}

func NewConversationHandler(engine *actor.Engine, newConvFn func(convID uint64) actor.Actor) *ConversationHandler {
	return &ConversationHandler{engine: engine, newConvFn: newConvFn}
}

func (h *ConversationHandler) managerRef() (*actor.ActorRef, bool) {
	return h.engine.Lookup("conv-manager")
}

func (h *ConversationHandler) convRef(convID uint64) (*actor.ActorRef, error) {
	name := fmt.Sprintf("conv:%d", convID)
	ref, err := h.engine.GetOrCreate(name, func() actor.Actor {
		return h.newConvFn(convID)
	})
	if err != nil {
		return nil, fmt.Errorf("conversation actor unavailable: %w", err)
	}
	return ref, nil
}

func (h *ConversationHandler) askManager(cmd any) (conversation.Result, error) {
	ref, ok := h.managerRef()
	if !ok {
		return conversation.Result{}, fmt.Errorf("conversation manager unavailable")
	}
	raw, err := ref.Ask(cmd, convAskTimeout)
	if err != nil {
		return conversation.Result{}, err
	}
	r, ok := raw.(conversation.Result)
	if !ok {
		return conversation.Result{}, fmt.Errorf("unexpected result type")
	}
	return r, nil
}

func (h *ConversationHandler) askConv(convID uint64, cmd any) (conversation.Result, error) {
	ref, err := h.convRef(convID)
	if err != nil {
		return conversation.Result{}, err
	}
	raw, err := ref.Ask(cmd, convAskTimeout)
	if err != nil {
		return conversation.Result{}, err
	}
	r, ok := raw.(conversation.Result)
	if !ok {
		return conversation.Result{}, fmt.Errorf("unexpected result type")
	}
	return r, nil
}

func handleResult(c *gin.Context, r conversation.Result, err error) {
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
