package singleton

import (
	"errors"
	"fmt"
	"github.com/jinzhu/copier"
	"github.com/naiba/nezha/proto"
	"sync"
	"time"

	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/pkg/utils"
)

var (
	ApiTokenList         = make(map[string]*model.ApiToken)
	UserIDToApiTokenList = make(map[uint64][]string)
	ApiLock              sync.RWMutex

	ServerAPI  = &ServerAPIService{}
	MonitorAPI = &MonitorAPIService{}
)

type ServerAPIService struct{}

// CommonResponse 常规返回结构 包含状态码 和 状态信息
type CommonResponse struct {
	Code    int    `json:"code" example:"0"`
	Message string `json:"message" example:"success"`
}

type CommonServerInfo struct {
	ID           uint64 `json:"id"`
	Name         string `json:"name"`
	Tag          string `json:"tag"`
	LastActive   int64  `json:"last_active"`
	IPV4         string `json:"ipv4"`
	IPV6         string `json:"ipv6"`
	ValidIP      string `json:"valid_ip"`
	DisplayIndex int    `json:"display_index"`
	AgentVersion string `json:"agent_version"`
}

// StatusResponse 服务器状态子结构 包含服务器信息与状态信息
type StatusResponse struct {
	CommonServerInfo
	Host   *model.Host      `json:"host"`
	Status *model.HostState `json:"status"`
}

// ServerStatusResponse 服务器状态返回结构 包含常规返回结构 和 服务器状态子结构
type ServerStatusResponse struct {
	CommonResponse
	Result []*StatusResponse `json:"result"`
}

// ServerInfoResponse 服务器信息返回结构 包含常规返回结构 和 服务器信息子结构
type ServerInfoResponse struct {
	CommonResponse
	Result []*CommonServerInfo `json:"result"`
}

type MonitorAPIService struct {
}

type MonitorInfoResponse struct {
	CommonResponse
	Result []*MonitorInfo `json:"result"`
}

type MonitorInfo struct {
	MonitorID   uint64    `json:"monitor_id"`
	ServerID    uint64    `json:"server_id"`
	MonitorName string    `json:"monitor_name"`
	ServerName  string    `json:"server_name"`
	CreatedAt   []int64   `json:"created_at"`
	AvgDelay    []float32 `json:"avg_delay"`
}

func InitAPI() {
	ApiTokenList = make(map[string]*model.ApiToken)
	UserIDToApiTokenList = make(map[uint64][]string)
}

func loadAPI() {
	InitAPI()
	var tokenList []*model.ApiToken
	DB.Find(&tokenList)
	for _, token := range tokenList {
		ApiTokenList[token.Token] = token
		UserIDToApiTokenList[token.UserID] = append(UserIDToApiTokenList[token.UserID], token.Token)
	}
}

// GetStatusByIDList 获取传入IDList的服务器状态信息
func (s *ServerAPIService) GetStatusByIDList(idList []uint64) *ServerStatusResponse {
	res := &ServerStatusResponse{}
	res.Result = make([]*StatusResponse, 0)

	ServerLock.RLock()
	defer ServerLock.RUnlock()

	for _, v := range idList {
		server := ServerList[v]
		if server == nil {
			continue
		}
		ipv4, ipv6, validIP := utils.SplitIPAddr(server.Host.IP)
		info := CommonServerInfo{
			ID:           server.ID,
			Name:         server.Name,
			Tag:          server.Tag,
			LastActive:   server.LastActive.Unix(),
			IPV4:         ipv4,
			IPV6:         ipv6,
			ValidIP:      validIP,
			AgentVersion: server.Host.Version,
		}
		res.Result = append(res.Result, &StatusResponse{
			CommonServerInfo: info,
			Host:             server.Host,
			Status:           server.State,
		})
	}
	res.CommonResponse = CommonResponse{
		Code:    0,
		Message: "success",
	}
	return res
}

// GetStatusByTag 获取传入分组的所有服务器状态信息
func (s *ServerAPIService) GetStatusByTag(tag string) *ServerStatusResponse {
	return s.GetStatusByIDList(ServerTagToIDList[tag])
}

