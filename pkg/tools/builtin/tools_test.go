package builtin

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"ClosedWheeler/pkg/security"
)

// testRoot creates a temp directory as a project root and returns cleanup func.
func testRoot(t *testing.T) (string, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "tools-test-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	return dir, func() { os.RemoveAll(dir) }
}

// ----- read_file -----

func TestReadFile_Success(t *testing.T) {
	root, cleanup := testRoot(t)
	defer cleanup()

	content := "hello world"
	f := filepath.Join(root, "test.txt")
	if err := os.WriteFile(f, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	auditor := security.NewAuditor(root)
	tool := ReadFileTool(root, auditor)
	result, err := tool.Handler(map[string]any{"path": "test.txt"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if !strings.Contains(result.Output, content) {
		t.Errorf("expected output to contain %q, got: %s", content, result.Output)
	}
}

func TestReadFile_Missing(t *testing.T) {
	root, cleanup := testRoot(t)
	defer cleanup()

	auditor := security.NewAuditor(root)
	tool := ReadFileTool(root, auditor)
	result, _ := tool.Handler(map[string]any{"path": "nonexistent.txt"})
	if result.Success {
		t.Error("expected failure for missing file")
	}
}

func TestReadFile_SecurityViolation(t *testing.T) {
	root, cleanup := testRoot(t)
	defer cleanup()

	auditor := security.NewAuditor(root)
	tool := ReadFileTool(root, auditor)
	result, _ := tool.Handler(map[string]any{"path": "../../etc/passwd"})
	if result.Success {
		t.Error("expected security violation for path traversal")
	}
}

// ----- write_file -----

func TestWriteFile_Success(t *testing.T) {
	root, cleanup := testRoot(t)
	defer cleanup()

	auditor := security.NewAuditor(root)
	tool := WriteFileTool(root, auditor)
	result, err := tool.Handler(map[string]any{
		"path":    "output.txt",
		"content": "written content",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Error)
	}

	data, err := os.ReadFile(filepath.Join(root, "output.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "written content" {
		t.Errorf("unexpected file content: %q", data)
	}
}

func TestWriteFile_CreatesSubdirectory(t *testing.T) {
	root, cleanup := testRoot(t)
	defer cleanup()

	auditor := security.NewAuditor(root)
	tool := WriteFileTool(root, auditor)
	result, _ := tool.Handler(map[string]any{
		"path":    "subdir/nested.txt",
		"content": "nested",
	})
	if !result.Success {
		t.Fatalf("expected success creating nested file, got: %s", result.Error)
	}
	if _, err := os.Stat(filepath.Join(root, "subdir", "nested.txt")); err != nil {
		t.Errorf("nested file not created: %v", err)
	}
}

// ----- list_files -----

func TestListFiles_Success(t *testing.T) {
	root, cleanup := testRoot(t)
	defer cleanup()

	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "b.go"), []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	auditor := security.NewAuditor(root)
	tool := ListFilesTool(root, auditor)
	result, _ := tool.Handler(map[string]any{"path": "."})
	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Error)
	}
	if !strings.Contains(result.Output, "a.go") || !strings.Contains(result.Output, "b.go") {
		t.Errorf("expected a.go and b.go in output: %s", result.Output)
	}
}

// ----- exec_command -----

func TestExecCommand_SimpleCommand(t *testing.T) {
	root, cleanup := testRoot(t)
	defer cleanup()

	auditor := security.NewAuditor(root)
	tool := ExecCommandTool(root, 10e9, auditor)

	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "echo hello"
	} else {
		cmd = "echo hello"
	}

	result, err := tool.Handler(map[string]any{"command": cmd})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Error)
	}
	if !strings.Contains(result.Output, "hello") {
		t.Errorf("expected 'hello' in output, got: %s", result.Output)
	}
}

func TestExecCommand_EnvHasPath(t *testing.T) {
	// Verify that commands requiring PATH work (e.g. 'go version')
	root, cleanup := testRoot(t)
	defer cleanup()

	auditor := security.NewAuditor(root)
	tool := ExecCommandTool(root, 30e9, auditor)

	result, err := tool.Handler(map[string]any{"command": "go version"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("go version failed (PATH issue?): %s", result.Error)
	}
	if !strings.Contains(result.Output, "go version") {
		t.Errorf("unexpected output: %s", result.Output)
	}
}

func TestExecCommand_SecurityBlock(t *testing.T) {
	root, cleanup := testRoot(t)
	defer cleanup()

	auditor := security.NewAuditor(root)
	tool := ExecCommandTool(root, 10e9, auditor)

	result, _ := tool.Handler(map[string]any{"command": "rm -rf /"})
	if result.Success {
		t.Error("expected security block for dangerous command")
	}
}

// ----- search_code -----

func TestSearchCode_Found(t *testing.T) {
	root, cleanup := testRoot(t)
	defer cleanup()

	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	auditor := security.NewAuditor(root)
	tool := SearchCodeTool(root, auditor)
	result, _ := tool.Handler(map[string]any{"query": "func main"})
	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Error)
	}
	if !strings.Contains(result.Output, "main.go") {
		t.Errorf("expected 'main.go' in output, got: %s", result.Output)
	}
}

func TestSearchCode_NotFound(t *testing.T) {
	root, cleanup := testRoot(t)
	defer cleanup()

	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	auditor := security.NewAuditor(root)
	tool := SearchCodeTool(root, auditor)
	result, _ := tool.Handler(map[string]any{"query": "nonexistentXYZ123"})
	if !result.Success {
		t.Fatalf("expected success (no results != error), got: %s", result.Error)
	}
	if strings.Contains(result.Output, "main.go") {
		t.Errorf("did not expect main.go when query has no matches")
	}
}

// ----- manage_tasks -----

func TestManageTasks_AddAndList(t *testing.T) {
	root, cleanup := testRoot(t)
	defer cleanup()

	// manage_tasks writes to <root>/workplace/task.md
	if err := os.MkdirAll(filepath.Join(root, "workplace"), 0755); err != nil {
		t.Fatal(err)
	}

	auditor := security.NewAuditor(root)
	tool := TaskManagerTool(root, auditor)

	// Add a task — parameter is "task", not "description"
	result, err := tool.Handler(map[string]any{
		"action": "add",
		"task":   "Do something important",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("add task failed: %s", result.Error)
	}

	// List tasks
	result, _ = tool.Handler(map[string]any{"action": "list"})
	if !result.Success {
		t.Fatalf("list tasks failed: %s", result.Error)
	}
	if !strings.Contains(result.Output, "Do something important") {
		t.Errorf("expected task in list, got: %s", result.Output)
	}
}

// ----- get_system_info -----

func TestGetSystemInfo(t *testing.T) {
	tool := GetSystemInfoTool()
	result, err := tool.Handler(map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Error)
	}
	// Should include OS info
	if !strings.Contains(result.Output, runtime.GOOS) {
		t.Errorf("expected OS %q in output, got: %s", runtime.GOOS, result.Output)
	}
}
