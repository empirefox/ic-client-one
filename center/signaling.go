package center

import (
	"errors"
	"fmt"

	"github.com/golang/glog"

	"github.com/empirefox/ic-client-one-wrap"
	"github.com/empirefox/ic-client-one/ipcam"
	"github.com/empirefox/ic-client-one/wsio"
)

type Signal struct {
	Camera string `json:"camera,omitempty"`
	Type   string `json:"type,omitempty"`

	Candidate string `json:"candidate,omitempty"`
	Id        string `json:"id,omitempty"`
	Label     int    `json:"label,omitempty"`

	Sdp string `json:"sdp,omitempty"`
}

// From => ClientId
// Content => Camera
// cmd from signaling-server many.go CreateSignalingConnectionCommand
func (center *central) OnCreateSignalingConnection(cmd *wsio.FromServerCommand) {
	defer func() {
		if err := recover(); err != nil {
			glog.Errorln(err)
		}
	}()

	socket, _, err := center.Dial(center.conf.SignalingUrl(string(cmd.Value())), nil)
	if err != nil {
		glog.Errorln(err)
		center.SendCtrl(cmd.ToManyInfo("Dial signaling failed"))
		return
	}
	defer socket.Close()
	ws := NewConn(center, socket, center.quit)
	go ws.WriteClose()
	center.onSignalingConnected(ws)
}

type Camera struct {
	ipcam.Ipcam
	center *central
	ws     Ws
	pc     rtc.PeerConn
}

func (c *Camera) onOffer(signal *Signal) error {
	glog.Infoln("creating peer")
	if !c.Online {
		return errors.New("Camera not online")
	}
	if c.pc = c.center.Conductor.CreatePeer(c.Id, c.ws.Send); c.pc.IsZero() {
		return errors.New("Create peer failed")
	}
	c.pc.CreateAnswer(signal.Sdp)
	return nil
}

func (c *Camera) onCandidate(signal *Signal) {
	glog.Infoln("add candidate")
	c.pc.AddCandidate(signal.Candidate, signal.Id, signal.Label)
}

func (c *Camera) close() {
	if !c.pc.IsZero() {
		glog.Infoln("deleting peer")
		c.center.Conductor.DeletePeer(c.pc)
	}
}

func (center *central) onSignalingConnected(ws Ws) {
	cs := make(map[string]*Camera)
	defer func() {
		glog.Infoln("onSignalingConnected finished")
		for _, c := range cs {
			c.close()
		}
	}()

	for {
		signal := &Signal{}
		if err := ws.ReadJSON(signal); err != nil {
			glog.Errorln(err)
			return
		}

		c, exist := cs[signal.Camera]
		switch signal.Type {
		case "offer":
			if exist {
				return
			}
			ch := make(chan ipcam.Ipcam)
			center.Connectors.CopyOf(signal.Camera, ch)
			i, ok := <-ch
			if !ok {
				ws.Send([]byte(`{"error":"Camera not found"}`))
				return
			}
			c = &Camera{Ipcam: i, center: center, ws: ws}
			if err := c.onOffer(signal); err != nil {
				ws.Send([]byte(fmt.Sprintf(`{"error":"%s"}`, err)))
				return
			}
			cs[signal.Camera] = c
		case "candidate":
			if !exist {
				return
			}
			c.onCandidate(signal)
		case "bye":
			if !exist {
				return
			}
			c.close()
			delete(cs, signal.Camera)
		default:
			glog.Errorln("Unknow signal json")
		}

	}
}