// GetAllStatus 获取所有服务器状态信息
func (s *ServerAPIService) GetAllStatus() *ServerStatusResponse {
	res := &ServerStatusResponse{}
	res.Result = make([]*StatusResponse, 0)
	ServerLock.RLock()
	defer ServerLock.RUnlock()
	for _, v := range ServerList {
		host := v.Host
		state := v.State
		if host == nil || state == nil {
			continue
		}
		ipv4, ipv6, validIP := utils.SplitIPAddr(host.IP)
		info := CommonServerInfo{
			ID:           v.ID,
			Name:         v.Name,
			Tag:          v.Tag,
			LastActive:   v.LastActive.Unix(),
			IPV4:         ipv4,
			IPV6:         ipv6,
			ValidIP:      validIP,
			DisplayIndex: v.DisplayIndex,
			AgentVersion: v.Host.Version,
		}
		res.Result = append(res.Result, &StatusResponse{
			CommonServerInfo: info,
			Host:             v.Host,
			Status:           v.State,
		})
	}
	res.CommonResponse = CommonResponse{
		Code:    0,
		Message: "success",
	}
	return res
}

// GetListByTag 获取传入分组的所有服务器信息
func (s *ServerAPIService) GetListByTag(tag string) *ServerInfoResponse {
	res := &ServerInfoResponse{}
	res.Result = make([]*CommonServerInfo, 0)

	ServerLock.RLock()
	defer ServerLock.RUnlock()
	for _, v := range ServerTagToIDList[tag] {
		server := ServerList[v]
		host := server.Host
		if host == nil {
			continue
		}
		ipv4, ipv6, validIP := utils.SplitIPAddr(host.IP)
		info := &CommonServerInfo{
			ID:           v,
			Name:         server.Name,
			Tag:          server.Tag,
			LastActive:   server.LastActive.Unix(),
			IPV4:         ipv4,
			IPV6:         ipv6,
			ValidIP:      validIP,
			AgentVersion: server.Host.Version,
		}
		res.Result = append(res.Result, info)
	}
	res.CommonResponse = CommonResponse{
		Code:    0,
		Message: "success",
	}
	return res
}

// GetAllList 获取所有服务器信息
func (s *ServerAPIService) GetAllList() *ServerInfoResponse {
	res := &ServerInfoResponse{}
	res.Result = make([]*CommonServerInfo, 0)

	ServerLock.RLock()
	defer ServerLock.RUnlock()
	for _, v := range ServerList {
		host := v.Host
		if host == nil {
			continue
		}
		ipv4, ipv6, validIP := utils.SplitIPAddr(host.IP)
		info := &CommonServerInfo{
			ID:           v.ID,
			Name:         v.Name,
			Tag:          v.Tag,
			LastActive:   v.LastActive.Unix(),
			IPV4:         ipv4,
			IPV6:         ipv6,
			ValidIP:      validIP,
			AgentVersion: v.Host.Version,
		}
		res.Result = append(res.Result, info)
	}
	res.CommonResponse = CommonResponse{
		Code:    0,
		Message: "success",
	}
	return res
}

func (m *MonitorAPIService) GetMonitorHistories(query map[string]any) *MonitorInfoResponse {
	var (
		resultMap        = make(map[uint64]*MonitorInfo)
		monitorHistories []*model.MonitorHistory
		sortedMonitorIDs []uint64
	)
	res := &MonitorInfoResponse{
		CommonResponse: CommonResponse{
			Code:    0,
			Message: "success",
		},
	}
	if err := DB.Model(&model.MonitorHistory{}).Select("monitor_id, created_at, server_id, avg_delay").
		Where(query).Where("created_at >= ?", time.Now().Add(-24*time.Hour)).Order("monitor_id, created_at").
		Scan(&monitorHistories).Error; err != nil {
		res.CommonResponse = CommonResponse{
			Code:    500,
			Message: err.Error(),
		}
	} else {
		for _, history := range monitorHistories {
			infos, ok := resultMap[history.MonitorID]
			if !ok {
				infos = &MonitorInfo{
					MonitorID:   history.MonitorID,
					ServerID:    history.ServerID,
					MonitorName: ServiceSentinelShared.monitors[history.MonitorID].Name,
					ServerName:  ServerList[history.ServerID].Name,
				}
				resultMap[history.MonitorID] = infos
				sortedMonitorIDs = append(sortedMonitorIDs, history.MonitorID)
			}
			infos.CreatedAt = append(infos.CreatedAt, history.CreatedAt.Truncate(time.Minute).Unix()*1000)
			infos.AvgDelay = append(infos.AvgDelay, history.AvgDelay)
		}
		for _, monitorID := range sortedMonitorIDs {
			res.Result = append(res.Result, resultMap[monitorID])
		}
	}
	return res
}

