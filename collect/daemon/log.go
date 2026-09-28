package daemon

import (
	"fmt"
	"os"
	"sync"
	"time"
)

const (
	// maxLog bounds the log; past it the file is rotated once.
	maxLog int64 = 256 << 10
	// logPerm keeps the log to its owner.
	logPerm os.FileMode = 0o600
)

// boundedLog appends timestamped lines and keeps at most two files of
// maxLog: the current one and <log>.1.
type boundedLog struct {
	mu   sync.Mutex
	path string
	file *os.File
	size int64
}

// openLog never fails: without a log the daemon still serves.
func openLog(path string) *boundedLog {
	l := &boundedLog{path: path}
	l.open()
	return l
}

func (l *boundedLog) open() {
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, logPerm)
	if err != nil {
		return
	}
	l.file = f
	if info, err := f.Stat(); err == nil {
		l.size = info.Size()
	}
}

func (l *boundedLog) printf(format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file == nil {
		return
	}
	line := time.Now().UTC().Format(time.RFC3339) + " " + fmt.Sprintf(format, args...) + "\n"
	if l.size+int64(len(line)) > maxLog {
		_ = l.file.Close()
		_ = os.Rename(l.path, l.path+".1")
		l.file, l.size = nil, 0
		l.open()
		if l.file == nil {
			return
		}
	}
	n, _ := l.file.WriteString(line)
	l.size += int64(n)
}

func (l *boundedLog) close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		_ = l.file.Close()
		l.file = nil
	}
}
