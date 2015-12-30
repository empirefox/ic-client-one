package connector

import (
	"github.com/empirefox/ic-client-one-wrap"
	"github.com/empirefox/ic-client-one/ipcam"
	"github.com/empirefox/ic-client-one/wsio"
)

const (
	StatusChanged = iota
	StatusNoChange
	SaveFailed
	GetOk
	IcNotFound
	DelOk
	DelFailed
	RecChanged
)

type Event struct {
	Type int
	Cmd  *wsio.FromServerCommand
	Ic   ipcam.Ipcam
	Msg  string
}

type ChIdEvent struct {
	New string
	Old string
}

type SaveData struct {
	Cmd    *wsio.FromServerCommand
	Setter ipcam.SetterIpcam
}

type regEndData struct {
	cmd  *wsio.FromServerCommand
	i    ipcam.Ipcam
	info rtc.IpcamAvInfo
}

type gangStatusData struct {
	ok bool
}
