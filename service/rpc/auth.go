package rpc

import (
	"context"
	"fmt"

	"github.com/p14yground/nezha/model"
	"github.com/p14yground/nezha/service/dao"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthHandler ..
type AuthHandler struct {
	ClientID     string
	ClientSecret string
}

// GetRequestMetadata ..
func (a *AuthHandler) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{"app_key": a.ClientID, "app_secret": a.ClientSecret}, nil
}

// RequireTransportSecurity ..
func (a *AuthHandler) RequireTransportSecurity() bool {
	return !dao.Conf.Debug
}

// Check ..
func (a *AuthHandler) Check(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Errorf(codes.Unauthenticated, "获取 metaData 失败")
	}

	var (
		ClientID     string
		ClientSecret string
	)
	if value, ok := md["app_key"]; ok {
		ClientID = value[0]
	}
	if value, ok := md["app_secret"]; ok {
		ClientSecret = value[0]
	}

	if _, ok := dao.Cache.Get(fmt.Sprintf("%s%s%s", model.CtxKeyServer, ClientID, ClientSecret)); !ok {
		return status.Errorf(codes.Unauthenticated, "客户端认证失败")
	}

	return nil
}
