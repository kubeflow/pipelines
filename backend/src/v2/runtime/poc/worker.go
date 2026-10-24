package poc

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func (c *Coordinator) lockInstallation() error {
	if c.coordinatorState == nil {
		c.coordinatorState = &coordinatorState{}
	}
	if err := os.MkdirAll(WorkspaceRoot(), 0o700); err != nil {
		return fmt.Errorf("create runtime installation directory: %w", err)
	}
	file, err := os.OpenFile(filepath.Join(WorkspaceRoot(), ".worker.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("open runtime worker lock: %w", err)
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = file.Close()
		return fmt.Errorf("another worker already manages %q; disable the embedded worker before starting an independent worker: %w", WorkspaceRoot(), err)
	}
	c.workerLock = file
	return nil
}
