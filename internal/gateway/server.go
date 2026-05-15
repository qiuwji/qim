package gateway

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"qim/internal/actor"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Server struct {
	engine *actor.Engine
	router *gin.Engine
}

func NewServer(engine *actor.Engine) *Server {
	s := &Server{
		engine: engine,
		router: gin.Default(),
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {}

func (s *Server) HandleWebSocket(c *gin.Context) {}

func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}
