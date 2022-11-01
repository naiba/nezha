package websocketx

import (
	"sync"

	"github.com/gorilla/websocket"
	"github.com/samber/lo"
)

type Conn struct {
	*websocket.Conn
	writeLock sync.Mutex
}

func (conn *Conn) WriteMessage(msgType int, data []byte) error {
	conn.writeLock.Lock()
	defer conn.writeLock.Unlock()
	var err error
	lo.TryCatchWithErrorValue(func() error {
		return conn.Conn.WriteMessage(msgType, data)
	}, func(res any) {
		err = res.(error)
	})
	return err
}
