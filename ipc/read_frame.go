// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package ipc

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

// readFrame refuses a length over MaxFrame before allocating it: a peer
// cannot make the other side allocate what it announces.
func readFrame(r io.Reader, v any) error {
	var header [headerSize]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return fmt.Errorf("ipc: read frame header: %w", err)
	}
	size := binary.BigEndian.Uint32(header[:])
	if size > uint32(MaxFrame) {
		return ErrFrameTooLarge
	}
	body := make([]byte, size)
	if _, err := io.ReadFull(r, body); err != nil {
		return fmt.Errorf("ipc: read frame body: %w", err)
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("ipc: decode frame: %w", err)
	}
	return nil
}
