//go:build !windows

package ui

import (
	"runtime"
	"syscall"
	"unsafe"
)

// detectTerminalWidth queries the terminal width via TIOCGWINSZ ioctl on Unix systems.
// Tries stdout first, then stderr (in case stdout is piped).
// Returns 0 if detection fails.
func detectTerminalWidth() int {
	if w := getTerminalWidthFromFd(syscall.Stdout); w > 0 {
		return w
	}
	if w := getTerminalWidthFromFd(syscall.Stderr); w > 0 {
		return w
	}
	return 0
}

func getTerminalWidthFromFd(fd int) int {
	// winsize matches the kernel struct winsize from <sys/ioctl.h>:
	//   unsigned short ws_row, ws_col, ws_xpixel, ws_ypixel
	type winsize struct {
		Row    uint16
		Col    uint16
		Xpixel uint16
		Ypixel uint16
	}

	// TIOCGWINSZ constants per platform
	tiocgwinsz := uintptr(0x5413) // Linux
	if runtime.GOOS == "darwin" {
		tiocgwinsz = 0x40087468 // macOS: _IOR('t', 104, struct winsize)
	}

	var ws winsize
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), tiocgwinsz, uintptr(unsafe.Pointer(&ws)))
	if err != 0 {
		return 0
	}
	return int(ws.Col)
}
