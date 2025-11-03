//go:build windows

package crypto

func lockMemory(b []byte) error   { return nil } // no-op on Windows
func unlockMemory(b []byte) error { return nil } // no-op on Windows
