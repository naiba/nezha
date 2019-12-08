package rpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthHandler ..
type AuthHandler struct {
	AppKey    string
	AppSecret string
}

// GetRequestMetadata ..
func (a *AuthHandler) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{"app_key": a.AppKey, "app_secret": a.AppSecret}, nil
}

// RequireTransportSecurity ..
func (a *AuthHandler) RequireTransportSecurity() bool {
	return false
}

// Check ..
func (a *AuthHandler) Check(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Errorf(codes.Unauthenticated, "metadata.FromIncomingContext err")
	}

	var (
		AppKey    string
		AppSecret string
	)
	if value, ok := md["app_key"]; ok {
		AppKey = value[0]
	}
	if value, ok := md["app_secret"]; ok {
		AppSecret = value[0]
	}

	if AppKey != a.GetAppKey() || AppSecret != a.GetAppSecret() {
		return status.Errorf(codes.Unauthenticated, "invalid token")
	}

	return nil
}

// GetAppKey ..
func (a *AuthHandler) GetAppKey() string {
	return a.AppKey
}

// GetAppSecret ..
func (a *AuthHandler) GetAppSecret() string {
	return a.AppSecret
}
