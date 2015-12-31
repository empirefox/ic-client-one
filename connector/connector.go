package connector

import (
	"strconv"
	"time"

	"github.com/empirefox/ic-client-one-wrap"
	"github.com/empirefox/ic-client-one/ipcam"
	"github.com/empirefox/ic-client-one/storage"
	"github.com/empirefox/ic-client-one/wsio"
	"github.com/golang/glog"
)

type Connector struct {
	cs        *Connectors
	Conf      *storage.Conf
	Conductor rtc.Conductor

	saveData *SaveData
	i        ipcam.Ipcam
	force    bool
	reging   bool
	deleted  bool
	delCmd   *wsio.FromServerCommand

	ChanView   chan *wsio.FromServerCommand
	ChanSave   chan *SaveData
	ChanGet    chan *wsio.FromServerCommand
	ChanDel    chan *wsio.FromServerCommand
	ChanQuit   chan struct{}
	chanQuit   chan struct{}
	chanReg    chan *wsio.FromServerCommand
	chanEndReg chan regEndData
	chanUnreg  chan string
	chanGs     chan gangStatusData
	chanCopy   chan (chan<- ipcam.Ipcam)
	chanRec    chan bool
	chanLbc    chan struct{}

	OnEvent     func(e *Event)
	OnIdChanged func(e *ChIdEvent)
}

func (c *Connector) Run() {
	ticker := time.NewTicker(time.Second * 20)
	defer func() {
		ticker.Stop()
	}()

	for {
		select {
		case cmd := <-c.ChanView:
			c.onView(cmd)

		case data := <-c.ChanSave:
			c.onSave(data)

		case rec := <-c.chanRec:
			c.onRec(rec)

		case <-c.chanLbc:
			c.onLocalBroadcast()

		case cmd := <-c.ChanGet:
			c.onGet(cmd)

		case cmd := <-c.ChanDel:
			c.onDel(cmd)

		case <-ticker.C:
			c.goReging(nil)
		case cmd := <-c.chanReg:
			c.goReging(cmd)
		case data := <-c.chanEndReg:
			c.onRegEnd(data)

		case id := <-c.chanUnreg:
			c.unregistry(id)

		case data := <-c.chanGs:
			c.onGangStatus(data)

		case ch := <-c.chanCopy:
			c.onCopyOf(ch)

		case <-c.ChanQuit:
			return
		case <-c.chanQuit:
			return
		}
	}
}

func (c *Connector) onView(cmd *wsio.FromServerCommand) {
	if !c.i.Off && c.i.Online {
		c.OnEvent(&Event{
			Type: StatusChanged,
			Cmd:  cmd,
			Ic:   c.i,
		})
	}
}

func (c *Connector) onLocalBroadcast() {
	c.OnEvent(&Event{
		Type: RecChanged,
		Ic:   c.i,
	})
}

func (c *Connector) onRec(rec bool) {
	if c.i.Rec != rec {
		err := c.Conf.SetIpcamAttr([]byte(c.i.Id), ipcam.K_IC_REC, []byte(strconv.FormatBool(rec)))
		if err != nil {
			// TODO report error?
			glog.Errorln(err)
			return
		}
		c.i.Rec = rec
		c.Conductor.SetRecordEnabled(c.i.Id, rec)
	}
	c.OnEvent(&Event{
		Type: RecChanged,
		Ic:   c.i,
	})
}

func (c *Connector) goReging(cmd *wsio.FromServerCommand) {
	if c.delCmd != nil {
		c.delNotify()
		return
	}
	if c.i.Off || (c.i.Online && !c.force) {
		return
	}
	if !c.reging {
		c.reging = true
		go c.registry(c.i, c.force, cmd)
	}
}

// run in standalone goroutine, need Ipcam copy
// TODO use force?
func (c *Connector) registry(i ipcam.Ipcam, force bool, cmd *wsio.FromServerCommand) {
	info := c.Conductor.Registry(i.Id, i.Url, c.Conf.GetRecPrefix(i.Id), i.Rec, i.AudioOff)
	c.chanEndReg <- regEndData{cmd: cmd, i: i, info: info}
}

