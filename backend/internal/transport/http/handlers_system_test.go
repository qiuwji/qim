package http

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"qim/internal/actor"
	"qim/internal/domain/conversation"
	"qim/internal/domain/friend"
	"qim/internal/domain/message"
	"qim/internal/domain/user"
	jwtpkg "qim/internal/pkg/jwt"
	"qim/internal/service"

	"github.com/gin-gonic/gin"
)

func TestHTTPHandlersSystemFlow_BitsUT(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := actor.NewEngine()
	mustSpawnHTTP(t, engine, "user-manager", httpUserActor{})
	mustSpawnHTTP(t, engine, "conv-manager", httpConvManagerActor{})
	mustSpawnHTTP(t, engine, "friend-manager", httpFriendActor{})
	mustSpawnHTTP(t, engine, "msg-store", httpMsgActor{})

	userSvc := service.NewUserService(engine, func(uid uint64) actor.Actor { return httpSessionActor{uid: uid} })
	convSvc := service.NewConvService(engine, func(convID uint64) actor.Actor { return httpConvActor{convID: convID} })
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("uid", uint64(1))
		c.Next()
	})
	userHandler := NewUserHandler(userSvc, jwtpkg.NewManager("secret", time.Hour))
	convHandler := NewConversationHandler(convSvc, userSvc)
	friendHandler := NewFriendHandler(service.NewFriendService(engine), userSvc)
	msgHandler := NewMessageHandler(service.NewMsgService(engine))

	router.POST("/api/auth/register", userHandler.Register)
	router.POST("/api/auth/login", userHandler.Login)
	router.GET("/api/user/profile", userHandler.Profile)
	router.PUT("/api/user/profile", userHandler.UpdateProfile)
	router.PUT("/api/user/password", userHandler.ChangePassword)
	router.GET("/api/users/search", userHandler.Search)
	router.GET("/api/users/:id", userHandler.GetUser)
	router.GET("/api/conversations", convHandler.List)
	router.POST("/api/conversations/private", convHandler.CreatePrivate)
	router.POST("/api/conversations/group", convHandler.CreateGroup)
	router.PUT("/api/conversations/:id/info", convHandler.UpdateInfo)
	router.GET("/api/conversations/:id/members", convHandler.Members)
	router.POST("/api/conversations/:id/members", convHandler.AddMember)
	router.DELETE("/api/conversations/:id/members/:uid", convHandler.RemoveMember)
	router.DELETE("/api/conversations/:id/leave", convHandler.Leave)
	router.PUT("/api/conversations/:id/members/:uid/role", convHandler.SetRole)
	router.PUT("/api/conversations/:id/owner", convHandler.TransferOwner)
	router.DELETE("/api/conversations/:id/dissolve", convHandler.Dissolve)
	router.PUT("/api/conversations/:id/pin", convHandler.Pin)
	router.PUT("/api/conversations/:id/mute", convHandler.Mute)
	router.PUT("/api/conversations/:id/read", convHandler.Read)
	router.PUT("/api/conversations/read-all", convHandler.ReadAll)
	router.POST("/api/friends/request", friendHandler.SendRequest)
	router.GET("/api/friends/requests/incoming", friendHandler.ListIncoming)
	router.GET("/api/friends/requests/outgoing", friendHandler.ListOutgoing)
	router.PUT("/api/friends/requests/:req_id", friendHandler.HandleRequest)
	router.DELETE("/api/friends/:friend_uid", friendHandler.DeleteFriend)
	router.GET("/api/friends", friendHandler.ListFriends)
	router.PUT("/api/friends/:friend_uid/remark", friendHandler.UpdateRemark)
	router.PUT("/api/friends/:friend_uid/group", friendHandler.MoveGroup)
	router.GET("/api/friend/groups", friendHandler.ListGroups)
	router.POST("/api/friend/groups", friendHandler.CreateGroup)
	router.PUT("/api/friend/groups/sort", friendHandler.SortGroups)
	router.PUT("/api/friend/groups/:group_id", friendHandler.RenameGroup)
	router.DELETE("/api/friend/groups/:group_id", friendHandler.DeleteGroup)
	router.GET("/api/conversations/:id/messages", msgHandler.List)
	router.GET("/api/messages/search", msgHandler.Search)

	requests := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/api/auth/register", `{"username":"alice","password":"secret","nickname":"Alice"}`},
		{http.MethodPost, "/api/auth/login", `{"username":"alice","password":"secret"}`},
		{http.MethodGet, "/api/user/profile", ``},
		{http.MethodPut, "/api/user/profile", `{"nickname":"A","avatar":"/a.png","sign":"hi"}`},
		{http.MethodPut, "/api/user/password", `{"old_password":"secret","new_password":"new"}`},
		{http.MethodGet, "/api/users/search?keyword=ali", ``},
		{http.MethodGet, "/api/users/2", ``},
		{http.MethodGet, "/api/conversations", ``},
		{http.MethodPost, "/api/conversations/private", `{"uid":2}`},
		{http.MethodPost, "/api/conversations/group", `{"name":"g","members":[2]}`},
		{http.MethodPut, "/api/conversations/1/info", `{"name":"g2"}`},
		{http.MethodGet, "/api/conversations/1/members", ``},
		{http.MethodPost, "/api/conversations/1/members", `{"uid":3,"role":0}`},
		{http.MethodDelete, "/api/conversations/1/members/3", ``},
		{http.MethodDelete, "/api/conversations/1/leave", ``},
		{http.MethodPut, "/api/conversations/1/members/2/role", `{"role":1}`},
		{http.MethodPut, "/api/conversations/1/owner", `{"new_owner_id":2}`},
		{http.MethodDelete, "/api/conversations/1/dissolve", ``},
		{http.MethodPut, "/api/conversations/1/pin", `{"pinned":true}`},
		{http.MethodPut, "/api/conversations/1/mute", `{"muted":true}`},
		{http.MethodPut, "/api/conversations/1/read", `{"seq":10}`},
		{http.MethodPut, "/api/conversations/read-all", ``},
		{http.MethodPost, "/api/friends/request", `{"to_uid":2,"message":"hi"}`},
		{http.MethodGet, "/api/friends/requests/incoming", ``},
		{http.MethodGet, "/api/friends/requests/outgoing", ``},
		{http.MethodPut, "/api/friends/requests/1", `{"action":"accept"}`},
		{http.MethodDelete, "/api/friends/2", ``},
		{http.MethodGet, "/api/friends", ``},
		{http.MethodPut, "/api/friends/2/remark", `{"remark":"bob"}`},
		{http.MethodPut, "/api/friends/2/group", `{"group_id":1}`},
		{http.MethodGet, "/api/friend/groups", ``},
		{http.MethodPost, "/api/friend/groups", `{"name":"work"}`},
		{http.MethodPut, "/api/friend/groups/1", `{"name":"friends"}`},
		{http.MethodDelete, "/api/friend/groups/1", ``},
		{http.MethodPut, "/api/friend/groups/sort", `{"groups":[{"group_id":1,"sort_order":2}]}`},
		{http.MethodGet, "/api/conversations/1/messages?seq=100&limit=10", ``},
		{http.MethodGet, "/api/messages/search?conversation_id=1&keyword=hello", ``},
	}
	for _, req := range requests {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(req.method, req.path, bytes.NewBufferString(req.body)))
		if w.Code != http.StatusOK {
			t.Fatalf("%s %s status=%d body=%s", req.method, req.path, w.Code, w.Body.String())
		}
		if !bytes.Contains(w.Body.Bytes(), []byte(`"code":"ok"`)) {
			t.Fatalf("%s %s body=%s", req.method, req.path, w.Body.String())
		}
	}
}