type ServerConfigData struct {
	ID           uint64 `json:"id,omitempty" form:"id" example:"0"`
	Name         string `binding:"required" json:"name" form:"name" example:"服务器名"`       // 服务器名称
	DisplayIndex int    `json:"displayIndex,omitempty" form:"DisplayIndex" example:"0"`   // 展示排序，越大越靠前
	Secret       string `json:"secret,omitempty" form:"secret" example:""`                // 服务器密钥, 默认18位随机字符串
	Tag          string `json:"tag,omitempty" form:"tag" example:"服务器组"`                  // 服务器分组
	Note         string `json:"note,omitempty" form:"note" example:"备注"`                  // 管理员可见备注
	HideForGuest string `json:"hideForGuest,omitempty" form:"hideForGuest" example:"off"` // 对游客隐藏
	EnableDDNS   string `json:"enableDDNS,omitempty" form:"enableDDNS" example:"off"`
	EnableIPv4   string `json:"enableIPv4,omitempty" form:"enableIPv4" example:"off"`
	EnableIpv6   string `json:"enableIpv6,omitempty" form:"enableIpv6" example:"off"`
	DDNSDomain   string `json:"DDNSDomain,omitempty" form:"DDNSDomain" example:""`
	DDNSProfile  string `json:"DDNSProfile,omitempty" form:"DDNSProfile" example:""`
}

type ServerConfigResponse struct {
	CommonResponse
	Result *ServerConfigData `json:"result"`
}

func (sf *ServerConfigData) MapToServer() model.Server {
	var server model.Server
	server.ID = sf.ID
	server.Name = sf.Name
	server.Secret = sf.Secret
	server.DisplayIndex = sf.DisplayIndex
	server.Tag = sf.Tag
	server.Note = sf.Note
	server.HideForGuest = sf.HideForGuest == "on"
	server.EnableDDNS = sf.EnableDDNS == "on"
	server.EnableIPv4 = sf.EnableIPv4 == "on"
	server.EnableIpv6 = sf.EnableIpv6 == "on"
	server.DDNSDomain = sf.DDNSDomain
	server.DDNSProfile = sf.DDNSProfile
	return server
}

func (sf *ServerConfigData) MapFromServer(server model.Server) {
	sf.ID = server.ID
	sf.Name = server.Name
	sf.Secret = server.Secret
	sf.DisplayIndex = server.DisplayIndex
	sf.Tag = server.Tag
	sf.Note = server.Note
	sf.HideForGuest = utils.BoolToString(server.HideForGuest, "on", "off")
	sf.EnableDDNS = utils.BoolToString(server.EnableDDNS, "on", "off")
	sf.EnableIPv4 = utils.BoolToString(server.EnableIPv4, "on", "off")
	sf.EnableIpv6 = utils.BoolToString(server.EnableIpv6, "on", "off")
	sf.DDNSDomain = server.DDNSDomain
	sf.DDNSProfile = server.DDNSProfile
}

