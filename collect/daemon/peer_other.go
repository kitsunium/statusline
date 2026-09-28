//go:build !linux

package daemon

import "net"

// samePeer relies on the private (0700) instance directory until the SDK
// ships a portable peer-identity primitive (getpeereid, named pipe DACL).
func samePeer(conn net.Conn) bool {
	_, ok := conn.(*net.UnixConn)
	return ok
}
