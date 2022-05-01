package model

type ServiceItemResponse struct {
	Monitor     *Monitor
	CurrentUp   uint64
	CurrentDown uint64
	Delay       *[30]float32
	Up          *[30]int
	Down        *[30]int
}

func sum(slice *[30]int) int {
	if slice == nil {
		return 0
	}
	var sum int
	for _, v := range *slice {
		sum += v
	}
	return sum
}

func (r ServiceItemResponse) TotalUp() int {
	return sum(r.Up)
}

func (r ServiceItemResponse) TotalDown() int {
	return sum(r.Down)
}

func (r ServiceItemResponse) TotalUptime() float32 {
	return float32(r.TotalUp()) / (float32(r.TotalUp() + r.TotalDown())) * 100
}
