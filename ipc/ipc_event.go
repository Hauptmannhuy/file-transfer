package ipc

import (
	"syscall"

	"golang.org/x/sys/unix"
)

type eventfd int

func initEventFd() (eventfd, error) {
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

func (fd eventfd) Write(signal int) (int, error) {
	return syscall.Write(int(fd), []byte{byte(signal)})
}
