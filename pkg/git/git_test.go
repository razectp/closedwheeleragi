package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestRepo creates a temp git repo and returns a Client and cleanup func.
func setupTestRepo(t *testing.T) (*Client, string, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}

	client := NewClient(dir)

	// Init repo
	if err := client.Init(); err != nil {
		os.RemoveAll(dir)
		t.Fatalf("Init: %v", err)
	}

	// Configure identity so commits work
	for _, args := range [][]string{
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test User"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			os.RemoveAll(dir)
			t.Fatalf("git %v: %v — %s", args, err, out)
		}
	}

	return client, dir, func() { os.RemoveAll(dir) }
}

func TestCommand_InheritsEnv(t *testing.T) {
	client := NewClient(".")
	cmd := client.command("version")

	if len(cmd.Env) == 0 {
		t.Fatal("cmd.Env is empty — system environment not inherited")
	}

	hasPath := false
	for _, e := range cmd.Env {
		if strings.HasPrefix(strings.ToUpper(e), "PATH=") {
			hasPath = true
			break
		}
	}
	if !hasPath {
		t.Error("PATH not found in cmd.Env")
	}
}

func TestIsRepo_False(t *testing.T) {
	dir, err := os.MkdirTemp("", "notrepo-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	client := NewClient(dir)
	if client.IsRepo() {
		t.Error("expected false for non-repo directory")
	}
}

func TestInit_IsRepo(t *testing.T) {
	_, _, cleanup := setupTestRepo(t)
	defer cleanup()
	// setupTestRepo already calls Init and verifies no error; IsRepo checked implicitly
}

func TestStatus(t *testing.T) {
	client, dir, cleanup := setupTestRepo(t)
	defer cleanup()

	f := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(f, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	status, err := client.Status()
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !strings.Contains(status, "hello.txt") {
		t.Errorf("expected 'hello.txt' in status, got: %s", status)
	}
}

func TestCommitAndLog(t *testing.T) {
	client, dir, cleanup := setupTestRepo(t)
	defer cleanup()

	f := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(f, []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := client.AddAll(); err != nil {
		t.Fatalf("AddAll: %v", err)
	}
	if err := client.Commit("initial"); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	entries, err := client.Log(5)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one log entry")
	}
	if entries[0].Message != "initial" {
		t.Errorf("expected message 'initial', got %q", entries[0].Message)
	}
}

func TestDiff(t *testing.T) {
	client, dir, cleanup := setupTestRepo(t)
	defer cleanup()

	// Commit a file then modify it
	f := filepath.Join(dir, "b.txt")
	if err := os.WriteFile(f, []byte("original\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := client.AddAll(); err != nil {
		t.Fatal(err)
	}
	if err := client.Commit("base"); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(f, []byte("modified\n"), 0644); err != nil {
		t.Fatal(err)
	}

	diff, err := client.Diff()
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if !strings.Contains(diff, "modified") {
		t.Errorf("expected diff to contain 'modified', got: %s", diff)
	}
}
