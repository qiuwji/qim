package agent

import (
	"encoding/json"
	"fmt"
	"time"

	"qim/internal/actor"
	"qim/internal/dal"
	"qim/internal/domain/conversation"
	"qim/internal/service"
)

const askTimeout = 5 * time.Second

const (
	permissionConversationRead = "conversation:read"
	permissionFriendRead       = "friend:read"
	permissionGroupRead        = "group:read"
	permissionMessageSearch    = "message:search"
)

// ToolSpec defines a single MCP tool — its metadata, permission requirements,
// and handler. Each tool self-registers via init() + autoRegister.
type ToolSpec struct {
	Name         string
	Desc         string
	Permissions  []string // all required permissions; checked before handler
	NeedConv     bool     // if true, validate bot is member of conversation_id in args
	NeedApproval bool     // if true, require human approval before executing handler
	Handle       func(ctx ToolContext, args json.RawMessage) (any, error)
}

// ToolContext carries pre-validated identity info into the handler.
type ToolContext struct {
	SessionID string
	BotUID    uint64
	ConvID    uint64
}

// --- Auto-registration infrastructure ---

type toolFactory func(*ToolRouter) []ToolSpec

var toolFactories []toolFactory

// autoRegister is called from init() in each tool file to register a group of tools.
func autoRegister(f toolFactory) {
	toolFactories = append(toolFactories, f)
}

// --- ToolRouter ---

type ToolRouter struct {
	registry  map[string]*ToolSpec
	convSvc   *service.ConvService
	msgSvc    *service.MsgService
	friendSvc *service.FriendService
	userSvc   *service.UserService
	botStore  dal.BotStore
	userStore dal.UserStore
	hubRef    *actor.ActorRef
	gwRef     *actor.ActorRef
	presence  *actor.ActorRef
	approvals *ApprovalManager
}

func NewToolRouter(
	convSvc *service.ConvService,
	msgSvc *service.MsgService,
	friendSvc *service.FriendService,
	userSvc *service.UserService,
	botStore dal.BotStore,
	userStore dal.UserStore,
	hubRef *actor.ActorRef,
	gwRef *actor.ActorRef,
	presenceRef *actor.ActorRef,
	approvals *ApprovalManager,
) *ToolRouter {
	r := &ToolRouter{
		registry:  make(map[string]*ToolSpec),
		convSvc:   convSvc,
		msgSvc:    msgSvc,
		friendSvc: friendSvc,
		userSvc:   userSvc,
		botStore:  botStore,
		userStore: userStore,
		hubRef:    hubRef,
		gwRef:     gwRef,
		presence:  presenceRef,
		approvals: approvals,
	}
	for _, f := range toolFactories {
		for _, spec := range f(r) {
			s := spec
			r.registry[s.Name] = &s
		}
	}
	return r
}

// Resolve looks up the tool, runs generic permission/conversation/approval checks, then dispatches.
func (r *ToolRouter) Resolve(sessionID string, toolName string, arguments json.RawMessage) (any, error) {
	spec, ok := r.registry[toolName]
	if !ok {
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}

	botUID := extractBotUID(arguments)
	convID := extractConvID(arguments)

	// 1. Permission check
	for _, perm := range spec.Permissions {
		if err := r.checkPermission(botUID, perm); err != nil {
			return nil, err
		}
	}

	// 2. Conversation membership check
	if spec.NeedConv {
		if err := r.validateBotConversation(botUID, convID); err != nil {
			return nil, err
		}
	}

	// 3. Human approval (server-enforced, agent cannot bypass)
	if spec.NeedApproval {
		if err := r.requireApproval(sessionID, botUID, convID, toolName, arguments); err != nil {
			return nil, err
		}
	}

	// 4. Execute handler
	ctx := ToolContext{SessionID: sessionID, BotUID: botUID, ConvID: convID}
	return spec.Handle(ctx, arguments)
}

// ListTools returns all registered tool definitions for the MCP tools/list response.
func (r *ToolRouter) ListTools() []map[string]any {
	tools := make([]map[string]any, 0, len(r.registry))
	for _, spec := range r.registry {
		tools = append(tools, map[string]any{
			"name":        spec.Name,
			"description": spec.Desc,
		})
	}
	return tools
}

