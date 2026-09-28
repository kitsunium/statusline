package ipc

import (
	"crypto/sha256"
	"encoding/binary"
	"strconv"
	"strings"
)

// headerSize is the frame length prefix: 4 bytes, big endian.
const headerSize int = 4

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

// semver is a parsed release version; a development build has dev set.
type semver struct {
	dev   bool
	parts [3]int
	pre   string
}

// parseVersion reads v1.2.3[-pre]; anything without a leading number is a
// development build.
func parseVersion(s string) semver {
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	core, pre, _ := strings.Cut(s, "-")
	fields := strings.Split(core, ".")
	var out semver
	for i := 0; i < len(out.parts) && i < len(fields); i++ {
		n, err := strconv.Atoi(fields[i])
		if err != nil || n < 0 {
			if i == 0 {
				return semver{dev: true}
			}
			break
		}
		out.parts[i] = n
	}
	out.pre = pre
	return out
}
