package cloudruntime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCloudExecutionPackageDoesNotDependOnCLIRuntime(t *testing.T) {
	t.Parallel()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		content, err := os.ReadFile(filepath.Clean(entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		source := string(content)
		for _, forbidden := range []string{
			"server/pkg/agent",
			"server/internal/daemon",
			"pkg/agent.New(",
			"exec.Command(",
		} {
			if strings.Contains(source, forbidden) {
				t.Errorf("%s contains forbidden CLI/host execution dependency %q", entry.Name(), forbidden)
			}
		}
	}
}
