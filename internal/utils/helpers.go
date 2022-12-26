package utils

import (
	"fmt"
	"runtime"
)

func ErrorHelper(err error) error {
	if err == nil {
		return err
	}
	_, filename, line, _ := runtime.Caller(1)
	return fmt.Errorf("[%s:%d] %w", filename, line, err)
}