// --- Argument extractors ---

func extractBotUID(args json.RawMessage) uint64 {
	var p struct {
		BotUID uint64 `json:"bot_uid"`
	}
	_ = json.Unmarshal(args, &p)
	return p.BotUID
}

func extractConvID(args json.RawMessage) uint64 {
	var p struct {
		ConversationID uint64 `json:"conversation_id"`
	}
	_ = json.Unmarshal(args, &p)
	return p.ConversationID
}

// --- Permission & validation helpers ---

func (r *ToolRouter) checkPermission(botUID uint64, perm string) error {
	if _, err := r.validateBot(botUID); err != nil {
		return err
	}
	if perm == "" {
		return nil
	}
	if r.hubRef != nil {
		raw, err := r.hubRef.Ask(PermissionQuery{BotUID: botUID, Permission: perm}, askTimeout)
		if err == nil {
			if allowed, ok := raw.(bool); ok {
				if allowed {
					return nil
				}
				return ErrPermissionDenied
			}
		}
	}
	if r.botStore == nil {
		return nil
	}
	cfg, err := r.botStore.GetConfig(botUID)
	if err != nil || cfg == nil {
		return ErrBotNotFound
	}
	for _, p := range parsePermissions(cfg.Permissions) {
		if p == perm {
			return nil
		}
	}
	return ErrPermissionDenied
}

func (r *ToolRouter) validateBot(botUID uint64) (*dal.User, error) {
	if botUID == 0 {
		return nil, ErrBotNotFound
	}
	if r.botStore != nil {
		cfg, err := r.botStore.GetConfig(botUID)
		if err != nil || cfg == nil {
			return nil, ErrBotNotFound
		}
	}
	if r.userStore == nil {
		return &dal.User{ID: botUID, UserType: 1, CreatorUID: botUID}, nil
	}
	u, err := r.userStore.GetUser(botUID)
	if err != nil || u == nil || u.UserType != 1 {
		return nil, ErrBotNotFound
	}
	return u, nil
}

func (r *ToolRouter) validateBotConversation(botUID, convID uint64) error {
	if _, err := r.validateBot(botUID); err != nil {
		return err
	}
	result, err := r.convSvc.AskConv(convID, conversation.ListMembersQuery{})
	if err != nil || result.Err != nil {
		return ErrPermissionDenied
	}
	members, ok := result.Data.([]conversation.MemberDTO)
	if !ok {
		return ErrPermissionDenied
	}
	for _, m := range members {
		if m.UID == botUID {
			return nil
		}
	}
	return ErrPermissionDenied
}

func (r *ToolRouter) botOwnerUID(botUID uint64) (uint64, error) {
	bot, err := r.validateBot(botUID)
	if err != nil {
		return 0, err
	}
	if bot.CreatorUID != 0 {
		return bot.CreatorUID, nil
	}
	return botUID, nil
}

// requireApproval triggers the human-in-the-loop approval flow.
// It blocks until the bot owner approves, rejects, or timeout.
func (r *ToolRouter) requireApproval(sessionID string, botUID, convID uint64, toolName string, args json.RawMessage) error {
	if r.approvals == nil {
		return fmt.Errorf("approval manager unavailable")
	}
	ownerUID, err := r.botOwnerUID(botUID)
	if err != nil {
		return err
	}
	summary := buildApprovalSummary(toolName, args)
	result, err := r.approvals.RequestUserApprovalAndWait(sessionID, r.presence, ownerUID, botUID, convID, toolName, summary, 0)
	if err != nil {
		return err
	}
	if result["status"] != "approved" {
		return ErrApprovalDenied
	}
	return nil
}

func buildApprovalSummary(toolName string, args json.RawMessage) string {
	var p struct {
		Content string `json:"content"`
	}
	_ = json.Unmarshal(args, &p)
	if p.Content != "" && len(p.Content) > 100 {
		p.Content = p.Content[:100] + "..."
	}
	if p.Content != "" {
		return toolName + ": " + p.Content
	}
	return toolName
}
