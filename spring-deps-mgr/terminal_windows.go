package main

import (
	"golang.org/x/sys/windows"
	"os"
)

func enableVirtualTerminal() (func(), error) {
	h := windows.Handle(os.Stderr.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(h, &mode); err != nil {
		return nil, err
	}
	if err := windows.SetConsoleMode(h, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING); err != nil {
		return nil, err
	}
	return func() { windows.SetConsoleMode(h, mode) }, nil
}
