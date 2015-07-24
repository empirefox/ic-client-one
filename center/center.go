package center

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/websocket"

	"github.com/empirefox/ic-client-one-wrap"
	"github.com/empirefox/ic-client-one/ipcam"
	. "github.com/empirefox/ic-client-one/storage"
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
	Conf                 Conf
	Upgrader             Upgrader
	Dialer               Dialer
	Conductor            rtc.Conductor
}

func NewCenter(cpath ...string) *Center {
	conf := NewConf(cpath...)
	checkOrigin := func(r *http.Request) bool {
		if r.Header["Origin"][0] == "file://" {
			return true
		}
		u, err := url.Parse(r.Header["Origin"][0])
		if strings.HasPrefix(u.Host, "127.0.0.1:") {
			// port 80/443 not supported
			return true
		}
		if err != nil {
			glog.Infoln(u.Host, conf.GetServer())
			return false
		}
		return u.Host == conf.GetServer()
	}

	return &Center{
		statusReciever:       make(map[*Connection]bool),
		AddStatusReciever:    make(chan *Connection, 1),
		RemoveStatusReciever: make(chan *Connection, 1),
		ChangeStatus:         make(chan string),
		Quit:                 make(chan bool),
		Conf:                 conf,
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
	glog.Infoln("preRun")
	center.onRegistryOfflines(true)
	center.Conductor.AddIceServer("stun:stun.l.google.com:19302", "", "")
	center.Conductor.AddIceServer("stun:stun.anyfirewall.com:3478", "", "")
	center.Conductor.AddIceServer("turn:turn.bistri.com:80", "homeo", "homeo")
	center.Conductor.AddIceServer("turn:turn.anyfirewall.com:443?transport=tcp", "webrtc", "webrtc")
}

func (center *Center) postRun() {
	glog.Infoln("postRun")
	center.Conductor.Release()
}

func (center *Center) run() {
	glog.Infoln("run")
	ticker := time.NewTicker(center.Conf.GetPingPeriod())
	defer func() {
		ticker.Stop()
	}()
	for {
		select {
		case c := <-center.AddStatusReciever:
			center.statusReciever[c] = true
		case c := <-center.RemoveStatusReciever:
			if _, ok := center.statusReciever[c]; ok {
				center.removeStatusReciever(c)
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
					center.removeStatusReciever(c)
				}
			}
		case <-ticker.C:
			center.onRegistryOfflines(false)
		case <-center.Quit:
			return
		}
	}
}

func (center *Center) removeStatusReciever(c *Connection) {
	defer func() { recover() }()
	delete(center.statusReciever, c)
	close(c.Send)
}

// return isOnline
func (center *Center) registry(i ipcam.Ipcam, force bool) bool {
	if i.Off {
		return false
	}
	if i.Online && !force {
		return true
	}
	return center.Conductor.Registry(i.Url, center.Conf.GetRecPrefix(i.Id), i.Rec)
}

func (center *Center) onRegistryOfflines(force bool) {
	var changed = false
	for _, i := range center.Conf.GetIpcams() {
		// registry must be called
		isOnline := center.registry(i, force)
		ichanged := !i.Online && isOnline
		changed = changed || ichanged
		i.Online = isOnline
		if ichanged {
			if err := center.Conf.PutIpcam(&i); err != nil {
				glog.Errorln(err)
			}
		}
	}
	if changed {
		center.OnGetIpcams()
	}
	glog.Infoln("onRegistryOfflines ok")
}

func (center *Center) Start() error {
	center.QuitWaitGroup.Add(1)
	if err := center.Conf.Open(); err != nil {
		return err
	}
	go center.Run()
	return nil
}

func (center *Center) Run() {
	defer func() {
		center.QuitWaitGroup.Done()
		center.Conf.Close()
		center.postRun()
	}()
	center.preRun()
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
	info, err := json.Marshal(center.Conf.GetIpcams())
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
	glog.Infoln("OnGetIpcams ok")
}

// Content => id
func (center *Center) OnManageReconnectIpcam(cmd *Command) {
	cam, err := center.Conf.GetIpcam([]byte(cmd.Content))
	if err != nil {
		center.CtrlConn.Send <- GenInfoMessage(cmd.From, "Cannot find ipcam")
	}
	cam.Online = center.registry(cam, true)
	if !cam.Online {
		center.CtrlConn.Send <- GenInfoMessage(cmd.From, "Failed to reconnect ipcam")
		return
	}
	center.OnGetIpcams()
}

func (center *Center) OnSetSecretAddress(addr []byte) {
	if err := center.Conf.Put(K_SEC_ADDR, addr); err != nil {
		return
	}
	center.CtrlConn.Close()
}

func (center *Center) OnRemoveRoom() {
	center.CtrlConn.Send <- GenServerCommand("RemoveRoom", "")
	center.Conf.Put(K_SEC_ADDR, nil)
}

// Content => id
func (center *Center) OnManageGetIpcam(cmd *Command) {
	ipcam, err := center.Conf.GetIpcam([]byte(cmd.Content))
	if err != nil {
		center.CtrlConn.Send <- GenInfoMessage(cmd.From, "Cannot get ipcam")
		return
	}
	msg, err := GenCtrlResMessage(cmd.From, cmd.Name, ipcam.Map())
	if err != nil {
		center.CtrlConn.Send <- GenInfoMessage(cmd.From, "Cannot get ipcam content")
		return
	}
	center.CtrlConn.Send <- msg
}

// Content => SetterIpcam
func (center *Center) OnManageSetIpcam(cmd *Command) {
	var data ipcam.SetterIpcam
	if err := json.Unmarshal([]byte(cmd.Content), &data); err != nil {
		center.CtrlConn.Send <- GenInfoMessage(cmd.From, "Cannot parse ipcam")
		return
	}
	if err := center.Conf.PutIpcam(&data.Ipcam, data.Target); err != nil {
		center.CtrlConn.Send <- GenInfoMessage(cmd.From, "Cannot get ipcam")
		return
	}
	center.OnGetIpcams()
}
