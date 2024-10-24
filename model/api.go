package model

const (
	ApiErrorUnauthorized = 10001
)

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
