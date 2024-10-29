package controller

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"golang.org/x/sync/singleflight"

	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/pkg/utils"
	"github.com/naiba/nezha/service/singleton"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  32768,
	WriteBufferSize: 32768,
}

// Websocket server stream
// @Summary Websocket server stream
// @tags common
// @Schemes
// @Description Websocket server stream
// @security BearerAuth
// @Produce json
// @Success 200 {object} model.StreamServerData
// @Router /ws/server [get]
func serverStream(c *gin.Context) (any, error) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	count := 0
	for {
		stat, err := getServerStat(c, count == 0)
		if err != nil {
			continue
		}
		if err := conn.WriteMessage(websocket.TextMessage, stat); err != nil {
			break
		}
		count += 1
		if count%4 == 0 {
			err = conn.WriteMessage(websocket.PingMessage, []byte{})
			if err != nil {
				break
			}
		}
		time.Sleep(time.Second * 2)
	}
	return nil, newWsError("")
}

var requestGroup singleflight.Group

func getServerStat(c *gin.Context, withPublicNote bool) ([]byte, error) {
	_, isMember := c.Get(model.CtxKeyAuthorizedUser)
	authorized := isMember // TODO || isViewPasswordVerfied
	v, err, _ := requestGroup.Do(fmt.Sprintf("serverStats::%t", authorized), func() (interface{}, error) {
		singleton.SortedServerLock.RLock()
		defer singleton.SortedServerLock.RUnlock()

		var serverList []*model.Server
		if authorized {
			serverList = singleton.SortedServerList
		} else {
			serverList = singleton.SortedServerListForGuest
		}

		var servers []model.StreamServer
		for i := 0; i < len(serverList); i++ {
			server := serverList[i]
			host := *server.Host
			host.IP = ""
			servers = append(servers, model.StreamServer{
				ID:           server.ID,
				Name:         server.Name,
				PublicNote:   utils.IfOr(withPublicNote, server.PublicNote, ""),
				DisplayIndex: server.DisplayIndex,
				Host:         &host,
				State:        server.State,
				LastActive:   server.LastActive,
			})
		}

		return utils.Json.Marshal(model.StreamServerData{
			Now:     time.Now().Unix() * 1000,
			Servers: servers,
		})
	})
	return v.([]byte), err
}
