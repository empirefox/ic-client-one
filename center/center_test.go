package center

import (
	"bytes"
	"testing"
	"time"

	"github.com/empirefox/ic-client-one/config"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	testConfFileForLoad = "../config/config_test_for_load.toml"
	testConfFileForSave = "../config/config_test_for_save.toml"
)

func newFakeCenter() *Center {
	return &Center{
		statusReciever:       make(map[*Connection]bool),
		AddStatusReciever:    make(chan *Connection),
		RemoveStatusReciever: make(chan *Connection),
		ChangeStatus:         make(chan string),
		Quit:                 make(chan bool),
		Conf:                 *config.NewConfigFile(testConfFileForLoad),
		Upgrader:             &fakeUpgrader{},
		Dialer:               &fakeDialer{},
		Conductor:            &fakeConductor{},
	}
}

func TestCenter_onRegistryOfflines(t *testing.T) {
	Convey("onRegistryOfflines", t, func() {
		Convey("should registry and send change", func() {
			center := newFakeCenter()
			center.CtrlConn = newFakeConn(center, "")
			go center.onRegistryOfflines()
			select {
			case result := <-center.CtrlConn.Send:
				So(bytes.HasPrefix(result, []byte("one:Ipcams:")), ShouldBeTrue)
			case <-time.After(time.Second * 2):
				So("Timeout", ShouldEqual, "Not timeout")
			}
			go center.onRegistryOfflines()
			select {
			case result := <-center.CtrlConn.Send:
				So(result, ShouldBeNil)
			case <-time.After(time.Second * 2):
				So("Timeout", ShouldEqual, "Timeout")
			}
		})
	})
}

func TestCenter_OnManageGetIpcam(t *testing.T) {
	Convey("OnManageGetIpcam", t, func() {
		Convey("should return ManagedIpcam", func() {
			center := newFakeCenter()
			center.CtrlConn = newFakeConn(center, "")
			go center.OnManageGetIpcam(&Command{
				From:    21,
				Name:    "OnManageGetIpcam",
				Content: center.Conf.Ipcams[0].Id,
			})
			select {
			case result := <-center.CtrlConn.Send:
				So(bytes.Contains(result, []byte(`"url"`)), ShouldBeTrue)
			case <-time.After(time.Second * 2):
				So("Timeout", ShouldEqual, "Not timeout")
			}
		})
	})
}
