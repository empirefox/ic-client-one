package wsio

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/golang/glog"
)

/////////////////////////////////////
// IN
/////////////////////////////////////

// From Server: "ManageGetIpcam", "ManageSetIpcam", "ManageReconnectIpcam"
type FromServerCommand struct {
	From    uint            `json:"from"`
	Name    string          `json:"name"`
	Content json.RawMessage `json:"content"`
}

func (c *FromServerCommand) Value() []byte {
	return bytes.Trim(c.Content, `"`)
}

func (c *FromServerCommand) ToManyObj(k []byte, obj interface{}) []byte {
	msg, err := json.Marshal(obj)
	if err != nil {
		glog.Errorln(err)
		return c.ToManyInfo(err.Error())
	}
	// Response will be unwrapped by server,
	// Then send content to Many
	return c.ToManyJSON(k, msg)
}

// response the same type to many
func (c *FromServerCommand) ToManyJSON(k []byte, j []byte) []byte {
	if c == nil {
		return []byte(fmt.Sprintf(`one:T2M:%s:0:%s`, k, j))
	}
	return []byte(fmt.Sprintf(`one:T2M:%s:%d:%s`, k, c.From, j))
}

var infoKey = []byte("Info")

func (c *FromServerCommand) ToManyInfo(msg string) []byte {
	return c.ToManyJSON(infoKey, []byte(msg))
}

var BcCmd = new(FromServerCommand)

func BcObj(k []byte, obj interface{}) []byte { return BcCmd.ToManyObj(k, obj) }
func BcJSON(k []byte, j []byte) []byte       { return BcCmd.ToManyJSON(k, j) }

func (c *FromServerCommand) String() string {
	return fmt.Sprintf(`{
	from:%d,
	name:"%s",
	content:%s
}`, c.From, c.Name, c.Content)
}

/////////////////////////////////////
// OUT
/////////////////////////////////////
