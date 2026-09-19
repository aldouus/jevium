//go:build !unix

package rlimit

func Raise() {}

func Current() uint64 { return 0 }
