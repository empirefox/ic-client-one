package wsio

import (
	"bytes"
	"encoding/json"
	"fmt"
)

/////////////////////////////////////
// IN
/////////////////////////////////////

// From One: "ManageGetIpcam", "ManageSetIpcam", "ManageReconnectIpcam"
type FromServerCommand struct {
	From    uint            `json:"from"`
	Name    string          `json:"name"`
	Content json.RawMessage `json:"content"`
}

func (c *FromServerCommand) Value() []byte {
	return bytes.Trim(c.Content, `"`)
}

func (c *FromServerCommand) Signaling() (*SubSignalCommand, error) {
	sub := &SubSignalCommand{}
	if err := json.Unmarshal(c.Content, sub); err != nil {
		return nil, err
	}
	return sub, nil
}

func (c *FromServerCommand) String() string {
	return fmt.Sprintf(`{
	from:%d,
	name:"%s",
	content:%s
}`, c.From, c.Name, c.Content)
}

// Camera => Id
type SubSignalCommand struct {
	Camera   string `json:"camera,omitempty"`
	Reciever string `json:"reciever,omitempty"`
}

/////////////////////////////////////
// OUT
/////////////////////////////////////
