package model

import (
	pb "github.com/naiba/nezha/proto"
)

type MonitorHistory struct {
	Common
	MonitorID  uint64
	Delay      float32 // 延迟，毫秒
	Data       string
	Successful bool // 是否成功
}

func PB2MonitorHistory(r *pb.TaskResult) MonitorHistory {
	return MonitorHistory{
		Delay:      r.GetDelay(),
		Successful: r.GetSuccessful(),
		MonitorID:  r.GetId(),
		Data:       r.GetData(),
	}
}
