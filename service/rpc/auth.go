package rpc

import (
	"context"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/hashicorp/go-uuid"
	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/service/singleton"
)

type authHandler struct {
	ClientSecret string
	ClientUUID   string
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

	if clientSecret != singleton.Conf.AgentSecretKey {
		return 0, status.Errorf(codes.Unauthenticated, "客户端认证失败")
	}

	var clientUUID string
	if value, ok := md["client_uuid"]; ok {
		clientUUID = value[0]
	}

	if _, err := uuid.ParseUUID(clientUUID); err != nil {
		return 0, status.Errorf(codes.Unauthenticated, "客户端 UUID 不合法")
	}

	singleton.ServerLock.RLock()
	defer singleton.ServerLock.RUnlock()
	clientID, hasID := singleton.ServerUUIDToID[clientUUID]
	if !hasID {
		s := model.Server{UUID: clientUUID}
		if err := singleton.DB.Create(&s).Error; err != nil {
			return 0, status.Errorf(codes.Unauthenticated, err.Error())
		}
		s.Host = &model.Host{}
		s.State = &model.HostState{}
		s.TaskCloseLock = new(sync.Mutex)
		singleton.ServerList[s.ID] = &s
		singleton.ServerUUIDToID[clientUUID] = s.ID
		clientID = s.ID
	}

	return clientID, nil
}
