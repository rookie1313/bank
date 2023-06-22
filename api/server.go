package api

import (
	db "bank/db/sqlc"
	"github.com/gin-gonic/gin"
)

// Server serves HTTP requests for our banking service.
type Server struct {
	store  *db.Store
	router *gin.Engine
}

func NewServer(store *db.Store) *Server {
	server := &Server{store: store}
	router := gin.Default()

	router.POST("/accounts", server.createAccount)
	router.GET("/accounts/:id", server.getAccount)

	server.router = router
	return server
}
func (server *Server) Start(addr string) error {
	return server.router.Run(addr)
}
func errResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}
