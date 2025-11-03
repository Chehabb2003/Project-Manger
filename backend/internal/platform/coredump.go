package platform

import "golang.org/x/sys/unix"

func DisableCoreDumps() error {
	var rlim unix.Rlimit
	rlim.Cur = 0
	rlim.Max = 0
	return unix.Setrlimit(unix.RLIMIT_CORE, &rlim)
}
