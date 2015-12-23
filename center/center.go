package center

import (
	"net/http"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/websocket"

	"github.com/empirefox/ic-client-one-wrap"
	"github.com/empirefox/ic-client-one/ipcam"
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

	conf       storage.Conf
	Conductor  rtc.Conductor
	gangStatus chan gangStatus
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

		quit:       make(chan struct{}),
		conf:       *conf,
		gangStatus: make(chan gangStatus, 8),
	}
	center.Conductor = rtc.NewConductor(center)

	return center, nil
}

func (center *central) preRun() {
	glog.Infoln("preRun")
	center.onConnectCtrl()
	center.onRegistryOfflines(true)
	center.Conductor.AddIceServer("stun:23.21.150.121", "", "")
	center.Conductor.AddIceServer("stun:stun.fwdnet.net", "", "")
	center.Conductor.AddIceServer("stun:stun.ideasip.com", "", "")
	center.Conductor.AddIceServer("stun:stun.anyfirewall.com:3478", "", "")
	center.Conductor.AddIceServer("stun:stun.voxgratia.org", "", "")
	center.Conductor.AddIceServer("stun:stun.ekiga.net", "", "")
	center.Conductor.AddIceServer("stun:stun.iptel.org", "", "")
	center.Conductor.AddIceServer("stun:stun.schlund.de", "", "")
	center.Conductor.AddIceServer("stun:stun.voiparound.com", "", "")
	center.Conductor.AddIceServer("stun:stun.voipbuster.com", "", "")
	center.Conductor.AddIceServer("stun:stun.voipstunt.com", "", "")
	center.Conductor.AddIceServer("stun:stun.voxgratia.org", "", "")
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

		case cmd := <-center.serverCommand:
			center.onServerCommand(cmd)

		case cmd := <-center.localCommand:
			center.onLocalCommand(cmd)

		case gs := <-center.gangStatus:
			center.onGangStatus(gs.id, gs.status)

		case <-ticker.C:
			center.onConnectCtrl()
			center.onRegistryOfflines(false)

		case <-center.quit:
			return
		}
	}
}

func (center *central) AddStatusObserver(c Ws) {
	center.addStatusObserver <- c
}
func (center *central) onAddStatusObserver(c Ws) {
	center.statusObservers[c] = true
}

func (center *central) DelStatusObserver(c Ws) {
	center.delStatusObserver <- c
}
func (center *central) onDelStatusObserver(c Ws) {
	delete(center.statusObservers, c)
}

func (center *central) ChangeStatus(status []byte) {
	center.changeStatus <- status
}
func (center *central) onStatusChange(status []byte) {
	if status != nil {
		center.status = status
	}
	for c := range center.statusObservers {
		c.Send(center.status)
	}
}

func (center *central) ChangeNoStatus(status []byte) {
	center.changeNoStatus <- status
}
func (center *central) onChangeNoStatus(status []byte) {
	for c := range center.statusObservers {
		c.Send(status)
	}
}

func (center *central) Conf() *storage.Conf {
	return &center.conf
}

func (center *central) Quiter() chan struct{} {
	return center.quit
}

func (center *central) OnConnectCtrl() {
	center.connectCtrl <- struct{}{}
}
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

func (center *central) SetCtrl(c Ws) {
	center.setCtrl <- c
}
func (center *central) onSetCtrl(c Ws) {
	center.hasCtrl = true
	center.ctrlConn = c
}

func (center *central) DelCtrl(c Ws) {
	center.delCtrl <- c
}
func (center *central) onDelCtrl(c Ws) {
	if center.ctrlConn == c {
		center.hasCtrl = false
		center.ctrlConn = nil
	}
}

func (center *central) SendCtrl(msg []byte) {
	center.ctrlSender <- msg
}
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

func (center *central) registry(i *ipcam.Ipcam, force bool) (changed bool) {
	if i.Off || (i.Online && !force) {
		return
	}
	info, isOnline := center.Conductor.Registry(i.Id, i.Url, center.conf.GetRecPrefix(i.Id), i.Rec, i.AudioOff)

	changed = i.Online != isOnline || i.Width != info.Width || i.Height != info.Height ||
		i.HasVideo != info.Video || i.HasAudio != info.Audio
	if changed {
		i.Online, i.Width, i.Height, i.HasVideo, i.HasAudio = isOnline, info.Width, info.Height, info.Video, info.Audio
		if err := center.conf.PutIpcam(i); err != nil {
			glog.Errorln(err)
		}
	}
	return
}

func (center *central) onRegistryOfflines(force bool) {
	var changed = false
	for _, i := range center.conf.GetIpcams() {
		// registry must be called
		changed = center.registry(&i, force) || changed
	}
	if changed && center.hasCtrl {
		center.onSendIpcams()
	}
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
type gangStatus struct {
	id     []byte
	status uint
}

func (center *central) OnGangStatus(id string, status uint) {
	center.gangStatus <- gangStatus{id: []byte(id), status: status}
}
func (center *central) onGangStatus(id []byte, status uint) {
	i, err := center.conf.GetIpcam(id)
	if err != nil {
		glog.Errorln("camera not found:", err)
		return
	}
	var isOnline bool
	switch status {
	case rtc.ALIVE:
		isOnline = true
	case rtc.DEAD:
		isOnline = false
	}
	if i.Online == isOnline {
		return
	}
	i.Online = isOnline
	if err := center.conf.PutIpcam(&i); err != nil {
		glog.Errorln(err)
		return
	}
	if center.hasCtrl {
		center.onSendIpcams()
	}
}
