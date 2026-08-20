//go:build windows

package admin

import (
	"golang.org/x/sys/windows"
)

func diskFree(path string) (free, total int64, err error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0, err
	}
	var freeBytes, totalBytes, totalFree uint64
	if err := windows.GetDiskFreeSpaceEx(p, &freeBytes, &totalBytes, &totalFree); err != nil {
		return 0, 0, err
	}
	return int64(freeBytes), int64(totalBytes), nil
}
