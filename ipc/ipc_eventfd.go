package ipc

import (
	"encoding/binary"
	"syscall"

	"golang.org/x/sys/unix"
)

type eventfd int

func InitEventFd() (eventfd, error) {
	fd, err := unix.Eventfd(0, 0)
	if err != nil {
		return 0, err
	}

	return eventfd(fd), err
}

func (fd eventfd) Read() (int, error) {
	var buffer []byte = make([]byte, 8)
	return syscall.Read(int(fd), buffer)
}

func (fd eventfd) Write(signal uint64) (int, error) {
	buffer := make([]byte, 8)
	binary.NativeEndian.PutUint64(buffer, signal)
	return syscall.Write(int(fd), buffer[:8])
}
