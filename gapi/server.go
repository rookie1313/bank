package gapi

import (
	db "bank/db/sqlc"
	"bank/pb"
	"bank/token"
	"bank/util"
	"fmt"
)

// Server serves requests for our banking service.
type Server struct {
	pb.UnimplementedBankServer
	config     util.Config
	store      db.Store
	tokenMaker token.Maker
}

// NewServer creates a new gRPC server.
func NewServer(config util.Config, store db.Store) (*Server, error) {
	tokenMaker, err := token.NewJWTMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("can not create token maker: %w", err)
	}
	server := &Server{
		store:      store,
		config:     config,
		tokenMaker: tokenMaker,
	}

	return server, nil
}
