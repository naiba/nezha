package rpc

import (
	"context"
	"strconv"

	"github.com/naiba/nezha/service/dao"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type AuthHandler struct {
	ClientID     string
	ClientSecret string
}

func (a *AuthHandler) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{"client_id": a.ClientID, "client_secret": a.ClientSecret}, nil
}

func (a *AuthHandler) RequireTransportSecurity() bool {
	return !dao.Conf.Debug
}

func (a *AuthHandler) Check(ctx context.Context) (clientID uint64, err error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		err = status.Errorf(codes.Unauthenticated, "获取 metaData 失败")
		return
	}
	var clientSecret string
	if value, ok := md["client_id"]; ok {
		clientID, _ = strconv.ParseUint(value[0], 10, 64)
	}
	if value, ok := md["client_secret"]; ok {
		clientSecret = value[0]
	}

	dao.ServerLock.RLock()
	defer dao.ServerLock.RUnlock()
	if server, has := dao.ServerList[clientID]; !has || server.Secret != clientSecret {
		err = status.Errorf(codes.Unauthenticated, "客户端认证失败")
	}
	return
}
