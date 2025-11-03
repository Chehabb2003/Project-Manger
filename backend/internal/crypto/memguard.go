//go:build linux || darwin

package crypto
import "golang.org/x/sys/unix"

func lockMemory(b []byte) error   { return unix.Mlock(b) }
func unlockMemory(b []byte) error { return unix.Munlock(b) }
