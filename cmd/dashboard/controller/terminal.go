package controller

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/go-uuid"
	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/pkg/utils"
	"github.com/naiba/nezha/pkg/websocketx"
	"github.com/naiba/nezha/proto"
	"github.com/naiba/nezha/service/rpc"
	"github.com/naiba/nezha/service/singleton"
)

// Create web ssh terminal
// @Summary Create web ssh terminal
// @Description Create web ssh terminal
// @Tags auth required
// @Accept json
// @Param terminal body model.TerminalForm true "TerminalForm"
// @Produce json
// @Success 200 {object} model.CreateTerminalResponse
// @Router /terminal [post]
func createTerminal(c *gin.Context) error {
	var createTerminalReq model.TerminalForm
	if err := c.ShouldBind(&createTerminalReq); err != nil {
		return err
	}

	streamId, err := uuid.GenerateUUID()
	if err != nil {
		return err
	}

	rpc.NezhaHandlerSingleton.CreateStream(streamId)

	singleton.ServerLock.RLock()
	server := singleton.ServerList[createTerminalReq.ServerID]
	singleton.ServerLock.RUnlock()
	if server == nil || server.TaskStream == nil {
		return errors.New("server not found or not connected")
	}

	terminalData, _ := utils.Json.Marshal(&model.TerminalTask{
		StreamID: streamId,
	})
	if err := server.TaskStream.Send(&proto.Task{
		Type: model.TaskTypeTerminalGRPC,
		Data: string(terminalData),
	}); err != nil {
		return err
	}

	c.JSON(http.StatusOK, model.CommonResponse[model.CreateTerminalResponse]{
		Success: true,
		Data: model.CreateTerminalResponse{
			SessionID:  streamId,
			ServerID:   server.ID,
			ServerName: server.Name,
		},
	})

	return nil
}

// TerminalStream web ssh terminal stream
// @Summary Terminal stream
// @Description Terminal stream
// @Tags auth required
// @Param id path string true "Stream ID"
// @Router /terminal/{id} [get]
func terminalStream(c *gin.Context) error {
	streamId := c.Param("id")
	if _, err := rpc.NezhaHandlerSingleton.GetStream(streamId); err != nil {
		return err
	}
	defer rpc.NezhaHandlerSingleton.CloseStream(streamId)

	wsConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return err
	}
	defer wsConn.Close()
	conn := websocketx.NewConn(wsConn)

	go func() {
		// PING 保活
		for {
			if err = conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
			time.Sleep(time.Second * 10)
		}
	}()

	if err = rpc.NezhaHandlerSingleton.UserConnected(streamId, conn); err != nil {
		return err
	}

	return rpc.NezhaHandlerSingleton.StartStream(streamId, time.Second*10)
}
