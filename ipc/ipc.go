package ipc

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
)

// headerSize is the frame length prefix: 4 bytes, big endian.
const headerSize int = 4

// hash digests every field length-prefixed, so that moving a byte from one
// field to the next changes the digest (a separator byte could itself
// appear in a field).
func (k Key) hash() string {
	sum := digest(k.SessionID, k.TranscriptPath, k.SessionDir, k.TaskListID)
	return hex.EncodeToString(sum[:12])
}

// digest hashes length-prefixed fields.
func digest(fields ...string) [sha256.Size]byte {
	h := sha256.New()
	var n [8]byte
	for _, f := range fields {
		binary.BigEndian.PutUint64(n[:], uint64(len(f)))
		h.Write(n[:])
		h.Write([]byte(f))
	}
	var out [sha256.Size]byte
	copy(out[:], h.Sum(nil))
	return out
}

// cachePath names the key's snapshot file in the instance's cache.
func (i Instance) cachePath(key Key) string {
	return filepath.Join(i.Cache, key.hash()+".json")
}

// locate puts the instance under a private per-user directory; the
// instance's own directory is named after the digest of what identifies
// it, so two binaries at two paths, or two configuration directories,
// never share a daemon.
func locate(in LocateInput) Instance {
	sum := digest(in.ConfigDir, in.Executable)
	dir := filepath.Join(in.RuntimeDir, "statusline-"+strconv.Itoa(in.UID), hex.EncodeToString(sum[:8]))
	return Instance{
		Dir:       dir,
		Socket:    filepath.Join(dir, "daemon.sock"),
		Lock:      filepath.Join(dir, "daemon.lock"),
		Cache:     filepath.Join(dir, "cache"),
		State:     filepath.Join(dir, "state"),
		Log:       filepath.Join(dir, "daemon.log"),
		PID:       filepath.Join(dir, "daemon.pid"),
		Heartbeat: filepath.Join(dir, "heartbeat"),
	}
}

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

// version is a parsed release version; a development build has dev set.
type version struct {
	dev   bool
	parts [3]int
	pre   string
}

// parseVersion reads v1.2.3[-pre]; anything without a leading number is a
// development build.
func parseVersion(s string) version {
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	core, pre, _ := strings.Cut(s, "-")
	fields := strings.Split(core, ".")
	var out version
	for i := 0; i < len(out.parts) && i < len(fields); i++ {
		n, err := strconv.Atoi(fields[i])
		if err != nil || n < 0 {
			if i == 0 {
				return version{dev: true}
			}
			break
		}
		out.parts[i] = n
	}
	out.pre = pre
	return out
}

// compareVersions: dev < any release; numbers first; a release without a
// pre-release suffix is newer than one with; suffixes compare as strings.
func compareVersions(a, b string) int {
	va, vb := parseVersion(a), parseVersion(b)
	switch {
	case va.dev && vb.dev:
		return 0
	case va.dev:
		return -1
	case vb.dev:
		return 1
	}
	for i := range va.parts {
		if va.parts[i] != vb.parts[i] {
			if va.parts[i] < vb.parts[i] {
				return -1
			}
			return 1
		}
	}
	switch {
	case va.pre == vb.pre:
		return 0
	case va.pre == "":
		return 1
	case vb.pre == "":
		return -1
	case va.pre < vb.pre:
		return -1
	default:
		return 1
	}
}

// compatible accepts any minor revision of the same major.
func compatible(protocol string) bool {
	name, major, ok := strings.Cut(protocol, "/v")
	if !ok || name != "statusline.ipc" {
		return false
	}
	head, _, _ := strings.Cut(major, ".")
	return head == "1"
}
