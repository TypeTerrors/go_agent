package pkg

import (
	"os"
	"path/filepath"
	"sync"
)

// LockManager manages per-path locks and atomic writes for FS tools.
// Flow: initialized during Agent.Init(); used by Tooling() and phases.
type LockManager struct{ m sync.Map } // abs path -> *sync.RWMutex

// NewLockManager constructs a LockManager.
// Flow: called by setLockManager() during Init.
func NewLockManager() *LockManager {
	return &LockManager{}
}

// Get returns an RWMutex for a path (stable per absolute path).
// Flow: called in Tooling() before FS ops to coordinate access.
func (lm *LockManager) Get(path string) *sync.RWMutex {
	v, _ := lm.m.LoadOrStore(path, &sync.RWMutex{})
	return v.(*sync.RWMutex)
}

// WriteAtomic persists bytes atomically (dir ensure + rename).
// Flow: used by write_file in Tooling().
func (lm *LockManager) WriteAtomic(filename string, data []byte) error {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	_, werr := f.Write(data)
	serr := f.Sync()
	cerr := f.Close()
	if werr != nil {
		_ = os.Remove(tmp)
		return werr
	}
	if serr != nil {
		_ = os.Remove(tmp)
		return serr
	}
	if cerr != nil {
		_ = os.Remove(tmp)
		return cerr
	}
	return os.Rename(tmp, filename)
}
