package grouphouse

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Workspace manages the shared file directory for a group house.
type Workspace struct {
	Path string
}

func NewWorkspace(basePath, houseName string) (*Workspace, error) {
	path := filepath.Join(basePath, "houses", houseName)
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("creating workspace: %w", err)
	}
	return &Workspace{Path: path}, nil
}

func (w *Workspace) WriteFile(relativePath string, content string) (string, error) {
	fullPath := filepath.Join(w.Path, relativePath)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("creating directory: %w", err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("writing file: %w", err)
	}
	return fullPath, nil
}

func (w *Workspace) ReadFile(relativePath string) (string, error) {
	fullPath := filepath.Join(w.Path, relativePath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("reading file: %w", err)
	}
	return string(data), nil
}

func (w *Workspace) DeleteFile(relativePath string) error {
	fullPath := filepath.Join(w.Path, relativePath)
	return os.Remove(fullPath)
}

func (w *Workspace) Tree() []string {
	var files []string
	filepath.Walk(w.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(w.Path, path)
		files = append(files, rel)
		return nil
	})
	return files
}

// Run executes a command inside the workspace directory and returns the output.
// This is a simplified version — a production version would use proper sandboxing.
func (w *Workspace) Run(command string) (string, string, int, error) {
	// Basic security: prevent command injection with dangerous patterns
	blocked := []string{"rm -rf", "sudo", "| sh", "`", "$(", "; rm"}
	for _, b := range blocked {
		if strings.Contains(command, b) {
			return "", "blocked: dangerous command pattern detected", 1, nil
		}
	}

	// Use the workspace as working directory
	cmd := NewCmd(command, w.Path)
	return cmd.Run()
}
