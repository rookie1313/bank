package gapi

import (
	"context"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

const (
	grpcGatewayUserAgentHeader = "grpcgateway-user-agent"
	userAgentHeader            = "user-agent"
	xForwardedForHeader        = "x-forwarded-for"
)

type Metadata struct {
	UserAgent string
	ClientIp  string
}

func (server *Server) extractMetaData(ctx context.Context) *Metadata {
	metaData := &Metadata{}
	if mD, ok := metadata.FromIncomingContext(ctx); ok {
		if userAgent := mD.Get(grpcGatewayUserAgentHeader); len(userAgent) > 0 {
			metaData.UserAgent = userAgent[0]
		}
		if userAgent := mD.Get(userAgentHeader); len(userAgent) > 0 {
			metaData.UserAgent = userAgent[0]
		}
		if clientIPs := mD.Get(xForwardedForHeader); len(clientIPs) > 0 {
			metaData.ClientIp = clientIPs[0]
		}
	}

	if p, ok := peer.FromContext(ctx); ok {
		metaData.ClientIp = p.Addr.String()
	}

	return metaData
}
