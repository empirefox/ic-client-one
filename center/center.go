package center

import (
	"net/http"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/websocket"

	"github.com/empirefox/ic-client-one-wrap"
	"github.com/empirefox/ic-client-one/connector"
	"github.com/empirefox/ic-client-one/storage"
	"github.com/empirefox/ic-client-one/wsio"
)

type central struct {
	websocket.Upgrader
	websocket.Dialer

	status            []byte
	statusObservers   map[Ws]bool
	addStatusObserver chan Ws
	delStatusObserver chan Ws
	changeStatus      chan []byte
	changeNoStatus    chan []byte

	quit          chan struct{}
	quitWaitGroup sync.WaitGroup

	connectCtrl   chan struct{}
	ctrlConn      Ws
	hasCtrl       bool
	setCtrl       chan Ws
	delCtrl       chan Ws
	ctrlSender    chan []byte
	serverCommand chan *wsio.FromServerCommand
	localCommand  chan *FromLocalCommand
	cntrEnt       chan *connector.Event
	chIdEnt       chan *connector.ChIdEvent

	conf             *storage.Conf
	Conductor        rtc.Conductor
	ConnectorFactory *connector.ConnectorFactory
	Connectors       *connector.Connectors
}

func NewCentral(setup string) (Central, error) {
	conf, err := storage.NewConf(setup)
	if err != nil {
		return nil, err
	}

	center := &central{
		Upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header["Origin"]
				if len(origin) == 0 {
					return false
				}
				switch origin[0] {
				case "http://ic.client", "file://":
					return true
				}
				return false
			},
		},
		Dialer: websocket.Dialer{},

		status:            DISCONNECTED,
		statusObservers:   make(map[Ws]bool),
		addStatusObserver: make(chan Ws, 1),
		delStatusObserver: make(chan Ws, 1),
		changeStatus:      make(chan []byte, 64),
		changeNoStatus:    make(chan []byte, 64),

		connectCtrl:   make(chan struct{}, 1),
		setCtrl:       make(chan Ws, 1),
		delCtrl:       make(chan Ws, 1),
		ctrlSender:    make(chan []byte, 64),
		serverCommand: make(chan *wsio.FromServerCommand, 64),
		localCommand:  make(chan *FromLocalCommand, 64),
		cntrEnt:       make(chan *connector.Event, 1),
		chIdEnt:       make(chan *connector.ChIdEvent, 1),

		quit: make(chan struct{}),
		conf: conf,
	}
	center.Conductor = rtc.NewConductor(center)
	center.ConnectorFactory = &connector.ConnectorFactory{
		Conf:        center.conf,
		Conductor:   center.Conductor,
		ChanQuit:    center.quit,
		OnEvent:     center.OnConnectorEvnet,
		OnIdChanged: center.OnIcIdChanged,
	}

	return center, nil
}

func (center *central) preRun() {
	glog.Infoln("preRun")
	center.onConnectCtrl()
	center.Connectors = center.ConnectorFactory.NewConnectors()
	center.Connectors.Start()
	for _, stun := range center.conf.GetStuns() {
		center.Conductor.AddIceServer(stun, "", "")
	}
}

func (center *central) postRun() {
	glog.Infoln("postRun")
	center.Conductor.Release()
}

func (center *central) run() {
	glog.Infoln("run")
	ticker := time.NewTicker(center.conf.GetPingSecond() / 4)
	defer func() {
		ticker.Stop()
	}()
	for {
		select {
		case c := <-center.addStatusObserver:
			center.onAddStatusObserver(c)

		case c := <-center.delStatusObserver:
			center.onDelStatusObserver(c)

		case status := <-center.changeStatus:
			center.onStatusChange(status)

		case status := <-center.changeNoStatus:
			center.onChangeNoStatus(status)

		case _, ok := <-center.connectCtrl:
			if ok {
				center.onConnectCtrl()
			}

		case c := <-center.setCtrl:
			center.onSetCtrl(c)

		case c := <-center.delCtrl:
			center.onDelCtrl(c)

		case e := <-center.cntrEnt:
			center.onConnectorEvnet(e)

		case e := <-center.chIdEnt:
			center.onIcIdChanged(e)

		case cmd := <-center.serverCommand:
			center.onServerCommand(cmd)

		case cmd := <-center.localCommand:
			center.onLocalCommand(cmd)

		case <-ticker.C:
			center.onConnectCtrl()

		case <-center.quit:
			return
		}
	}
}

