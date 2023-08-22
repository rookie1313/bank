package gapi

import (
	"bank/token"
	"context"
	"fmt"
	"google.golang.org/grpc/metadata"
	"strings"
)

const (
	authorizationHeader     = "authorization"
	authorizationTypeBearer = "bearer"
)

func (server *Server) authorizeUser(ctx context.Context) (*token.Payload, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("missing metadata")
	}

	value := md.Get(authorizationHeader)
	if len(value) <= 0 {
		return nil, fmt.Errorf("missing authorization header")
	}

	header := value[0]
	fields := strings.Fields(header)
	if len(fields) < 2 {
		return nil, fmt.Errorf("invalid authorization header fomat")
	}

	authType := strings.ToLower(fields[0])
	if authType != authorizationTypeBearer {
		return nil, fmt.Errorf("unsupported authorization type %s", authType)
	}

	accessToken := fields[1]
	payload, err := server.tokenMaker.VerifyToken(accessToken)
	if err != nil {
		return nil, fmt.Errorf("invalid access token %s", accessToken)
	}

	return payload, nil
}
