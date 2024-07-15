package rpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/naiba/nezha/service/singleton"
)

type authHandler struct {
	ClientSecret string
}

func (a *authHandler) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{"client_secret": a.ClientSecret}, nil
}

func (a *authHandler) RequireTransportSecurity() bool {
	return false
}

func (a *authHandler) Check(ctx context.Context) (uint64, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return 0, status.Errorf(codes.Unauthenticated, "获取 metaData 失败")
	}

	var clientSecret string
	if value, ok := md["client_secret"]; ok {
		clientSecret = value[0]
	}

	singleton.ServerLock.RLock()
	defer singleton.ServerLock.RUnlock()
	clientID, hasID := singleton.SecretToID[clientSecret]
	if !hasID {
		return 0, status.Errorf(codes.Unauthenticated, "客户端认证失败")
	}
	_, hasServer := singleton.ServerList[clientID]
	if !hasServer {
		return 0, status.Errorf(codes.Unauthenticated, "客户端认证失败")
	}
	return clientID, nil
}
