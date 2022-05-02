package model

type ServiceItemResponse struct {
	Monitor     *Monitor
	CurrentUp   uint64
	CurrentDown uint64
	TotalUp     uint64
	TotalDown   uint64
	Delay       *[30]float32
	Up          *[30]int
	Down        *[30]int
}

func (r ServiceItemResponse) TotalUptime() float32 {
	if r.TotalUp+r.TotalDown == 0 {
		return 0
	}
	return float32(r.TotalUp) / (float32(r.TotalUp + r.TotalDown)) * 100
}
