package singleton

import (
	"time"

	"github.com/naiba/nezha/model"
)

func GetServiceHistories(serverID uint64) ([]*model.ServiceInfos, error) {
	var serviceHistories []*model.ServiceHistory
	if err := DB.Model(&model.ServiceHistory{}).Select("service_id, created_at, server_id, avg_delay").
		Where("server_id = ?", serverID).Where("created_at >= ?", time.Now().Add(-24*time.Hour)).Order("service_id, created_at").
		Scan(&serviceHistories).Error; err != nil {
		return nil, err
	}

	var sortedServiceIDs []uint64
	resultMap := make(map[uint64]*model.ServiceInfos)
	for _, history := range serviceHistories {
		infos, ok := resultMap[history.ServiceID]
		if !ok {
			infos = &model.ServiceInfos{
				ServiceID:   history.ServiceID,
				ServerID:    history.ServerID,
				ServiceName: ServiceSentinelShared.services[history.ServiceID].Name,
				ServerName:  ServerList[history.ServerID].Name,
			}
			resultMap[history.ServiceID] = infos
			sortedServiceIDs = append(sortedServiceIDs, history.ServiceID)
		}
		infos.CreatedAt = append(infos.CreatedAt, history.CreatedAt.Truncate(time.Minute).Unix()*1000)
		infos.AvgDelay = append(infos.AvgDelay, history.AvgDelay)
	}

	ret := make([]*model.ServiceInfos, 0, len(sortedServiceIDs))
	for _, id := range sortedServiceIDs {
		ret = append(ret, resultMap[id])
	}

	return ret, nil
}
