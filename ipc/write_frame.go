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

// writeFrame encodes first so that a value too large never leaves half a
// frame on the wire.
func writeFrame(w io.Writer, v any) error {
	body, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("ipc: encode frame: %w", err)
	}
	if len(body) > MaxFrame {
		return ErrFrameTooLarge
	}
	buf := make([]byte, headerSize+len(body))
	binary.BigEndian.PutUint32(buf, uint32(len(body)))
	copy(buf[headerSize:], body)
	if _, err := w.Write(buf); err != nil {
		return fmt.Errorf("ipc: write frame: %w", err)
	}
	return nil
}