func TestHTTPHandleResultErrorBranches_BitsUT(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name string
		run  func(c *gin.Context)
	}{
		{"user service error", func(c *gin.Context) { handleUserResult(c, user.Result{}, errors.New("service down")) }},
		{"user domain error", func(c *gin.Context) { handleUserResult(c, user.Result{Err: user.ErrInvalidCredentials}, nil) }},
		{"conversation service error", func(c *gin.Context) { handleResult(c, conversation.Result{}, errors.New("service down")) }},
		{"conversation domain error", func(c *gin.Context) { handleResult(c, conversation.Result{Err: conversation.ErrNotMember}, nil) }},
		{"friend service error", func(c *gin.Context) { handleFriendResult(c, friend.Result{}, errors.New("service down")) }},
		{"friend domain error", func(c *gin.Context) { handleFriendResult(c, friend.Result{Err: friend.ErrNotYourRequest}, nil) }},
		{"message service error", func(c *gin.Context) { handleMsgResult(c, message.Result{}, errors.New("service down")) }},
		{"message domain error", func(c *gin.Context) { handleMsgResult(c, message.Result{Err: errors.New("domain")}, nil) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
			tt.run(c)
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
			}
			if bytes.Contains(w.Body.Bytes(), []byte(`"code":"ok"`)) {
				t.Fatalf("error branch should not return ok: %s", w.Body.String())
			}
		})
	}
}

