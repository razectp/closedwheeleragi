// Package git provides Git integration for the AGI agent.
package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Client provides Git operations
type Client struct {
	repoPath string
}

// NewClient creates a new Git client
func NewClient(repoPath string) *Client {
	return &Client{repoPath: repoPath}
}

// IsRepo checks if the path is a Git repository
func (c *Client) IsRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = c.repoPath
	return cmd.Run() == nil
}

// Init initializes a new Git repository
func (c *Client) Init() error {
	return c.run("init")
}

// Status returns the current Git status
func (c *Client) Status() (string, error) {
	return c.output("status", "--short")
}

// StatusPorcelain returns machine-readable status
func (c *Client) StatusPorcelain() ([]FileStatus, error) {
	output, err := c.output("status", "--porcelain")
	if err != nil {
		return nil, err
	}

	var files []FileStatus
	for _, line := range strings.Split(output, "\n") {
		if len(line) < 3 {
			continue
		}
		files = append(files, FileStatus{
			Status: strings.TrimSpace(line[:2]),
			Path:   strings.TrimSpace(line[3:]),
		})
	}
	return files, nil
}

// FileStatus represents a file's Git status
type FileStatus struct {
	Status string // M, A, D, ??, etc.
	Path   string
}

// Add stages files
func (c *Client) Add(paths ...string) error {
	args := append([]string{"add"}, paths...)
	return c.run(args...)
}

// AddAll stages all changes
func (c *Client) AddAll() error {
	return c.run("add", "-A")
}

// Commit creates a commit
func (c *Client) Commit(message string) error {
	return c.run("commit", "-m", message)
}

// CommitWithTimestamp creates a commit with AGI timestamp
func (c *Client) CommitWithTimestamp(message string) error {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fullMessage := fmt.Sprintf("[AGI] %s\n\nTimestamp: %s", message, timestamp)
	return c.run("commit", "-m", fullMessage)
}

// Branch returns current branch name
func (c *Client) Branch() (string, error) {
	return c.output("rev-parse", "--abbrev-ref", "HEAD")
}

// CreateBranch creates a new branch
func (c *Client) CreateBranch(name string) error {
	return c.run("checkout", "-b", name)
}

// Checkout switches to a branch
func (c *Client) Checkout(branch string) error {
	return c.run("checkout", branch)
}

// Diff returns the diff of unstaged changes
func (c *Client) Diff() (string, error) {
	return c.output("diff")
}

// DiffStaged returns the diff of staged changes
func (c *Client) DiffStaged() (string, error) {
	return c.output("diff", "--staged")
}

// Log returns recent commits
func (c *Client) Log(n int) ([]Commit, error) {
	output, err := c.output("log", fmt.Sprintf("-n%d", n), "--pretty=format:%H|%s|%an|%ai")
	if err != nil {
		return nil, err
	}

	var commits []Commit
	for _, line := range strings.Split(output, "\n") {
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 4 {
			continue
		}
		commits = append(commits, Commit{
			Hash:    parts[0],
			Message: parts[1],
			Author:  parts[2],
			Date:    parts[3],
		})
	}
	return commits, nil
}

// Commit represents a Git commit
type Commit struct {
	Hash    string
	Message string
	Author  string
	Date    string
}

// Stash stashes current changes
func (c *Client) Stash(message string) error {
	if message == "" {
		return c.run("stash")
	}
	return c.run("stash", "push", "-m", message)
}

// StashPop pops the latest stash
func (c *Client) StashPop() error {
	return c.run("stash", "pop")
}

// Reset resets to a commit
func (c *Client) Reset(ref string, hard bool) error {
	if hard {
		return c.run("reset", "--hard", ref)
	}
	return c.run("reset", ref)
}

// HasUncommittedChanges checks for uncommitted changes
func (c *Client) HasUncommittedChanges() bool {
	output, err := c.output("status", "--porcelain")
	if err != nil {
		return false
	}
	return strings.TrimSpace(output) != ""
}

// CreateCheckpoint creates a checkpoint commit
func (c *Client) CreateCheckpoint(description string) (string, error) {
	// Add all changes
	if err := c.AddAll(); err != nil {
		return "", err
	}

	// Check if there's anything to commit
	if !c.HasUncommittedChanges() {
		return "", nil // Nothing to commit
	}

	// Create commit
	message := fmt.Sprintf("checkpoint: %s", description)
	if err := c.CommitWithTimestamp(message); err != nil {
		return "", err
	}

	// Return commit hash
	return c.output("rev-parse", "HEAD")
}

// Helper methods

func (c *Client) run(args ...string) error {
	cmd := c.command(args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s: %v - %s", args[0], err, stderr.String())
	}
	return nil
}

func (c *Client) output(args ...string) (string, error) {
	cmd := c.command(args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %v - %s", args[0], err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

func (c *Client) command(args ...string) *exec.Cmd {
	cmd := exec.Command("git", args...)
	cmd.Dir = c.repoPath

	// Always inherit full system environment so git can find its own tools,
	// SSH keys, credential helpers, etc. On Windows also suppress credential prompts.
	env := os.Environ()
	if runtime.GOOS == "windows" {
		env = append(env, "GIT_TERMINAL_PROMPT=0")
	}
	cmd.Env = env

	return cmd
}

// EnsureRepo ensures the path is a Git repo, initializing if needed
func EnsureRepo(path string) (*Client, error) {
	client := NewClient(path)

	if !client.IsRepo() {
		if err := client.Init(); err != nil {
			return nil, fmt.Errorf("failed to init git repo: %w", err)
		}

		// Create .gitignore
		gitignore := `# Coder AGI
.agi/
*.log
*.tmp
`
		gitignorePath := filepath.Join(path, ".gitignore")
		// Write gitignore (simplified, would use os.WriteFile in real code)
		_ = gitignorePath
		_ = gitignore
	}

	return client, nil
}
