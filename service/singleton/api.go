package singleton

import "github.com/naiba/nezha/model"

type ServerAPI struct {
	Token  string // 传入Token 后期可能会需要用于scope判定
	IDList []uint64
	Tag    string
}

type CommonResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type StatusResponse struct {
	Host   *model.Host      `json:"host"`
	Status *model.HostState `json:"status"`
}

type ServerStatusResponse struct {
	CommonResponse
	Result []*StatusResponse `json:"result"`
}

// GetStatusByIDList 获取传入IDList的服务器状态信息
func (s *ServerAPI) GetStatusByIDList() *ServerStatusResponse {
	var res []*StatusResponse

	ServerLock.RLock()
	defer ServerLock.RUnlock()

	for _, v := range s.IDList {
		server := ServerList[v]
		if server == nil {
			continue
		}
		res = append(res, &StatusResponse{
			Host:   server.Host,
			Status: server.State,
		})
	}

	return &ServerStatusResponse{
		CommonResponse: CommonResponse{
			Code:    0,
			Message: "success",
		},
		Result: res,
	}
}

// GetStatusByTag 获取传入分组的所有服务器状态信息
func (s *ServerAPI) GetStatusByTag() *ServerStatusResponse {
	s.IDList = ServerTagToIDList[s.Tag]
	return s.GetStatusByIDList()
}

// GetAllStatus 获取所有服务器状态信息
func (s *ServerAPI) GetAllStatus() *ServerStatusResponse {
	ServerLock.RLock()
	defer ServerLock.RUnlock()
	var res []*StatusResponse
	for _, v := range ServerList {
		host := v.Host
		state := v.State
		if host == nil || state == nil {
			continue
		}
		res = append(res, &StatusResponse{
			Host:   v.Host,
			Status: v.State,
		})
	}

	return &ServerStatusResponse{
		CommonResponse: CommonResponse{
			Code:    0,
			Message: "success",
		},
		Result: res,
	}
}
