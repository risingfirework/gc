//go:build linux

package handler

import "syscall"

func diskUsage(path string) (uint64, uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, err
	}
	return stat.Blocks * uint64(stat.Bsize), stat.Bavail * uint64(stat.Bsize), nil
}

func systemMemory() (uint64, uint64, error) {
	var info syscall.Sysinfo_t
	if err := syscall.Sysinfo(&info); err != nil {
		return 0, 0, err
	}
	total := info.Totalram * uint64(info.Unit)
	free := info.Freeram * uint64(info.Unit)
	// Buffers/cache juga dihitung sebagai available di kernel modern, tapi
	// Freeram cukup untuk indikator sederhana.
	return total, free, nil
}