package center

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/websocket"

	"github.com/empirefox/ic-client-one-wrap"
	. "github.com/empirefox/ic-client-one/config"
	. "github.com/empirefox/ic-client-one/utils"
)

// From One: "ManageGetIpcam", "ManageSetIpcam", "ManageReconnectIpcam"
type Command struct {
	From    uint   `json:"from"`
	Name    string `json:"name"`
	Content string `json:"content"`
}

type Center struct {
	status               string
	statusReciever       map[*Connection]bool
	AddStatusReciever    chan *Connection
	RemoveStatusReciever chan *Connection
	ChangeStatus         chan string
	Quit                 chan bool
	QuitWaitGroup        sync.WaitGroup
	CtrlConn             *Connection
	ctrlConnMutex        sync.Mutex
	Conf                 Config
	Upgrader             Upgrader
	Dialer               Dialer
	Conductor            rtc.Conductor
}

func NewCenter() *Center {
	conf := NewConfig()
	checkOrigin := func(r *http.Request) bool {
		u, err := url.Parse(r.Header["Origin"][0])
		if err != nil {
			return false
		}
		return u.Host == conf.Server
	}

	return &Center{
		statusReciever:       make(map[*Connection]bool),
		AddStatusReciever:    make(chan *Connection),
		RemoveStatusReciever: make(chan *Connection),
		ChangeStatus:         make(chan string),
		Quit:                 make(chan bool),
		Conf:                 *conf,
		Upgrader: &websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin:     checkOrigin,
		},
		Dialer: &websocket.Dialer{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Conductor: rtc.NewConductor(),
	}
}

func (center *Center) preRun() {
	center.onRegistryOfflines()
	center.Conductor.AddIceServer("stun:stun.l.google.com:19302", "", "")
	center.Conductor.AddIceServer("stun:stun.anyfirewall.com:3478", "", "")
	center.Conductor.AddIceServer("turn:turn.bistri.com:80", "homeo", "homeo")
	center.Conductor.AddIceServer("turn:turn.anyfirewall.com:443?transport=tcp", "webrtc", "webrtc")
}

func (center *Center) postRun() {
	center.Conductor.Release()
}

func (center *Center) run() {
	ticker := time.NewTicker(center.Conf.PingPeriod)
	defer func() {
		ticker.Stop()
	}()
	for {
		select {
		case c := <-center.AddStatusReciever:
			center.statusReciever[c] = true
		case c := <-center.RemoveStatusReciever:
			if _, ok := center.statusReciever[c]; ok {
				delete(center.statusReciever, c)
				close(c.Send)
			}
		case center.status = <-center.ChangeStatus:
			status, err := center.GetStatus()
			if err != nil {
				continue
			}
			for c := range center.statusReciever {
				select {
				case c.Send <- status:
				default:
					close(c.Send)
					delete(center.statusReciever, c)
				}
			}
		case <-ticker.C:
			center.onRegistryOfflines()
		case <-center.Quit:
			return
		}
	}
}

func (center *Center) onRegistryOfflines() {
	var changed = false
	for i, _ := range center.Conf.Ipcams {
		cam := &center.Conf.Ipcams[i]
		isOnline := cam.Online || (!cam.Off && center.Conductor.Registry(cam.Url))
		changed = !cam.Online && isOnline
		cam.Online = isOnline
	}
	if changed {
		center.OnGetIpcams()
	}
}

func (center *Center) Start() {
	center.QuitWaitGroup.Add(1)
	go center.Run()
}

func (center *Center) Run() {
	defer center.QuitWaitGroup.Done()
	center.preRun()
	defer center.postRun()
	center.run()
}

func (center *Center) Close() {
	close(center.Quit)
	center.QuitWaitGroup.Wait()
}

func (center *Center) GetStatus() ([]byte, error) {
	statusMap := map[string]string{"type": "Status", "content": center.status}
	return json.Marshal(statusMap)
}

func (center *Center) AddCtrlConn(c *Connection) {
	center.ctrlConnMutex.Lock()
	defer center.ctrlConnMutex.Unlock()
	center.CtrlConn = c
	center.AddStatusReciever <- c
}

func (center *Center) RemoveCtrlConn() {
	center.ctrlConnMutex.Lock()
	defer center.ctrlConnMutex.Unlock()
	center.RemoveStatusReciever <- center.CtrlConn
	center.CtrlConn = nil
}

func (center *Center) OnGetIpcams() {
	ipcams := make(map[string]Ipcam, len(center.Conf.Ipcams))
	for _, ipcam := range center.Conf.Ipcams {
		ipcams[ipcam.Id] = ipcam
	}
	info, err := json.Marshal(ipcams)
	if err != nil {
		glog.Errorln(err)
		return
	}

	center.ctrlConnMutex.Lock()
	defer center.ctrlConnMutex.Unlock()
	if center.CtrlConn == nil {
		glog.Errorln("No control connection")
		return
	}
	center.CtrlConn.Send <- append([]byte("one:Ipcams:"), info...)
}

// Content => id
func (center *Center) OnManageReconnectIpcam(cmd *Command) {
	for i, _ := range center.Conf.Ipcams {
		if cam := &center.Conf.Ipcams[i]; cam.Id == cmd.Content {
			cam.Online = !cam.Off && center.Conductor.Registry(cam.Url)
			if cam.Online {
				center.OnGetIpcams()
				return
			}
			center.CtrlConn.Send <- GenInfoMessage(cmd.From, "Failed to reconnect ipcam")
			return
		}
	}
	center.CtrlConn.Send <- GenInfoMessage(cmd.From, "Cannot find ipcam")
}

func (center *Center) OnSetSecretAddress(addr string) {
	if err := center.Conf.SetAddr(addr); err != nil {
		return
	}
	center.CtrlConn.Close()
}

// Content => id
func (center *Center) OnManageGetIpcam(cmd *Command) {
	ipcam, err := center.Conf.GetIpcam(cmd.Content)
	if err != nil {
		center.CtrlConn.Send <- GenInfoMessage(cmd.From, "Cannot get ipcam")
		return
	}
	msg, err := GenCtrlResMessage(cmd.From, cmd.Name, GetManaged(ipcam))
	if err != nil {
		center.CtrlConn.Send <- GenInfoMessage(cmd.From, "Cannot get ipcam content")
		return
	}
	center.CtrlConn.Send <- msg
}

// Content => Ipcam
func (center *Center) OnManageSetIpcam(cmd *Command) {
	var ipcam ManageIpcam
	if err := json.Unmarshal([]byte(cmd.Content), &ipcam); err != nil {
		center.CtrlConn.Send <- GenInfoMessage(cmd.From, "Cannot parse ipcam")
		return
	}
	if err := center.Conf.SaveIpcam(ipcam.Get()); err != nil {
		center.CtrlConn.Send <- GenInfoMessage(cmd.From, "Cannot get ipcam")
		return
	}
	center.OnGetIpcams()
}
