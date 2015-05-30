package utils

import (
	"encoding/json"
	"fmt"

	. "github.com/empirefox/ic-client-one/config"
)

type ManageIpcam struct {
	Id     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Url    string `json:"url,omitempty"`
	Off    bool   `json:"off,omitempty"`
	Online bool   `json:"online,omitempty"`
}

func (ipcam *ManageIpcam) Get() Ipcam {
	return Ipcam{ipcam.Id, ipcam.Name, ipcam.Url, ipcam.Off, ipcam.Online}
}

func GetManaged(ipcam *Ipcam) *ManageIpcam {
	return &ManageIpcam{ipcam.Id, ipcam.Name, ipcam.Url, ipcam.Off, ipcam.Online}
}

// tp: response.type
func GenCtrlResMessage(to uint, tp string, m interface{}) ([]byte, error) {
	// Parsed by OnResponse
	msg, err := json.Marshal(map[string]interface{}{
		"type":    tp,
		"content": m,
	})
	if err != nil {
		return nil, err
	}
	msg = append([]byte(fmt.Sprintf(`one:ResponseToMany:%s:{"type":"Response","content":`, to, tp)), msg...)
	return append(msg, '}'), nil
}

func GenInfoMessage(to uint, msg string) []byte {
	return []byte(fmt.Sprintf(`one:ResponseToMany:%s:{"type":"Info","content","%s"}`, to, msg))
}
