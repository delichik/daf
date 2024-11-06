package utils

import (
	"runtime"
	"strings"
)

func IsCalledByInit(skip int) bool {
	pcs := make([]uintptr, 16)
	n := runtime.Callers(skip+1, pcs)
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		parts := strings.Split(frame.Function, ".")
		if len(parts) < 2 {
			continue
		}
		if parts[len(parts)-2] == "init" {
			return true
		}

		if !more {
			break
		}
	}

	return false
}

func IsCalledByMainDirect(skip int) bool {
	pc, _, _, ok := runtime.Caller(skip + 1)
	if !ok {
		return false
	}

	frame := runtime.FuncForPC(pc)
	parts := strings.Split(frame.Name(), ".")
	if len(parts) < 2 {
		return false
	}

	if parts[len(parts)-2] != "main" {
		return false
	}

	return true
}
