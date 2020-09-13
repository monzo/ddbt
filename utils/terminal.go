// +build linux darwin freebsd netbsd openbsd solaris dragonfly

package utils

import (
	"os"
	"syscall"
	"unsafe"
)

var tty *os.File

func init() {
	var err error
	tty, err = os.Open("/dev/tty")

	// If we can't get the tty, get the Stdout instead as that should be attached to the terminal
	if err != nil {
		tty = os.Stdout
	}
}

type window struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

// Gets the current terminal width and returns an error if we can't get it
func TerminalWidth() (int, error) {
	w := new(window)
	returnCode, _, err := syscall.Syscall(
		syscall.SYS_IOCTL,
		tty.Fd(),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(w)),
	)

	if int(returnCode) == -1 {
		return 0, err
	}

	return int(w.Col), nil
}
