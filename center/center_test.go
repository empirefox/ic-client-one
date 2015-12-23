package center

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/empirefox/ic-client-one/ipcam"
	"github.com/empirefox/ic-client-one/storage"
	. "github.com/smartystreets/goconvey/convey"
)

func init() {
	flag.Set("stderrthreshold", "INFO")
}

func tempfile() string {
	f, _ := ioutil.TempFile("", "ic-client-one-center-db-")
	f.Close()
	os.Remove(f.Name())
	return f.Name()
}

const jsonContent = `{
		"DbPath":     "%s",
		"RecDir":     "/tmp/ic-client-one-rec-dir",
		"Server":     "http://127.0.0.1:9998",
		"TlsOn":      false,
		"PingSecond": 50
	}`

func newSetup() string {
	return fmt.Sprintf(jsonContent, tempfile())
}

type FakeCenter struct {
	*Center
	file              string
	calledOnGetIpcams bool
}

func newFakeCenter() *FakeCenter {
	c := storage.NewConf(newSetup())
	if err := c.Open(); err != nil {
		panic(err)
	}
	center := &Center{
		statusReciever:       make(map[*Connection]bool),
		AddStatusReciever:    make(chan *Connection),
		RemoveStatusReciever: make(chan *Connection),
		ChangeStatus:         make(chan string),
		Quit:                 make(chan bool),
		Conf:                 c,
		Upgrader:             &fakeUpgrader{},
		Dialer:               &fakeDialer{},
		Conductor:            &fakeConductor{},
	}
	return &FakeCenter{Center: center, file: file}
}

func (c *FakeCenter) Close() {
	defer os.Remove(c.file)
	c.Conf.Close()
}

func (c *FakeCenter) AddIpcam(id string) {
	if err := c.Conf.PutIpcam(&ipcam.Ipcam{Id: id}); err != nil {
		panic(err)
	}
}

func TestCenter_registry(t *testing.T) {
	Convey("registry", t, func() {
		Convey("should not register ipcam ok when off", func() {
			center := newFakeCenter()
			defer center.Close()

			ok := center.registry(ipcam.Ipcam{Off: true}, false)
			So(ok, ShouldBeFalse)

			ok = center.registry(ipcam.Ipcam{Off: true, Online: true}, false)
			So(ok, ShouldBeFalse)
		})
		Convey("should register ipcam ok when online", func() {
			center := newFakeCenter()
			defer center.Close()

			ok := center.registry(ipcam.Ipcam{}, false)
			So(ok, ShouldBeTrue)
		})
		Convey("should register ipcam ok when forced", func() {
			center := newFakeCenter()
			defer center.Close()

			ok := center.registry(ipcam.Ipcam{}, true)
			So(ok, ShouldBeTrue)

			ok = center.registry(ipcam.Ipcam{Online: true}, true)
			So(ok, ShouldBeTrue)
		})
	})
}

func TestCenter_onRegistryOfflines(t *testing.T) {
	Convey("onRegistryOfflines", t, func() {
		Convey("should registry and send change", func() {
			center := newFakeCenter()
			defer center.Close()

			center.AddIpcam("aid")
			send := make(chan []byte, 1)
			center.CtrlConn = &Connection{Send: send}

			size := len(center.Conf.GetIpcams())
			So(size, ShouldBeGreaterThan, 0)

			center.onRegistryOfflines(false)
			r := <-send
			So(string(r), ShouldStartWith, "one:Ipcams:")

			close(send)
			So(func() { center.onRegistryOfflines(false) }, ShouldNotPanic)
		})
	})
}
