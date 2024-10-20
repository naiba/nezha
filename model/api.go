package model

const (
	ApiErrorUnauthorized = 10001
)

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

type LoginRequest struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type CommonResponse[T any] struct {
	Success bool   `json:"success,omitempty"`
	Data    T      `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

type LoginResponse struct {
	Token  string `json:"token,omitempty"`
	Expire string `json:"expire,omitempty"`
}
