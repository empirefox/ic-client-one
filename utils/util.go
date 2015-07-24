package utils

import (
	"encoding/json"
	"fmt"
)

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
	msg = append([]byte(fmt.Sprintf(`one:ResponseToMany:%d:{"type":"Response","content":`, to, tp)), msg...)
	return append(msg, '}'), nil
}

func GenInfoMessage(to uint, msg string) []byte {
	return []byte(fmt.Sprintf(`one:ResponseToMany:%d:{"type":"Info","content":"%s"}`, to, msg))
}

func GenServerCommand(name, content string) []byte {
	return []byte(fmt.Sprintf(`one:ServerCommand:{"name":"%s","content":"%s"}`, name, content))
}
