package utils

import "fmt"

func GenServerCommand(name, content string) []byte {
	return []byte(fmt.Sprintf(`one:ServerCommand:{"name":"%s","content":"%s"}`, name, content))
}
