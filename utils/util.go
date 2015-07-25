package utils

import (
	"encoding/json"
	"fmt"
)

// tp: response.type
func GenCtrlResMessage(to uint, tp string, m interface{}) ([]byte, error) {
	msg, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	// Response will be unwrapped by server,
	// Then send content to Many
	msg = []byte(fmt.Sprintf(`one:ResponseToMany:%d:{"type":"Response","content":{
		"type":"%s","content":%s
	}}`, to, tp, msg))
	return msg, nil
}

func GenInfoMessage(to uint, msg string) []byte {
	return []byte(fmt.Sprintf(`one:ResponseToMany:%d:{"type":"Info","content":"%s"}`, to, msg))
}

func GenServerCommand(name, content string) []byte {
	return []byte(fmt.Sprintf(`one:ServerCommand:{"name":"%s","content":"%s"}`, name, content))
}
