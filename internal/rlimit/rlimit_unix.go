//go:build unix

package rlimit

import "golang.org/x/sys/unix"

func Raise() {
	var lim unix.Rlimit
	if err := unix.Getrlimit(unix.RLIMIT_NOFILE, &lim); err != nil {
		return
	}
	if lim.Cur >= 4096 {
		return
	}
	lim.Cur = 4096
	if lim.Max < lim.Cur {
		lim.Cur = lim.Max
	}
	_ = unix.Setrlimit(unix.RLIMIT_NOFILE, &lim)
}

func Current() uint64 {
	var lim unix.Rlimit
	if err := unix.Getrlimit(unix.RLIMIT_NOFILE, &lim); err != nil {
		return 0
	}
	return uint64(lim.Cur)
}
