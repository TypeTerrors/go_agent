package pkg

import (
	"os"
	"path/filepath"
	"sync"
)

// LockManager manages file locks for concurrent operations.
type LockManager struct{ m sync.Map } // abs path -> *sync.RWMutex

// NewLockManager creates a new LockManager instance.
func NewLockManager() *LockManager {
	return &LockManager{}
}

// Get retrieves or creates a lock for the specified path.
// Parameters:
// - path: the file path for which to get the lock.
func (lm *LockManager) Get(path string) *sync.RWMutex {
	v, _ := lm.m.LoadOrStore(path, &sync.RWMutex{})
	return v.(*sync.RWMutex)
}

// WriteAtomic writes data to a file atomically.
// It ensures the directory exists and uses a temporary file for atomicity.
// Parameters:
// - filename: the name of the file to write.
// - data: the data to write to the file.
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
