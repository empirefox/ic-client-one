package center

import (
	"bytes"
	"encoding/json"
	"fmt"
	"syscall"

	"github.com/empirefox/ic-client-one/connector"
	"github.com/empirefox/ic-client-one/storage"
	"github.com/empirefox/ic-client-one/utils"
	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
)

func (center *central) ServeLocal(c *gin.Context) {
	socket, err := center.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		glog.Errorln(err)
		return
	}
	defer socket.Close()
	ws := NewConn(center, socket, center.quit)
	go ws.WriteClose()

	center.AddStatusObserver(ws)
	defer center.DelStatusObserver(ws)

	for {
		cmd := &FromLocalCommand{}
		if err := ws.ReadJSON(cmd); err != nil {
			glog.Errorln(err)
			return
		}
		cmd.Ws = ws
		glog.Infoln("Exec local cmd:", cmd.Type)
		center.OnLocalCommand(cmd)
		glog.Infoln("Finished local cmd:", cmd.Type)
	}
}

func (center *central) OnLocalCommand(cmd *FromLocalCommand) {
	center.localCommand <- cmd
}

func (center *central) onLocalCommand(cmd *FromLocalCommand) {
	defer func() {
		if err := recover(); err != nil {
			glog.Errorln(err)
		}
	}()
	switch cmd.Type {
	case "GetStatus":
		cmd.Ws.Send(center.status)
	case "GetRoomInfo":
		center.onGetRoomInfo(cmd.Ws)
		center.onGetLocalCameras()
	case "GetCameras":
		center.onGetLocalCameras()
	case "DoConnect":
		center.onConnectCtrl()
	case "DoLogin":
		center.onDoLogin()
	case "SetRecOn":
		center.onSetRecEnabled(cmd.Value(), true)
	case "SetRecOff":
		center.onSetRecEnabled(cmd.Value(), false)
	case "GetRegable":
		center.onGetRegable()
	case "SetRegToken":
		center.onSetRegToken(cmd.Value())
	case "DoRemoveRegToken":
		center.onDoRemoveRegToken()
	case "DoRegRoom":
		center.onRegRoom([]byte(cmd.Content))
	case "DoRemoveRoom":
		center.onRemoveRoom()
	case "Close":
		return
	case "Exit":
		syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		return
	default:
		glog.Errorln("Unknow local command:", *cmd)
	}
}

func (center *central) onGetRoomInfo(ws Ws) {
	ws.Send([]byte(fmt.Sprintf(`{
		"type":"RoomInfo",
		"content":{
			"pid":%d
		}
	}`, syscall.Getpid())))
}

func (center *central) onGetLocalCameras() {
	center.Connectors.LocalBroadcast()
}

// TODO make it more reliable
func (center *central) onSetRecEnabled(id []byte, on bool) {
	center.Connectors.SetRec(string(id), on)
}
func (center *central) sendLocalCamera(e *connector.Event) {
	msg, _ := json.Marshal(map[string]interface{}{
		"type":   "Rec",
		"camera": e.Ic.Map(),
		"ids":    center.Connectors.Ids(),
	})
	center.onChangeNoStatus(msg)
}

func (center *central) onGetRegable() {
	token := center.conf.GetRegToken()
	if bytes.Count(token, []byte{'.'}) != 2 {
		center.onChangeNoStatus(BAD_REG_TOKEN)
		return
	}
	center.onChangeNoStatus(REGABLE)
}

func (center *central) onSetRegToken(token []byte) {
	if err := center.conf.Put(storage.K_REG_TOKEN, token); err != nil {
		center.onChangeNoStatus(SAVE_REG_TOKEN_ERROR)
		return
	}
	center.onChangeNoStatus(REGABLE)
}

func (center *central) onDoRemoveRegToken() {
	center.conf.Del(storage.K_REG_TOKEN)
	center.onChangeNoStatus(BAD_REG_TOKEN)
	center.onStatusChange(nil)
}

func (center *central) onRegRoom(nameJson []byte) {
	pre := center.status
	if center.hasCtrl {
		center.onStatusChange(REGGING)
		center.ctrlConn.Send([]byte(fmt.Sprintf(`one:RegRoom:%s:%s`, center.conf.GetRegToken(), nameJson)))
	} else {
		center.onStatusChange(DISCONNECTED)
	}
	center.status = pre
}

func (center *central) onRemoveRoom() {
	if center.hasCtrl {
		center.sendCtrl(utils.GenServerCommand("RemoveRoom", ""))
	} else {
		center.onStatusChange(DISCONNECTED)
	}
}