func (c *Connector) onRegEnd(data regEndData) {
	if c.delCmd != nil {
		c.delNotify()
		return
	}
	c.reging = false

	if c.saveData != nil {
		c.unregistry(c.saveData.Setter.Target)
		c.i = c.saveData.Setter.Ipcam
		c.saveData = nil
		c.goReging(data.cmd)
		return
	}

	sameStatus := c.i.Online == data.info.Ok &&
		c.i.HasAudio == data.info.Audio && c.i.HasVideo == data.info.Video &&
		c.i.Width == data.info.Width && c.i.Height == data.info.Height
	if sameStatus {
		c.OnEvent(&Event{
			Type: StatusNoChange,
			Cmd:  data.cmd,
			Ic:   c.i,
			Msg:  "Not changed: " + c.i.Id,
		})
		return
	}

	c.i.Online = data.info.Ok
	c.i.HasAudio, c.i.HasVideo = data.info.Audio, data.info.Video
	c.i.Width, c.i.Height = data.info.Width, data.info.Height
	if err := c.Conf.PutIpcam(&c.i); err != nil {
		// TODO report error?
		glog.Errorln(err)
	}
	c.OnEvent(&Event{
		Type: StatusChanged,
		Cmd:  data.cmd,
		Ic:   c.i,
		Msg:  "Changes saved: " + c.i.Id,
	})
}

func (c *Connector) onSave(data *SaveData) {
	sameDevice := c.i.Id == data.Setter.Ipcam.Id && c.i.Url == data.Setter.Ipcam.Url &&
		c.i.AudioOff == data.Setter.Ipcam.AudioOff && c.i.Off == data.Setter.Ipcam.Off

	if sameDevice {
		c.OnEvent(&Event{
			Type: StatusNoChange,
			Cmd:  data.Cmd,
			Ic:   c.i,
			Msg:  "No need to change: " + c.i.Id,
		})
		return
	}

	if err := c.Conf.PutIpcam(&data.Setter.Ipcam, []byte(data.Setter.Target)); err != nil {
		glog.Errorln(err)
		c.OnEvent(&Event{
			Type: SaveFailed,
			Cmd:  data.Cmd,
			Ic:   c.i,
			Msg:  "Save failed: " + c.i.Id,
		})
		return
	}
	if data.Setter.Target != c.i.Id {
		c.cs.onSaved(data.Setter.Target, c)
		c.OnIdChanged(&ChIdEvent{New: c.i.Id, Old: data.Setter.Target})
	}

	if c.reging {
		c.saveData = data
		return
	}

	c.unregistry(data.Setter.Target)
	c.i = data.Setter.Ipcam
	c.goReging(data.Cmd)
}

func (c *Connector) onGet(cmd *wsio.FromServerCommand) {
	c.OnEvent(&Event{
		Type: GetOk,
		Cmd:  cmd,
		Ic:   c.i,
		Msg:  "Got ipcam: " + c.i.Id,
	})
}

func (c *Connector) onDel(cmd *wsio.FromServerCommand) {
	if err := c.Conf.RemoveIpcam([]byte(c.i.Id)); err != nil {
		c.OnEvent(&Event{
			Type: DelFailed,
			Cmd:  cmd,
			Ic:   c.i,
			Msg:  "Remove failed: " + c.i.Id,
		})
		return
	}
	c.cs.onDeleted(c.i.Id)
	if c.reging {
		c.delCmd = cmd
		return
	}
	c.delNotify()
}

func (c *Connector) unregistry(id string) {
	if id != "" {
		c.Conductor.UnRegistry(id)
	}
}

func (c *Connector) delNotify() {
	if c.deleted {
		return
	}
	c.deleted = true

	c.unregistry(c.i.Id)
	c.OnEvent(&Event{
		Type: DelOk,
		Cmd:  c.delCmd,
		Ic:   c.i,
		Msg:  "Ipcam removed: " + c.i.Id,
	})
	close(c.chanQuit)
}

func (c *Connector) onGangStatus(data gangStatusData) {
	if c.i.Online == data.ok {
		return
	}
	c.i.Online = data.ok
	c.OnEvent(&Event{
		Type: StatusChanged,
		Cmd:  new(wsio.FromServerCommand),
		Ic:   c.i,
	})
}

func (c *Connector) onCopyOf(ch chan<- ipcam.Ipcam) { ch <- c.i }
