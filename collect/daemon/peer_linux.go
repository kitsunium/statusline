//go:build linux

package daemon

import (
	"net"
	"os"
	"syscall"
)

// samePeer reads the peer's credentials from the kernel (SO_PEERCRED): only
// this user may talk to the daemon, whatever the socket's mode.
func samePeer(conn net.Conn) bool {
	uc, ok := conn.(*net.UnixConn)
	if !ok {
		return false
	}
	raw, err := uc.SyscallConn()
	if err != nil {
		return false
	}
	var cred *syscall.Ucred
	var credErr error
	if err := raw.Control(func(fd uintptr) {
		cred, credErr = syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	}); err != nil || credErr != nil {
		return false
	}
	return int(cred.Uid) == os.Getuid()
}