func (s *ServerAPIService) AddServer(sf ServerConfigData) (*ServerConfigResponse, error) {
	var err error

	res := &ServerConfigResponse{
		Result: &ServerConfigData{},
	}
	res.CommonResponse = CommonResponse{
		Code:    0,
		Message: "success",
	}
	server := sf.MapToServer()
	server.TaskCloseLock = new(sync.Mutex)
	if server.Secret, err = utils.GenerateRandomString(18); err != nil {
		return nil, err
	}

	if err = DB.Create(&server).Error; err != nil {
		return nil, err
	}

	res.Result.MapFromServer(server)

	server.Host = &model.Host{}
	server.State = &model.HostState{}

	ServerLock.Lock()
	SecretToID[server.Secret] = server.ID
	ServerList[server.ID] = &server
	ServerTagToIDList[server.Tag] = append(ServerTagToIDList[server.Tag], server.ID)
	ServerLock.Unlock()

	ReSortServer()
	return res, nil
}

func (s *ServerAPIService) EditServer(sf ServerConfigData) (*ServerConfigResponse, error) {
	if sf.Secret == "" {
		return nil, errors.New("secret is required")
	}

	res := &ServerConfigResponse{
		Result: &ServerConfigData{},
	}
	res.CommonResponse = CommonResponse{
		Code:    0,
		Message: "success",
	}

	server := sf.MapToServer()
	err := DB.Save(&server).Error
	if err != nil {
		return nil, err
	}

	res.Result.MapFromServer(server)

	ServerLock.Lock()
	server.CopyFromRunningServer(ServerList[server.ID])
	// 如果修改了 Secret
	if server.Secret != ServerList[server.ID].Secret {
		// 删除旧 Secret-ID 绑定关系
		SecretToID[server.Secret] = server.ID
		// 设置新的 Secret-ID 绑定关系
		delete(SecretToID, ServerList[server.ID].Secret)
	}
	// 如果修改了Tag
	oldTag := ServerList[server.ID].Tag
	newTag := server.Tag
	if newTag != oldTag {
		index := -1
		for i := 0; i < len(ServerTagToIDList[oldTag]); i++ {
			if ServerTagToIDList[oldTag][i] == server.ID {
				index = i
				break
			}
		}
		if index > -1 {
			// 删除旧 Tag-ID 绑定关系
			ServerTagToIDList[oldTag] = append(ServerTagToIDList[oldTag][:index], ServerTagToIDList[oldTag][index+1:]...)
			if len(ServerTagToIDList[oldTag]) == 0 {
				delete(ServerTagToIDList, oldTag)
			}
		}
		// 设置新的 Tag-ID 绑定关系
		ServerTagToIDList[newTag] = append(ServerTagToIDList[newTag], server.ID)
	}
	ServerList[server.ID] = &server
	ServerLock.Unlock()

	ReSortServer()
	return res, nil
}

type ServerDeleteRequest struct {
	IDList []uint64 `json:"id_list" form:"id_list" example:"1,2"` // 需要删除的服务器ID
}

type ServerDeleteResponse struct {
	CommonResponse
}

func (s *ServerAPIService) DeleteServer(sf ServerDeleteRequest) *ServerDeleteResponse {
	// 先确定要删除的 Server 是否存在
	ServerLock.RLock()
	for _, id := range sf.IDList {
		if _, ok := ServerList[id]; !ok {
			ServerLock.RUnlock()
			return &ServerDeleteResponse{
				CommonResponse: CommonResponse{
					Code:    1001,
					Message: fmt.Sprintf("Server %d not found", id),
				},
			}
		}
	}
	ServerLock.RUnlock()

	// 开始删除流程
	ServerLock.Lock()
	// 删除数据库记录
	err := DB.Unscoped().Delete(&model.Server{}, "id in ?", sf.IDList).Error
	if err != nil {
		ServerLock.Unlock()
		return &ServerDeleteResponse{
			CommonResponse: CommonResponse{
				Code:    1002,
				Message: err.Error(),
			},
		}
	}
	// 删除映射关系
	for _, id := range sf.IDList {
		OnServerDelete(id)
	}
	ServerLock.Unlock()
	ReSortServer()
	return &ServerDeleteResponse{
		CommonResponse: CommonResponse{
			Code:    0,
			Message: "success",
		},
	}
}

