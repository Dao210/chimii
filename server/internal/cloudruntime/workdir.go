package cloudruntime

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

var (
	ErrUnsafeWorkDir  = errors.New("unsafe cloud runtime work directory")
	ErrWorkDirPrepare = errors.New("cloud runtime work directory preparation failed")
)

// PrepareWorkDir creates a task-private directory from validated UUID path
// components. Cloud execution currently exposes no host tools, but the SDK
// still reads CWD for system context, so even the text-only executor must not
// point at the API server source tree or another tenant's directory.
func PrepareWorkDir(root, workspaceID, taskID string) (string, error) {
	if !filepath.IsAbs(root) {
		return "", fmt.Errorf("%w: root must be absolute", ErrUnsafeWorkDir)
	}
	if _, err := uuid.Parse(workspaceID); err != nil {
		return "", fmt.Errorf("%w: invalid workspace id", ErrUnsafeWorkDir)
	}
	if _, err := uuid.Parse(taskID); err != nil {
		return "", fmt.Errorf("%w: invalid task id", ErrUnsafeWorkDir)
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return "", fmt.Errorf("%w: create root: %v", ErrWorkDirPrepare, err)
	}
	if err := ensurePrivateDirectory(root); err != nil {
		return "", err
	}
	workspaceDir := filepath.Join(root, workspaceID)
	if err := mkdirPrivate(workspaceDir); err != nil {
		return "", err
	}
	taskDir := filepath.Join(workspaceDir, taskID)
	if err := mkdirPrivate(taskDir); err != nil {
		return "", err
	}
	return taskDir, nil
}

func mkdirPrivate(path string) error {
	if err := os.Mkdir(path, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
		return fmt.Errorf("%w: create directory: %v", ErrWorkDirPrepare, err)
	}
	return ensurePrivateDirectory(path)
}

func ensurePrivateDirectory(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("%w: inspect directory: %v", ErrWorkDirPrepare, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("%w: %s is not a real directory", ErrUnsafeWorkDir, path)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		return fmt.Errorf("%w: set private directory permissions: %v", ErrWorkDirPrepare, err)
	}
	return nil
}