func (center *central) AddStatusObserver(c Ws)   { center.addStatusObserver <- c }
func (center *central) onAddStatusObserver(c Ws) { center.statusObservers[c] = true }

func (center *central) DelStatusObserver(c Ws)   { center.delStatusObserver <- c }
func (center *central) onDelStatusObserver(c Ws) { delete(center.statusObservers, c) }

func (center *central) ChangeStatus(status []byte) { center.changeStatus <- status }
func (center *central) onStatusChange(status []byte) {
	if status != nil {
		center.status = status
	}
	for c := range center.statusObservers {
		c.Send(center.status)
	}
}

func (center *central) ChangeNoStatus(status []byte) { center.changeNoStatus <- status }
func (center *central) onChangeNoStatus(status []byte) {
	for c := range center.statusObservers {
		c.Send(status)
	}
}

func (center *central) Conf() *storage.Conf   { return center.conf }
func (center *central) Quiter() chan struct{} { return center.quit }

func (center *central) OnConnectCtrl() { center.connectCtrl <- struct{}{} }
func (center *central) onConnectCtrl() {
	if center.hasCtrl {
		center.onStatusChange(center.status)
		return
	}
	center.onStatusChange(CONNECTING)
	socket, _, err := center.Dial(center.conf.CtrlUrl(), nil)
	if err != nil {
		glog.Errorln(err)
		center.onStatusChange(UNREACHABLE)
		return
	}

	ws := NewConn(center, socket, center.quit)
	center.onSetCtrl(ws)

	go center.readCtrl(ws)
	go ws.WriteClose()
	center.onDoLogin()
}

func (center *central) SetCtrl(c Ws) { center.setCtrl <- c }
func (center *central) onSetCtrl(c Ws) {
	center.hasCtrl = true
	center.ctrlConn = c
}

func (center *central) DelCtrl(c Ws) { center.delCtrl <- c }
func (center *central) onDelCtrl(c Ws) {
	if center.ctrlConn == c {
		center.hasCtrl = false
		center.ctrlConn = nil
	}
}

func (center *central) SendCtrl(msg []byte) { center.ctrlSender <- msg }
func (center *central) sendCtrl(msg []byte) {
	if center.hasCtrl {
		center.ctrlConn.Send(msg)
	}
}

func (center *central) closeCtrl() {
	if center.hasCtrl {
		center.ctrlConn.Close()
	}
}

func (center *central) OnConnectorEvnet(e *connector.Event) { center.cntrEnt <- e }
func (center *central) onConnectorEvnet(e *connector.Event) {
	if center.hasCtrl {
		switch e.Type {
		case connector.StatusChanged:
			center.sendViewIpcam(e)

		case connector.RecChanged:
			center.sendLocalCamera(e)

		case connector.StatusNoChange:
		case connector.SaveFailed:

		case connector.GetOk:
			center.sendMgrIpcam(e)

		case connector.IcNotFound:
			center.sendMgrIpcamNotFound(e)

		case connector.DelOk:
			center.broadcastDelIpcam(e)

		case connector.DelFailed:
		}
		if e.Cmd != nil && e.Msg != "" {
			center.ctrlConn.Send(e.Cmd.ToManyInfo(e.Msg))
		}
	}
	// For local
	switch e.Type {
	case connector.RecChanged:
		center.sendLocalCamera(e)
	}
}

func (center *central) OnIcIdChanged(e *connector.ChIdEvent) { center.chIdEnt <- e }
func (center *central) onIcIdChanged(e *connector.ChIdEvent) {
}

func (center *central) Start() error {
	if err := center.conf.Open(); err != nil {
		return err
	}
	center.quitWaitGroup.Add(1)
	go center.start()
	return nil
}

func (center *central) start() {
	defer center.quitWaitGroup.Done()
	defer func() {
		center.conf.Close()
		center.postRun()
	}()
	center.preRun()
	center.run()
}

func (center *central) Close() {
	close(center.quit)
	center.quitWaitGroup.Wait()
}

// implement rtc.StatusObserver
func (center *central) OnGangStatus(id string, status uint) {
	center.Connectors.OnGangStatus(id, status)
}
