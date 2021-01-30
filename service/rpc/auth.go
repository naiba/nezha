package rpc

import (
	"context"

	"github.com/naiba/nezha/service/dao"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type AuthHandler struct {
	ClientSecret string
}

func (a *AuthHandler) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{"client_secret": a.ClientSecret}, nil
}

func (a *AuthHandler) RequireTransportSecurity() bool {
	return !dao.Conf.Debug
}

func (a *AuthHandler) Check(ctx context.Context) (uint64, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return 0, status.Errorf(codes.Unauthenticated, "获取 metaData 失败")
	}

	var clientSecret string
	if value, ok := md["client_secret"]; ok {
		clientSecret = value[0]
	}

	dao.ServerLock.RLock()
	defer dao.ServerLock.RUnlock()
	clientID, hasID := dao.SecretToID[clientSecret]
	_, hasServer := dao.ServerList[clientID]
	if !hasID || !hasServer {
		return 0, status.Errorf(codes.Unauthenticated, "客户端认证失败")
	}
	return clientID, nil
}
