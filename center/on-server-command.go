package center

import (
	"bytes"
	"encoding/json"

	"github.com/empirefox/ic-client-one/connector"
	"github.com/empirefox/ic-client-one/ipcam"
	"github.com/empirefox/ic-client-one/storage"
	"github.com/empirefox/ic-client-one/wsio"
	"github.com/golang/glog"
)

var (
	kIcIds  = []byte("IcIds")
	kIc     = []byte("Ic")
	kIcIdCh = []byte("IcIdCh")
	kXIc    = []byte("XIc")
	kSecIc  = []byte("SecIc")
	kNoIc   = []byte("NoIc")
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
		glog.Infoln("Exec server cmd:", cmd.Name)
		center.OnServerCommand(&cmd)
		glog.Infoln(cmd.Name, "============")
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
	case "ManageGetIpcam":
		center.onManageGetIpcam(cmd)

	case "ManageSetIpcam":
		center.onManageSetIpcam(cmd)

	case "ManageDelIpcam":
		center.onManageDelIpcam(cmd)

	case "CreateSignalingConnection":
		go center.OnCreateSignalingConnection(cmd)

	case "Broadcast", "UserOnline":
		center.onViewRoom(cmd)

	case "BadRoomToken":
		//		center.conf.Del(storage.K_ROOM_TOKEN)
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

func (center *central) onViewRoom(cmd *wsio.FromServerCommand) {
	center.onStatusChange(READY)
	center.ctrlConn.Send(cmd.ToManyObj(kIcIds, center.Connectors.Ids()))
	center.Connectors.ViewRoom(cmd)
}
func (center *central) sendViewIpcam(e *connector.Event) {
	center.ctrlConn.Send(e.Cmd.ToManyObj(kIc, e.Ic.Map(ipcam.TAG_VIEW)))
}

// Content => SetterIpcam
func (center *central) onManageSetIpcam(cmd *wsio.FromServerCommand) {
	var data ipcam.SetterIpcam
	if err := json.Unmarshal(cmd.Value(), &data); err != nil {
		center.ctrlConn.Send(cmd.ToManyInfo("Cannot parse ipcam"))
		return
	}
	center.Connectors.Save(cmd, data)
}
func (center *central) sendChIcId(e *connector.ChIdEvent) {
	center.ctrlConn.Send(wsio.BcObj(kIcIdCh, e))
}

// Content => id
func (center *central) onManageGetIpcam(cmd *wsio.FromServerCommand) {
	center.Connectors.Get(cmd, string(cmd.Value()))
}
func (center *central) sendMgrIpcam(e *connector.Event) {
	center.ctrlConn.Send(e.Cmd.ToManyObj(kSecIc, e.Ic.Map()))
}
func (center *central) sendMgrIpcamNotFound(e *connector.Event) {
	center.ctrlConn.Send(e.Cmd.ToManyJSON(kNoIc, []byte(e.Ic.Id)))
}

// Content => Ipcam.Id
func (center *central) onManageDelIpcam(cmd *wsio.FromServerCommand) {
	center.Connectors.Del(cmd, string(cmd.Value()))
}
func (center *central) broadcastDelIpcam(e *connector.Event) {
	center.ctrlConn.Send(wsio.BcJSON(kXIc, []byte(e.Ic.Id)))
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
