// utils/base64.go
package utils

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func SaveBase64Image(b64, folder string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil { return "", err }

	if err := os.MkdirAll(folder, 0755); err != nil {
		return "", err
	}
	filename := fmt.Sprintf("%d.png", time.Now().UnixNano())
	path := filepath.Join(folder, filename)

	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", err
	}
	return path, nil
}
