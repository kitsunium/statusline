// Package transcript reads the end of session transcripts (JSONL).
package transcripts

import (
	"bytes"
	"io"
	"os"
)

// Tail reads the last size bytes of a file, starting at a line boundary.
//
// A transcript grows to megabytes while only its recent lines matter, so
// reading the end is enough and keeps every redraw cheap.
//
// Params:
//   - path: file to read
//   - size: maximum number of bytes to read from the end
//
// Returns:
//   - []byte: the last complete lines
//   - error: any error opening or reading the file
func tail(path string, size int64) ([]byte, error) {
	f, err := os.Open(path)
	// A missing transcript is the caller's to handle
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	// A file that cannot be measured cannot be tailed
	if err != nil {
		return nil, err
	}
	offset := max(info.Size()-size, 0)
	data, err := io.ReadAll(io.NewSectionReader(f, offset, info.Size()-offset))
	// A read cut short yields nothing rather than a torn line
	if err != nil {
		return nil, err
	}
	// Drop the partial first line when reading from the middle of the file
	if offset > 0 {
		// A tail with no newline at all is one partial line: nothing complete
		if idx := bytes.IndexByte(data, '\n'); idx >= 0 {
			data = data[idx+1:]
		} else {
			data = nil
		}
	}
	return data, nil
}
