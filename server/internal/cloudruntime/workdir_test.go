package cloudruntime

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestPrepareWorkDirIsTenantAndTaskScoped(t *testing.T) {
	root := filepath.Join(t.TempDir(), "cloud")
	got, err := PrepareWorkDir(root, "00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002")
	if got != want {
		t.Fatalf("workdir = %q, want %q", got, want)
	}
	for _, path := range []string{root, filepath.Dir(got), got} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if gotPerm := info.Mode().Perm(); gotPerm != 0o700 {
			t.Fatalf("permissions for %s = %o, want 700", path, gotPerm)
		}
	}
}

func TestPrepareWorkDirRejectsTraversalAndRelativeRoot(t *testing.T) {
	for _, tc := range []struct{ root, workspace, task string }{
		{"relative", "00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002"},
		{t.TempDir(), "../other", "00000000-0000-0000-0000-000000000002"},
		{t.TempDir(), "00000000-0000-0000-0000-000000000001", "/tmp/other"},
	} {
		if _, err := PrepareWorkDir(tc.root, tc.workspace, tc.task); !errors.Is(err, ErrUnsafeWorkDir) {
			t.Fatalf("PrepareWorkDir(%q, %q, %q) error = %v", tc.root, tc.workspace, tc.task, err)
		}
	}
}

func TestPrepareWorkDirRejectsSymlinkedTenantDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires additional privileges on Windows")
	}
	root := t.TempDir()
	workspace := "00000000-0000-0000-0000-000000000001"
	if err := os.Symlink(t.TempDir(), filepath.Join(root, workspace)); err != nil {
		t.Fatal(err)
	}
	_, err := PrepareWorkDir(root, workspace, "00000000-0000-0000-0000-000000000002")
	if !errors.Is(err, ErrUnsafeWorkDir) {
		t.Fatalf("error = %v", err)
	}
}