type BatchUpdateServerGroupRequest struct {
	Servers []uint64 `json:"servers" form:"servers" example:"1,2"`  // 需要更新的服务器ID
	Group   string   `json:"group" form:"group" example:"newGroup"` // 新的分组
}

type BatchUpdateServerGroupResponse struct {
	CommonResponse
}

func (s *ServerAPIService) BatchUpdateGroup(req BatchUpdateServerGroupRequest) *BatchUpdateServerGroupResponse {
	ServerLock.Lock()
	// 先检查一遍确保所有server都存在
	for _, id := range req.Servers {
		if _, ok := ServerList[id]; !ok {
			ServerLock.Unlock()
			return &BatchUpdateServerGroupResponse{
				CommonResponse: CommonResponse{
					Code:    1002,
					Message: fmt.Sprintf("Server %d not found", id),
				},
			}
		}
	}

	// 移除数据库记录
	if err := DB.Model(&model.Server{}).Where("id in (?)", req.Servers).Update("tag", req.Group).Error; err != nil {
		ServerLock.Unlock()
		return &BatchUpdateServerGroupResponse{
			CommonResponse: CommonResponse{
				Code:    1001,
				Message: err.Error(),
			},
		}
	}

	for _, serverId := range req.Servers {
		oldServer, ok := ServerList[serverId]
		if !ok {
			continue
		}
		var newServer model.Server
		copier.Copy(&newServer, oldServer)
		newServer.Tag = req.Group
		// 如果修改了Ta
		oldTag := oldServer.Tag
		newTag := newServer.Tag
		if newTag != oldTag {
			index := -1
			for i := 0; i < len(ServerTagToIDList[oldTag]); i++ {
				if ServerTagToIDList[oldTag][i] == newServer.ID {
					index = i
					break
				}
			}
			if index > -1 {
				// 删除旧 Tag-ID 绑定关系
				ServerTagToIDList[oldTag] = append(ServerTagToIDList[oldTag][:index], ServerTagToIDList[oldTag][index+1:]...)
				if len(ServerTagToIDList[oldTag]) == 0 {
					delete(ServerTagToIDList, oldTag)
				}
			}
			// 设置新的 Tag-ID 绑定关系
			ServerTagToIDList[newTag] = append(ServerTagToIDList[newTag], newServer.ID)
		}
		ServerList[newServer.ID] = &newServer
	}

	ServerLock.Unlock()
	ReSortServer()
	return &BatchUpdateServerGroupResponse{
		CommonResponse{
			Code:    0,
			Message: "success",
		},
	}
}

type ForceUpdateAgentRequest struct {
	ServerIDList []uint64 `json:"server_id_list" example:"1,2"`
}
type ForceUpdateAgentResponse struct {
	CommonResponse
	Success []uint64          `json:"success" example:"1,2"`
	Fail    map[uint64]string `json:"fail" example:"3:\"offline\""`
}

func (s *ServerAPIService) ForceUpdateAgent(req ForceUpdateAgentRequest) *ForceUpdateAgentResponse {
	res := &ForceUpdateAgentResponse{
		CommonResponse: CommonResponse{
			Code:    0,
			Message: "success",
		},
		Success: make([]uint64, 0),
		Fail:    map[uint64]string{},
	}
	successCount := 0
	for _, serverId := range req.ServerIDList {
		ServerLock.RLock()
		server := ServerList[serverId]
		ServerLock.RUnlock()
		if server != nil && server.TaskStream != nil {
			if err := server.TaskStream.Send(&proto.Task{
				Type: model.TaskTypeUpgrade,
			}); err != nil {
				res.Fail[serverId] = err.Error()
			} else {
				res.Success = append(res.Success, serverId)
				successCount += 1
			}
		} else {
			res.Fail[serverId] = "offline"
		}
	}
	if successCount == 0 {
		res.CommonResponse = CommonResponse{
			Code:    1001,
			Message: "Fail",
		}
	} else if successCount != len(req.ServerIDList) {
		res.CommonResponse = CommonResponse{
			Code:    1002,
			Message: "Partial Success",
		}
	}
	return res
}
