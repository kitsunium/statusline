package statefile

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/kitsunium/statusline/collect/state"
	"github.com/kitsunium/statusline/ipc"
	"github.com/kitsunium/statusline/snapshot"
)

const (
	// dirPerm keeps every state directory private to its owner.
	dirPerm os.FileMode = 0o700
	// filePerm keeps every state file private to its owner.
	filePerm os.FileMode = 0o600
	// networkFile holds the network bookkeeping.
	networkFile string = "network.json"
	// updateFile holds the update bookkeeping.
	updateFile string = "update.json"
)

func loadNetwork(instance ipc.Instance) (state.Network, error) {
	var n state.Network
	return n, readJSON(filepath.Join(instance.State, networkFile), &n)
}

func saveNetwork(instance ipc.Instance, n state.Network) error {
	return writeJSON(filepath.Join(instance.State, networkFile), n)
}

func saveSnapshot(instance ipc.Instance, key ipc.Key, snap snapshot.Snapshot) error {
	return writeJSON(instance.CachePath(key), snap)
}

func loadUpdate(instance ipc.Instance) (state.Update, error) {
	var u state.Update
	return u, readJSON(filepath.Join(instance.State, updateFile), &u)
}

func saveUpdate(instance ipc.Instance, u state.Update) error {
	return writeJSON(filepath.Join(instance.State, updateFile), u)
}

// readJSON leaves dst untouched when the file does not exist yet.
func readJSON(path string, dst any) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}

// writeJSON goes through a temporary file so that a concurrent reader (a
// client rendering from the cache) never sees a half-written payload.
func writeJSON(path string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(name)
		return err
	}
	if err := os.Chmod(name, filePerm); err != nil {
		_ = os.Remove(name)
		return err
	}
	return os.Rename(name, path)
}
