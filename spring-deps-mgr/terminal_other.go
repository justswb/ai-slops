//go:build !windows

package main

func enableVirtualTerminal() (func(), error) { return func() {}, nil }
