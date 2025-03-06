package utils

import (
	"errors"
	"fmt"
)

func HandlePanic() {
	func() {
		// recover from panic if one occurred. Set err to nil otherwise.
		if err := recover(); err != nil {
			msg := fmt.Sprintf("panic occurred: %v", err)
			err = errors.New(msg)
		}
	}()
}
