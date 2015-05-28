package config

import (
	"testing"
	"time"

	"github.com/dchest/uniuri"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	testConfFileForLoad = "config_test_for_load.toml"
	testConfFileForSave = "config_test_for_save.toml"
)

func getDefaultTestConf() Config {
	return Config{
		SecretAddress: "a-128",
		Secure:        false,
		Server:        "127.0.0.1:9999",
		PingPeriod:    45 * time.Second,
		Ipcams: []Ipcam{
			{
				Id:   uniuri.NewLen(16),
				Name: "客厅",
				Url:  "rtsp://127.0.0.1:1235/test1.sdp",
			},
		},
	}
}

func TestConfig_Load(t *testing.T) {
	Convey("Load", t, func() {
		Convey("should load from .toml", func() {
			c := getDefaultTestConf()
			c.file = testConfFileForLoad
			result := newConfig(testConfFileForLoad)
			So(result.Ipcams[0].Id, ShouldNotBeBlank)
			result.Ipcams[0].Id = c.Ipcams[0].Id
			So(result, ShouldResemble, &c)
		})
	})
}

func TestConfig_Save(t *testing.T) {
	Convey("Save", t, func() {
		Convey("should save to .toml", func() {
			o := getDefaultTestConf()
			o.file = testConfFileForSave
			err := o.Save()
			So(err, ShouldBeNil)

			result := newConfig(testConfFileForSave)
			So(result.Ipcams[0].Id, ShouldNotEqual, o.Ipcams[0].Id)
			result.Ipcams[0].Id = o.Ipcams[0].Id
			So(result, ShouldResemble, &o)
		})
	})
}

func TestConfig_SaveIpcam(t *testing.T) {
	Convey("SaveIpcam", t, func() {
		Convey("should save new ipcam to .toml", func() {
			o := getDefaultTestConf()
			o.file = testConfFileForSave
			err := o.Save()
			So(err, ShouldBeNil)

			newIpcam := Ipcam{
				Name: "111",
				Url:  "rtsp://127.0.0.1:1235/test11.sdp",
			}
			o.SaveIpcam(newIpcam)

			result := newConfig(testConfFileForSave)
			index := -1
			for i, ipcam := range result.Ipcams {
				if ipcam.Name == "111" {
					index = i
					break
				}
			}
			So(index, ShouldNotEqual, -1)
			result.Ipcams[index].Id = ""
			So(result.Ipcams[index], ShouldResemble, newIpcam)
		})
	})
}
