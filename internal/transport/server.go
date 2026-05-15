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
	engine     *actor.Engine
	router     *gin.Engine
	dispatcher ws.WsDispatcher
	handlers   *httphandler.Handlers
}

func NewServer(
	engine *actor.Engine,
	handlers *httphandler.Handlers,
	dispatcher ws.WsDispatcher,
) *Server {
	s := &Server{
		engine:     engine,
		router:     gin.Default(),
		dispatcher: dispatcher,
		handlers:   handlers,
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
				convGrp.GET("", s.handlers.Conv.List)
				convGrp.POST("/private", s.handlers.Conv.CreatePrivate)
				convGrp.POST("/group", s.handlers.Conv.CreateGroup)
				convGrp.DELETE("/:id", s.handlers.Conv.Delete)
				convGrp.PUT("/:id/pin", s.handlers.Conv.Pin)
				convGrp.PUT("/:id/mute", s.handlers.Conv.Mute)
				convGrp.PUT("/:id/read", s.handlers.Conv.Read)
				convGrp.PUT("/read-all", s.handlers.Conv.ReadAll)
				convGrp.GET("/:id/members", s.handlers.Conv.Members)
				convGrp.POST("/:id/members", s.handlers.Conv.AddMember)
				convGrp.DELETE("/:id/members/:uid", s.handlers.Conv.RemoveMember)
				convGrp.DELETE("/:id/leave", s.handlers.Conv.Leave)
				convGrp.PUT("/:id/members/:uid/role", s.handlers.Conv.SetRole)
				convGrp.PUT("/:id/owner", s.handlers.Conv.TransferOwner)
				convGrp.DELETE("/:id/dissolve", s.handlers.Conv.Dissolve)
				convGrp.PUT("/:id/info", s.handlers.Conv.UpdateInfo)
				convGrp.GET("/:id/messages", s.handlers.Msg.List)
			}

			msgGrp := authGrp.Group("/messages")
			{
				msgGrp.GET("/search", s.handlers.Msg.Search)
			}

			friendGrp := authGrp.Group("/friends")
			{
				friendGrp.POST("/request", s.handlers.Friend.SendRequest)
				friendGrp.GET("/requests/incoming", s.handlers.Friend.ListIncoming)
				friendGrp.GET("/requests/outgoing", s.handlers.Friend.ListOutgoing)
				friendGrp.PUT("/requests/:req_id", s.handlers.Friend.HandleRequest)
				friendGrp.DELETE("/:friend_uid", s.handlers.Friend.DeleteFriend)
				friendGrp.GET("", s.handlers.Friend.ListFriends)
				friendGrp.PUT("/:friend_uid/remark", s.handlers.Friend.UpdateRemark)
				friendGrp.PUT("/:friend_uid/group", s.handlers.Friend.MoveGroup)
			}

			friendGroupGrp := authGrp.Group("/friend/groups")
			{
				friendGroupGrp.GET("", s.handlers.Friend.ListGroups)
				friendGroupGrp.POST("", s.handlers.Friend.CreateGroup)
				friendGroupGrp.PUT("/:group_id", s.handlers.Friend.RenameGroup)
				friendGroupGrp.DELETE("/:group_id", s.handlers.Friend.DeleteGroup)
				friendGroupGrp.PUT("/sort", s.handlers.Friend.SortGroups)
			}

			userGrp := authGrp.Group("/user")
			{
				userGrp.GET("/profile", s.handlers.User.Profile)
				userGrp.PUT("/profile", s.handlers.User.UpdateProfile)
				userGrp.PUT("/password", s.handlers.User.ChangePassword)
			}

			usersGrp := authGrp.Group("/users")
			{
				usersGrp.GET("/search", s.handlers.User.Search)
				usersGrp.GET("/:id", s.handlers.User.GetUser)
			}
		}

		api.POST("/auth/register", s.handlers.User.Register)
		api.POST("/auth/login", s.handlers.User.Login)
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
