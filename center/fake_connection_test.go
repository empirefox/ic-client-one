package center

import (
	"net/http"

	"github.com/empirefox/ic-client-one-wrap"
	"github.com/gorilla/websocket"
)

type fakeUpgrader struct {
}

func (fakeUpgrader) Upgrade(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (*websocket.Conn, error) {
	return &websocket.Conn{}, nil
}

type fakeDialer struct {
}

func (fakeDialer) Dial(urlStr string, requestHeader http.Header) (*websocket.Conn, *http.Response, error) {
	return &websocket.Conn{}, nil, nil
}

func newFakeConn(center *Center, msg string) *Connection {
	return &Connection{
		Ws:     newFakeWsConn(websocket.TextMessage, msg),
		Send:   make(chan []byte),
		Center: center,
	}
}

type fakePeer struct{}

func (fakePeer) Delete()                                {}
func (fakePeer) CreateAnswer(sdp string)                {}
func (fakePeer) AddCandidate(sdp, mid string, line int) {}

type fakeConductor struct{}

func (fakeConductor) Release()                                             {}
func (fakeConductor) Registry(url, recName string, recEnabled bool) bool   { return true }
func (fakeConductor) SetRecordEnabled(url string, recEnabled bool)         {}
func (fakeConductor) CreatePeer(url string, send chan []byte) rtc.PeerConn { return &fakePeer{} }
func (fakeConductor) DeletePeer(pc rtc.PeerConn)                           {}
func (fakeConductor) AddIceServer(uri, name, psd string)                   {}