type httpUserActor struct{}

func (httpUserActor) Receive(ctx actor.Context) {
	switch ctx.Message().(type) {
	case user.RegisterCmd, user.LoginCmd, user.GetUserCmd:
		ctx.Reply(user.Result{Data: user.UserDTO{ID: 1, Username: "alice", Nickname: "Alice"}})
	case user.SearchUsersCmd:
		ctx.Reply(user.Result{Data: []user.UserDTO{{ID: 1, Username: "alice"}}})
	default:
		ctx.Reply(user.Result{Data: true})
	}
}

type httpSessionActor struct{ uid uint64 }

func (a httpSessionActor) Receive(ctx actor.Context) {
	switch ctx.Message().(type) {
	case user.GetProfileQuery:
		ctx.Reply(user.Result{Data: user.UserDTO{ID: a.uid, Username: "alice", Nickname: "Alice"}})
	default:
		ctx.Reply(user.Result{Data: true})
	}
}

type httpConvManagerActor struct{}

func (httpConvManagerActor) Receive(ctx actor.Context) {
	switch ctx.Message().(type) {
	case conversation.ListUserConversationsCmd:
		ctx.Reply(conversation.Result{Data: []conversation.UserConvDTO{{ConversationID: 1, UnreadCount: 1}}})
	default:
		ctx.Reply(conversation.Result{Data: true})
	}
}

type httpConvActor struct{ convID uint64 }

func (a httpConvActor) Receive(ctx actor.Context) {
	switch ctx.Message().(type) {
	case conversation.ListMembersQuery:
		ctx.Reply(conversation.Result{Data: []conversation.MemberDTO{{UID: 1, Role: conversation.MemberRoleOwner}}})
	default:
		ctx.Reply(conversation.Result{Data: true})
	}
}

type httpFriendActor struct{}

func (httpFriendActor) Receive(ctx actor.Context) {
	switch ctx.Message().(type) {
	case friend.ListIncomingCmd, friend.ListOutgoingCmd:
		ctx.Reply(friend.Result{Data: []friend.FriendRequestDTO{{ID: 1, FromUID: 1, ToUID: 2}}})
	case friend.ListFriendsCmd:
		ctx.Reply(friend.Result{Data: []friend.FriendDTO{{ID: 1, FriendUID: 2}}})
	default:
		ctx.Reply(friend.Result{Data: true})
	}
}

type httpMsgActor struct{}

func (httpMsgActor) Receive(ctx actor.Context) {
	ctx.Reply(message.Result{Data: []message.MessageDTO{{ID: 1, ConversationID: 1, Seq: 1, Content: "hello"}}})
}

func mustSpawnHTTP(t *testing.T, engine *actor.Engine, name string, a actor.Actor) {
	t.Helper()
	if _, err := engine.Spawn(name, a); err != nil {
		t.Fatalf("spawn %s: %v", name, err)
	}
}
