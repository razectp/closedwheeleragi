package brain

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBrain_Initialize(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	brain := NewBrain(tmpDir)

	err := brain.Initialize()
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Check file exists (now directly in tmpDir)
	brainPath := filepath.Join(tmpDir, "brain.md")
	if _, err := os.Stat(brainPath); os.IsNotExist(err) {
		t.Fatal("brain.md was not created")
	}

	// Read content
	content, err := os.ReadFile(brainPath)
	if err != nil {
		t.Fatalf("Failed to read brain.md: %v", err)
	}

	// Verify sections
	contentStr := string(content)
	if !strings.Contains(contentStr, "## Errors and Solutions") {
		t.Error("Missing 'Errors and Solutions' section")
	}
	if !strings.Contains(contentStr, "## Code Patterns") {
		t.Error("Missing 'Code Patterns' section")
	}
	if !strings.Contains(contentStr, "## Architectural Decisions") {
		t.Error("Missing 'Architectural Decisions' section")
	}
	if !strings.Contains(contentStr, "## Insights") {
		t.Error("Missing 'Insights' section")
	}
}

func TestBrain_AddError(t *testing.T) {
	tmpDir := t.TempDir()
	brain := NewBrain(tmpDir)

	if err := brain.Initialize(); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	err := brain.AddError(
		"Test Error",
		"This is a test error",
		"This is the solution",
		[]string{"test", "error"},
	)

	if err != nil {
		t.Fatalf("AddError failed: %v", err)
	}

	// Read and verify
	content, _ := brain.Read()
	if !strings.Contains(content, "Test Error") {
		t.Error("Error title not found in brain")
	}
	if !strings.Contains(content, "This is a test error") {
		t.Error("Error description not found")
	}
	if !strings.Contains(content, "This is the solution") {
		t.Error("Solution not found")
	}
	if !strings.Contains(content, "`test`") {
		t.Error("Tag not found")
	}
}

func TestBrain_AddPattern(t *testing.T) {
	tmpDir := t.TempDir()
	brain := NewBrain(tmpDir)

	if err := brain.Initialize(); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	err := brain.AddPattern(
		"Test Pattern",
		"Always use mutex for shared state",
		[]string{"concurrency"},
	)

	if err != nil {
		t.Fatalf("AddPattern failed: %v", err)
	}

	content, _ := brain.Read()
	if !strings.Contains(content, "Test Pattern") {
		t.Error("Pattern not found in brain")
	}
}

func TestBrain_AddDecision(t *testing.T) {
	tmpDir := t.TempDir()
	brain := NewBrain(tmpDir)

	if err := brain.Initialize(); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	err := brain.AddDecision(
		"Use PostgreSQL",
		"Migrate from SQLite to PostgreSQL",
		"Better scalability and concurrency",
		[]string{"database"},
	)

	if err != nil {
		t.Fatalf("AddDecision failed: %v", err)
	}

	content, _ := brain.Read()
	if !strings.Contains(content, "Use PostgreSQL") {
		t.Error("Decision not found in brain")
	}
	if !strings.Contains(content, "Better scalability") {
		t.Error("Rationale not found")
	}
}

func TestBrain_Search(t *testing.T) {
	tmpDir := t.TempDir()
	brain := NewBrain(tmpDir)

	if err := brain.Initialize(); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Add multiple entries
	if err := brain.AddError("Database Error", "Connection failed", "Increased timeout", []string{"database"}); err != nil {
		t.Fatalf("Failed to add error: %v", err)
	}
	if err := brain.AddPattern("Cache Pattern", "Use Redis for caching", []string{"cache"}); err != nil {
		t.Fatalf("Failed to add pattern: %v", err)
	}
	if err := brain.AddDecision("Use gRPC", "REST to gRPC", "Performance", []string{"grpc"}); err != nil {
		t.Fatalf("Failed to add decision: %v", err)
	}

	// Search for "database"
	results, err := brain.Search("database")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected at least one result for 'database'")
	}

	// Check if result contains the error
	found := false
	for _, result := range results {
		if strings.Contains(result, "Database Error") {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find 'Database Error' in search results")
	}
}

func TestBrain_MultipleEntries(t *testing.T) {
	tmpDir := t.TempDir()
	brain := NewBrain(tmpDir)

	if err := brain.Initialize(); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Add multiple errors
	for i := 0; i < 5; i++ {
		brain.AddError(
			"Error "+string(rune('A'+i)),
			"Description",
			"Solution",
			[]string{"test"},
		)
	}

	content, _ := brain.Read()

	// Check all errors are present
	for i := 0; i < 5; i++ {
		expected := "Error " + string(rune('A'+i))
		if !strings.Contains(content, expected) {
			t.Errorf("Missing entry: %s", expected)
		}
	}
}
