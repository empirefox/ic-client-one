package connector

import (
	"sync"

	"github.com/empirefox/ic-client-one-wrap"
	"github.com/empirefox/ic-client-one/ipcam"
	"github.com/empirefox/ic-client-one/storage"
	"github.com/empirefox/ic-client-one/wsio"
)

type Connectors struct {
	f  *ConnectorFactory
	s  map[string]*Connector
	mu sync.Mutex
}

func (cs *Connectors) Start() {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	for _, c := range cs.s {
		go c.Run()
		c.chanReg <- nil
	}
}

func (cs *Connectors) ViewRoom(cmd *wsio.FromServerCommand) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	for _, c := range cs.s {
		c.ChanView <- cmd
	}
}

func (cs *Connectors) LocalBroadcast() {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	for _, c := range cs.s {
		c.chanLbc <- struct{}{}
	}
}

func (cs *Connectors) SetRec(id string, rec bool) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if c, ok := cs.s[id]; ok {
		c.chanRec <- rec
	}
}

func (cs *Connectors) Ids() (ids []string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	for id, _ := range cs.s {
		ids = append(ids, id)
	}
	return ids
}

func (cs *Connectors) Save(cmd *wsio.FromServerCommand, setter ipcam.SetterIpcam) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if c, ok := cs.s[setter.Target]; ok {
		c.ChanSave <- &SaveData{Cmd: cmd, Setter: setter}
	} else {
		// TODO send id list
		setter.Target = ""
		c = cs.f.NewConnector(cs, setter.Ipcam)
		cs.s[c.i.Id] = c
		go c.Run()
		c.chanReg <- cmd
	}
}
func (cs *Connectors) onSaved(id string, c *Connector) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	delete(cs.s, id)
	cs.s[c.i.Id] = c
}

func (cs *Connectors) CopyOf(id string, ch chan<- ipcam.Ipcam) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if c, ok := cs.s[id]; ok {
		c.chanCopy <- ch
	} else {
		close(ch)
	}
}

func (cs *Connectors) Get(cmd *wsio.FromServerCommand, id string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if c, ok := cs.s[id]; ok {
		c.ChanGet <- cmd
	} else {
		cs.f.OnEvent(&Event{
			Type: IcNotFound,
			Cmd:  cmd,
			Ic:   ipcam.Ipcam{Id: id},
			Msg:  "Ipcam not found: " + c.i.Id,
		})
	}
}

func (cs *Connectors) Del(cmd *wsio.FromServerCommand, id string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if c, ok := cs.s[id]; ok {
		c.ChanDel <- cmd
	} else {
		cs.f.OnEvent(&Event{
			Type: IcNotFound,
			Cmd:  cmd,
			Ic:   ipcam.Ipcam{Id: id},
			Msg:  "Ipcam not found: " + c.i.Id,
		})
	}
}
func (cs *Connectors) onDeleted(id string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	delete(cs.s, id)
}

// implement rtc.StatusObserver
func (cs *Connectors) OnGangStatus(id string, status uint) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	c, exist := cs.s[id]
	if !exist {
		c.chanUnreg <- id
		return
	}

	var ok bool
	switch status {
	case rtc.ALIVE:
		ok = true
	case rtc.DEAD:
		ok = false
	}

	c.chanGs <- gangStatusData{ok: ok}
}

type ConnectorFactory struct {
	Conf        *storage.Conf
	Conductor   rtc.Conductor
	ChanQuit    chan struct{}
	OnEvent     func(e *Event)
	OnIdChanged func(e *ChIdEvent)
}

func (f *ConnectorFactory) NewConnector(cs *Connectors, i ipcam.Ipcam) *Connector {
	return &Connector{
		cs:          cs,
		i:           i,
		Conf:        f.Conf,
		Conductor:   f.Conductor,
		ChanQuit:    f.ChanQuit,
		OnEvent:     f.OnEvent,
		OnIdChanged: f.OnIdChanged,

		ChanView:   make(chan *wsio.FromServerCommand, 1),
		ChanSave:   make(chan *SaveData, 1),
		ChanGet:    make(chan *wsio.FromServerCommand, 1),
		ChanDel:    make(chan *wsio.FromServerCommand, 1),
		chanQuit:   make(chan struct{}, 1),
		chanReg:    make(chan *wsio.FromServerCommand, 1),
		chanEndReg: make(chan regEndData, 1),
		chanGs:     make(chan gangStatusData, 1),
		chanCopy:   make(chan (chan<- ipcam.Ipcam), 1),
		chanRec:    make(chan bool, 1),
		chanLbc:    make(chan struct{}, 1),
	}
}

func (f *ConnectorFactory) NewConnectors() *Connectors {
	cs := &Connectors{f: f}
	s := make(map[string]*Connector)
	for id, i := range f.Conf.GetIpcams() {
		s[id] = f.NewConnector(cs, i)
	}
	cs.s = s
	return cs
}
