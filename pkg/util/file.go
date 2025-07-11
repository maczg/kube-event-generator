package util

import (
	"fmt"
	"os"
)

func CreateDir(dirName string) error {
	if _, err := os.Stat(dirName); os.IsNotExist(err) {
		if err := os.MkdirAll(dirName, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dirName, err)
		}
	}

	return nil
}
