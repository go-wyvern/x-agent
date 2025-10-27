package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				RepoURL:   "https://github.com/example/repo.git",
				LocalPath: "/tmp/test-workspace",
				Branch:    "main",
			},
			wantErr: false,
		},
		{
			name: "missing repo URL",
			config: Config{
				LocalPath: "/tmp/test-workspace",
				Branch:    "main",
			},
			wantErr: true,
		},
		{
			name: "missing local path",
			config: Config{
				RepoURL: "https://github.com/example/repo.git",
				Branch:  "main",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWorkspace(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "test-workspace-path")
	defer os.RemoveAll(tempDir)

	ws := &gitWorkspace{
		config: Config{
			LocalPath: tempDir,
		},
	}

	if got := ws.Workspace(); got != tempDir {
		t.Errorf("Workspace() = %v, want %v", got, tempDir)
	}
}
