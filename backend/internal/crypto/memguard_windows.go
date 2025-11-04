//go:build windows

package crypto

func lockMemory(b []byte) error   { return nil }
func unlockMemory(b []byte) error { return nil }
