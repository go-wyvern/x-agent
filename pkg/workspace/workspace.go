package workspace

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

type Workspace interface {
	Update() error
	Workspace() string
}

type Config struct {
	RepoURL    string
	LocalPath  string
	Branch     string
}

type gitWorkspace struct {
	config Config
	mu     sync.RWMutex
}

func New(config Config) (Workspace, error) {
	if config.RepoURL == "" {
		return nil, fmt.Errorf("repo URL is required")
	}
	if config.LocalPath == "" {
		return nil, fmt.Errorf("local path is required")
	}
	if config.Branch == "" {
		config.Branch = "main"
	}

	ws := &gitWorkspace{
		config: config,
	}

	if err := ws.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize workspace: %w", err)
	}

	return ws, nil
}

func (w *gitWorkspace) initialize() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, err := os.Stat(filepath.Join(w.config.LocalPath, ".git")); err == nil {
		return nil
	}

	if err := os.MkdirAll(w.config.LocalPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	cmd := exec.Command("git", "clone", "-b", w.config.Branch, w.config.RepoURL, w.config.LocalPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to clone repository: %w, output: %s", err, string(output))
	}

	return nil
}

func (w *gitWorkspace) Update() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	cmd := exec.Command("git", "-C", w.config.LocalPath, "pull", "origin", w.config.Branch)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to pull repository: %w, output: %s", err, string(output))
	}

	return nil
}

func (w *gitWorkspace) Workspace() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.config.LocalPath
}
