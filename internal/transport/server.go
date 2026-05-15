package transport

import (
	"fmt"
	"net/http"
	"time"

	"qim/internal/actor"
	"qim/internal/middleware"
	httphandler "qim/internal/transport/http"
	"qim/internal/transport/ws"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Server struct {
	engine        *actor.Engine
	router        *gin.Engine
	convHandler   *httphandler.ConversationHandler
	userHandler   *httphandler.UserHandler
	msgHandler    *httphandler.MessageHandler
	friendHandler *httphandler.FriendHandler
	dispatcher    ws.WsDispatcher
}

func NewServer(
	engine *actor.Engine,
	convHandler *httphandler.ConversationHandler,
	userHandler *httphandler.UserHandler,
	msgHandler *httphandler.MessageHandler,
	friendHandler *httphandler.FriendHandler,
	dispatcher ws.WsDispatcher,
) *Server {
	s := &Server{
		engine:        engine,
		router:        gin.Default(),
		convHandler:   convHandler,
		userHandler:   userHandler,
		msgHandler:    msgHandler,
		friendHandler: friendHandler,
		dispatcher:    dispatcher,
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	auth := middleware.Auth()
	cors := middleware.CORS()

	s.router.Use(cors)

	api := s.router.Group("/api")
	{
		authGrp := api.Group("", auth)
		{
			convGrp := authGrp.Group("/conversations")
			{
				convGrp.GET("", s.convHandler.List)
				convGrp.POST("/private", s.convHandler.CreatePrivate)
				convGrp.POST("/group", s.convHandler.CreateGroup)
				convGrp.DELETE("/:id", s.convHandler.Delete)
				convGrp.PUT("/:id/pin", s.convHandler.Pin)
				convGrp.PUT("/:id/mute", s.convHandler.Mute)
				convGrp.PUT("/:id/read", s.convHandler.Read)
				convGrp.PUT("/read-all", s.convHandler.ReadAll)
				convGrp.GET("/:id/members", s.convHandler.Members)
				convGrp.POST("/:id/members", s.convHandler.AddMember)
				convGrp.DELETE("/:id/members/:uid", s.convHandler.RemoveMember)
				convGrp.DELETE("/:id/leave", s.convHandler.Leave)
				convGrp.PUT("/:id/members/:uid/role", s.convHandler.SetRole)
				convGrp.PUT("/:id/owner", s.convHandler.TransferOwner)
				convGrp.DELETE("/:id/dissolve", s.convHandler.Dissolve)
				convGrp.PUT("/:id/info", s.convHandler.UpdateInfo)
				convGrp.GET("/:id/messages", s.msgHandler.List)
			}

			msgGrp := authGrp.Group("/messages")
			{
				msgGrp.GET("/search", s.msgHandler.Search)
			}

			friendGrp := authGrp.Group("/friends")
			{
				friendGrp.POST("/request", s.friendHandler.SendRequest)
				friendGrp.GET("/requests/incoming", s.friendHandler.ListIncoming)
				friendGrp.GET("/requests/outgoing", s.friendHandler.ListOutgoing)
				friendGrp.PUT("/requests/:req_id", s.friendHandler.HandleRequest)
				friendGrp.DELETE("/:friend_uid", s.friendHandler.DeleteFriend)
				friendGrp.GET("", s.friendHandler.ListFriends)
				friendGrp.PUT("/:friend_uid/remark", s.friendHandler.UpdateRemark)
				friendGrp.PUT("/:friend_uid/group", s.friendHandler.MoveGroup)
			}

			friendGroupGrp := authGrp.Group("/friend/groups")
			{
				friendGroupGrp.GET("", s.friendHandler.ListGroups)
				friendGroupGrp.POST("", s.friendHandler.CreateGroup)
				friendGroupGrp.PUT("/:group_id", s.friendHandler.RenameGroup)
				friendGroupGrp.DELETE("/:group_id", s.friendHandler.DeleteGroup)
				friendGroupGrp.PUT("/sort", s.friendHandler.SortGroups)
			}

			userGrp := authGrp.Group("/user")
			{
				userGrp.GET("/profile", s.userHandler.Profile)
				userGrp.PUT("/profile", s.userHandler.UpdateProfile)
				userGrp.PUT("/password", s.userHandler.ChangePassword)
			}

			usersGrp := authGrp.Group("/users")
			{
				usersGrp.GET("/search", s.userHandler.Search)
				usersGrp.GET("/:id", s.userHandler.GetUser)
			}
		}

		api.POST("/auth/register", s.userHandler.Register)
		api.POST("/auth/login", s.userHandler.Login)
	}

	s.router.GET("/ws", auth, s.HandleWebSocket)
}

func (s *Server) HandleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	uid := c.GetUint64("uid")
	a := ws.NewGatewayActor(uid, conn, s.engine, s.dispatcher)
	name := fmt.Sprintf("gw:%s", generateConnID())
	s.engine.Spawn(name, a)
}

func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}

func generateConnID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
