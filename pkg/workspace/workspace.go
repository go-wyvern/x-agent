package workspace

import (
	"fmt"
	"io"
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
	RepoURL   string
	LocalPath string
	Branch    string
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

type SessionWorkspaceConfig struct {
	SessionID        string
	SkillsRepoURL    string
	SkillsRepoBranch string
}

func InitializeSessionWorkspace(config SessionWorkspaceConfig) (string, error) {
	if config.SessionID == "" {
		return "", fmt.Errorf("session ID is required")
	}
	if config.SkillsRepoURL == "" {
		return "", fmt.Errorf("skills repo URL is required")
	}
	if config.SkillsRepoBranch == "" {
		config.SkillsRepoBranch = "main"
	}

	workspacePath := filepath.Join("/tmp", fmt.Sprintf("xagent-session-%s", config.SessionID))

	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		return "", fmt.Errorf("failed to create workspace directory: %w", err)
	}

	skillsPath := filepath.Join(workspacePath, ".claude", "skills")
	if err := os.MkdirAll(skillsPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create skills directory: %w", err)
	}

	tempSkillsRepo := filepath.Join("/tmp", fmt.Sprintf("x-skills-clone-%s", config.SessionID))
	defer os.RemoveAll(tempSkillsRepo)

	cloneCmd := exec.Command("git", "clone", "-b", config.SkillsRepoBranch, "--depth", "1", config.SkillsRepoURL, tempSkillsRepo)
	if output, err := cloneCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to clone skills repository: %w, output: %s", err, string(output))
	}

	entries, err := os.ReadDir(tempSkillsRepo)
	if err != nil {
		return "", fmt.Errorf("failed to read skills repository: %w", err)
	}

	for _, entry := range entries {
		if entry.Name() == ".git" {
			continue
		}

		srcPath := filepath.Join(tempSkillsRepo, entry.Name())
		dstPath := filepath.Join(skillsPath, entry.Name())

		if err := copyFile(srcPath, dstPath, entry.IsDir()); err != nil {
			return "", fmt.Errorf("failed to copy %s: %w", entry.Name(), err)
		}
	}

	claudeMDContent := `# Knowledge Base Workspace

This is a knowledge base repository for Claude Code.

This repository stores knowledge bases in markdown format, supporting Claude Code skills specification.

## How to use

1. Each skill is stored in a separate directory under .claude/skills/
2. When you receive a prompt, analyze the topic and determine which skill category it belongs to
3. Read the corresponding skill file to answer questions or provide assistance
4. Follow the guidelines and best practices defined in each skill

## Available Skills

Check the .claude/skills/ directory for all available skills and their documentation.
`

	claudeMDPath := filepath.Join(workspacePath, "CLAUDE.md")
	if err := os.WriteFile(claudeMDPath, []byte(claudeMDContent), 0644); err != nil {
		return "", fmt.Errorf("failed to create CLAUDE.md: %w", err)
	}

	return workspacePath, nil
}

func copyFile(src, dst string, isDir bool) error {
	if isDir {
		return copyDir(src, dst)
	}

	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	return destFile.Sync()
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if err := copyFile(srcPath, dstPath, entry.IsDir()); err != nil {
			return err
		}
	}

	return nil
}
