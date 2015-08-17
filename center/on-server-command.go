package center

import (
	"bytes"
	"encoding/json"

	"github.com/empirefox/ic-client-one/ipcam"
	"github.com/empirefox/ic-client-one/storage"
	"github.com/empirefox/ic-client-one/wsio"
	"github.com/golang/glog"
)

func (center *central) readCtrl(c Ws) {
	defer center.DelCtrl(c)
	defer c.Close()
	for {
		var cmd wsio.FromServerCommand
		if err := c.ReadJSON(&cmd); err != nil {
			glog.Errorln(err)
			center.ChangeStatus(BAD_SERVER_MSG)
			return
		}
		center.OnServerCommand(&cmd)
	}
}

func (center *central) OnServerCommand(cmd *wsio.FromServerCommand) {
	center.serverCommand <- cmd
}

func (center *central) onServerCommand(cmd *wsio.FromServerCommand) {
	defer func() {
		if err := recover(); err != nil {
			glog.Errorln(err)
		}
	}()
	switch cmd.Name {
	case "GetIpcams":
		center.onSendIpcams()
	case "ManageGetIpcam":
		center.onManageGetIpcam(cmd)
	case "ManageSetIpcam":
		center.onManageSetIpcam(cmd)
	case "ManageDelIpcam":
		center.onManageDelIpcam(cmd)
	case "ManageReconnectIpcam":
		center.onReconnectIpcam(cmd)
	case "CreateSignalingConnection":
		go center.OnCreateSignalingConnection(cmd)
	case "LoginOk":
		center.onStatusChange(READY)
		center.onSendIpcams()
	case "BadRoomToken":
		center.conf.Del(storage.K_ROOM_TOKEN)
		center.onStatusChange(BAD_ROOM_TOKEN)
	case "SetRoomToken":
		center.onSetRoomToken(cmd)
	case "BadRegToken":
		center.conf.Del(storage.K_REG_TOKEN)
		center.onChangeNoStatus(BAD_REG_TOKEN)
	case "RegError":
		center.onStatusChange(REG_ERROR)
	default:
		glog.Errorln("Unknow server command:", *cmd)
	}
}

func (center *central) onSendIpcams() {
	info, err := json.Marshal(center.conf.GetIpcams().Map(ipcam.TAG_VIEW))
	if err != nil {
		glog.Errorln(err)
		return
	}
	center.ctrlConn.Send(append([]byte("one:Ipcams:"), info...))
}

// Content => id
func (center *central) onManageGetIpcam(cmd *wsio.FromServerCommand) {
	i, err := center.conf.GetIpcam(cmd.Value())
	if err != nil {
		center.ctrlConn.Send(cmd.ToManyInfo("Cannot get ipcam"))
		return
	}
	msg, err := cmd.ToManyObj(i.Map())
	if err != nil {
		center.ctrlConn.Send(cmd.ToManyInfo("Cannot get ipcam content"))
		return
	}
	center.ctrlConn.Send(msg)
}

// Content => SetterIpcam
func (center *central) onManageSetIpcam(cmd *wsio.FromServerCommand) {
	var data ipcam.SetterIpcam
	if err := json.Unmarshal(cmd.Value(), &data); err != nil {
		center.ctrlConn.Send(cmd.ToManyInfo("Cannot parse ipcam"))
		return
	}
	if err := center.conf.PutIpcam(&data.Ipcam, []byte(data.Target)); err != nil {
		center.ctrlConn.Send(cmd.ToManyInfo("Cannot get ipcam"))
		return
	}
	center.onSendIpcams()
}

// Content => Ipcam.Id
func (center *central) onManageDelIpcam(cmd *wsio.FromServerCommand) {
	if err := center.conf.RemoveIpcam(cmd.Value()); err != nil {
		center.ctrlConn.Send(cmd.ToManyInfo("Cannot remove ipcam"))
		return
	}
	center.onSendIpcams()
}

// Content => id
func (center *central) onReconnectIpcam(cmd *wsio.FromServerCommand) {
	cam, err := center.conf.GetIpcam(cmd.Value())
	if err != nil {
		center.ctrlConn.Send(cmd.ToManyInfo("Cannot find ipcam"))
	}
	cam.Online = center.registry(cam, true)
	if !cam.Online {
		center.ctrlConn.Send(cmd.ToManyInfo("Failed to reconnect ipcam"))
		return
	}
	center.onSendIpcams()
}

func (center *central) onSetRoomToken(cmd *wsio.FromServerCommand) {
	if err := center.conf.Put(storage.K_ROOM_TOKEN, cmd.Value()); err != nil {
		center.onStatusChange(SAVE_ROOM_TOKEN_ERROR)
		return
	}
	center.onDoLogin()
}

func (center *central) onDoLogin() {
	token := center.conf.GetRoomToken()
	if bytes.Count(token, []byte{'.'}) != 2 {
		center.onStatusChange(BAD_ROOM_TOKEN)
		return
	}
	center.onStatusChange(LOGGING_IN)
	center.ctrlConn.Send(append([]byte("one:Login:"), token...))
}
