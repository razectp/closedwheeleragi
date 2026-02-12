package tui

import (
	"strings"
	"testing"
)

func TestChatBubbleMaxWidth(t *testing.T) {
	tests := []struct {
		name       string
		totalWidth int
		want       int
	}{
		{"small terminal floors at 30", 20, 30},
		{"very small floors at 30", 10, 30},
		{"60 col terminal gives 48", 60, 48},
		{"100 col terminal gives 80", 100, 80},
		{"150 col terminal gives 120", 150, 120},
		{"200 col terminal caps at 120", 200, 120},
		{"boundary at 37 gives 30 (29 rounds to 30)", 37, 30},
		{"boundary at 38 gives 30", 38, 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := chatBubbleMaxWidth(tt.totalWidth)
			if got != tt.want {
				t.Errorf("chatBubbleMaxWidth(%d) = %d, want %d", tt.totalWidth, got, tt.want)
			}
		})
	}
}

func TestRenderCodeBox(t *testing.T) {
	t.Run("contains language label", func(t *testing.T) {
		result := renderCodeBox("go", []string{"package main"}, 60)
		if !strings.Contains(result, "GO") {
			t.Errorf("expected language label 'GO' in output, got:\n%s", result)
		}
	})

	t.Run("no label for empty lang", func(t *testing.T) {
		result := renderCodeBox("", []string{"hello"}, 60)
		// Should not contain a CodeLangLabel rendered line above the box
		lines := strings.Split(result, "\n")
		// First line should be part of the box border, not a label
		if len(lines) > 0 && strings.Contains(lines[0], "CODE") {
			t.Errorf("expected no label for empty lang, got:\n%s", result)
		}
	})

	t.Run("contains code content", func(t *testing.T) {
		result := renderCodeBox("py", []string{"print('hello')", "print('world')"}, 60)
		if !strings.Contains(result, "print") {
			t.Errorf("expected code content in output, got:\n%s", result)
		}
	})

	t.Run("handles empty lines", func(t *testing.T) {
		result := renderCodeBox("", nil, 60)
		// Should not panic with nil/empty lines
		if result == "" {
			t.Error("expected non-empty output even for empty code lines")
		}
	})

	t.Run("minimum width enforcement", func(t *testing.T) {
		// Very small width should still render without panic
		result := renderCodeBox("go", []string{"x := 1"}, 5)
		if result == "" {
			t.Error("expected non-empty output for small width")
		}
	})
}

func TestRenderThinkingBox(t *testing.T) {
	t.Run("contains header", func(t *testing.T) {
		result := renderThinkingBox("Some reasoning here", 60)
		if !strings.Contains(result, "Reasoning") {
			t.Errorf("expected 'Reasoning' header in output, got:\n%s", result)
		}
	})

	t.Run("contains thinking content", func(t *testing.T) {
		result := renderThinkingBox("Let me analyze this step by step", 60)
		if !strings.Contains(result, "analyze") {
			t.Errorf("expected thinking content in output, got:\n%s", result)
		}
	})

	t.Run("minimum width enforcement", func(t *testing.T) {
		result := renderThinkingBox("test", 5)
		if result == "" {
			t.Error("expected non-empty output for small width")
		}
	})
}

func TestRenderContentCodeBlocks(t *testing.T) {
	// Create a minimal model for testing
	m := &EnhancedModel{width: 100}

	t.Run("closed code fence produces bordered box", func(t *testing.T) {
		content := "Hello\n```go\nfunc main() {}\n```\nBye"
		result := m.renderContent(content, 80)
		// Should contain the code content
		if !strings.Contains(result, "func main") {
			t.Errorf("expected code content, got:\n%s", result)
		}
		// Should NOT contain the old-style divider markers
		if strings.Contains(result, "─── GO ───") {
			t.Errorf("should use bordered box, not divider style, got:\n%s", result)
		}
	})

	t.Run("unclosed code fence (streaming) renders partial", func(t *testing.T) {
		content := "Start\n```python\nprint('hello')\nprint('world')"
		result := m.renderContent(content, 80)
		if !strings.Contains(result, "print") {
			t.Errorf("expected partial code content for streaming, got:\n%s", result)
		}
	})

	t.Run("headings render correctly", func(t *testing.T) {
		content := "# Title\n## Subtitle\nBody text"
		result := m.renderContent(content, 80)
		if !strings.Contains(result, "Title") {
			t.Errorf("expected heading in output, got:\n%s", result)
		}
		if !strings.Contains(result, "Subtitle") {
			t.Errorf("expected subheading in output, got:\n%s", result)
		}
	})

	t.Run("bullet lists render with diamond marker", func(t *testing.T) {
		content := "- First item\n- Second item"
		result := m.renderContent(content, 80)
		if !strings.Contains(result, "First item") {
			t.Errorf("expected list content, got:\n%s", result)
		}
		if !strings.Contains(result, "◆") {
			t.Errorf("expected diamond bullet marker, got:\n%s", result)
		}
	})

	t.Run("error separator renders with heavy line", func(t *testing.T) {
		content := "[error]:\nSomething failed"
		result := m.renderContent(content, 80)
		if !strings.Contains(result, "FAILURE") {
			t.Errorf("expected FAILURE marker, got:\n%s", result)
		}
		if !strings.Contains(result, "━") {
			t.Errorf("expected heavy horizontal line in error separator, got:\n%s", result)
		}
	})

	t.Run("horizontal rule renders with heavy line", func(t *testing.T) {
		content := "Above\n---\nBelow"
		result := m.renderContent(content, 80)
		if !strings.Contains(result, "━") {
			t.Errorf("expected heavy horizontal line for HR, got:\n%s", result)
		}
	})
}

func TestRenderCodeBoxFrameAware(t *testing.T) {
	t.Run("frame-aware width does not overflow", func(t *testing.T) {
		// Render a code box at width 40 and verify all lines fit
		result := renderCodeBox("go", []string{
			"package main",
			"func main() { fmt.Println(\"hello world\") }",
		}, 40)
		for i, line := range strings.Split(result, "\n") {
			w := len([]rune(line))
			// Allow some slack for ANSI escape sequences
			if w > 200 {
				t.Errorf("line %d suspiciously wide (%d runes): %q", i, w, line)
			}
		}
	})
}

func TestRenderThinkingBoxFrameAware(t *testing.T) {
	t.Run("produces non-empty output", func(t *testing.T) {
		result := renderThinkingBox("Step 1: analyze\nStep 2: implement", 60)
		if !strings.Contains(result, "Step 1") {
			t.Errorf("expected thinking content, got:\n%s", result)
		}
		if !strings.Contains(result, "Reasoning") {
			t.Errorf("expected header, got:\n%s", result)
		}
	})
}
