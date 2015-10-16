package wsio

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func (c *FromServerCommand) ToManyObj(obj interface{}) ([]byte, error) {
	msg, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	// Response will be unwrapped by server,
	// Then send content to Many
	return c.ToManyJSON(msg), nil
}

// response the same type to many
func (c *FromServerCommand) ToManyJSON(j []byte) []byte {
	return []byte(fmt.Sprintf(`one:ResponseToMany:%d:{"type":"Response",
		"to":"%s","content":%s
	}`, c.From, c.Name, j))
}

func (c *FromServerCommand) ToManyInfo(msg string) []byte {
	return []byte(fmt.Sprintf(`one:ResponseToMany:%d:{"type":"Info","content":"%s"}`, c.From, msg))
}

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
