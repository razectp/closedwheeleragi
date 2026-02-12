// Package tools provides detailed debug capabilities for tool execution
package tools

import (
	"encoding/json"
	"fmt"
	"log"
	"runtime/debug"
	"strings"
	"time"
)

// DebugLevel represents the verbosity of debug output
type DebugLevel int

const (
	DebugOff     DebugLevel = 0
	DebugBasic   DebugLevel = 1
	DebugVerbose DebugLevel = 2
	DebugTrace   DebugLevel = 3
)

// ExecutionTrace tracks detailed information about tool execution
type ExecutionTrace struct {
	ToolName      string
	Arguments     map[string]any
	StartTime     time.Time
	EndTime       time.Time
	Duration      time.Duration
	Success       bool
	Error         error
	ErrorType     string // "validation", "execution", "panic", "timeout"
	ErrorStack    string
	Output        string
	OutputPreview string // First 200 chars
	Metadata      map[string]string
}

// DebugLogger handles debug output for tool execution
type DebugLogger struct {
	Level  DebugLevel
	traces []ExecutionTrace
}

// NewDebugLogger creates a new debug logger
func NewDebugLogger(level DebugLevel) *DebugLogger {
	return &DebugLogger{
		Level:  level,
		traces: make([]ExecutionTrace, 0),
	}
}

// StartTrace begins tracking a tool execution
func (d *DebugLogger) StartTrace(toolName string, args map[string]any) *ExecutionTrace {
	trace := &ExecutionTrace{
		ToolName:  toolName,
		Arguments: args,
		StartTime: time.Now(),
		Metadata:  make(map[string]string),
	}

	if d.Level >= DebugBasic {
		log.Printf("[TOOL START] %s at %s", toolName, trace.StartTime.Format("2006-01-02 15:04:05.000"))

		if d.Level >= DebugVerbose {
			argsJSON, _ := json.MarshalIndent(args, "", "  ")
			log.Printf("[TOOL ARGS] %s: %s", toolName, string(argsJSON))
		}
	}

	return trace
}

// EndTrace completes tracking and logs results
func (d *DebugLogger) EndTrace(trace *ExecutionTrace, result ToolResult, err error) {
	trace.EndTime = time.Now()
	trace.Duration = trace.EndTime.Sub(trace.StartTime)
	trace.Success = result.Success
	trace.Error = err
	trace.Output = result.Output

	// Determine error type
	if err != nil {
		trace.ErrorType = "execution"
		if strings.Contains(err.Error(), "validation") {
			trace.ErrorType = "validation"
		} else if strings.Contains(err.Error(), "timeout") {
			trace.ErrorType = "timeout"
		} else if strings.Contains(err.Error(), "panic") {
			trace.ErrorType = "panic"
		}
	}

	// Create output preview
	if len(trace.Output) > 200 {
		trace.OutputPreview = trace.Output[:200] + "..."
	} else {
		trace.OutputPreview = trace.Output
	}

	// Store trace
	d.traces = append(d.traces, *trace)

	// Log results
	if d.Level >= DebugBasic {
		d.logTraceResult(trace, result)
	}
}

// CaptureError records detailed error information with stack trace
func (d *DebugLogger) CaptureError(trace *ExecutionTrace, err error, errorType string) {
	trace.Error = err
	trace.ErrorType = errorType
	trace.ErrorStack = string(debug.Stack())

	if d.Level >= DebugVerbose {
		log.Printf("[TOOL ERROR] Type: %s | Message: %v", errorType, err)

		if d.Level >= DebugTrace {
			log.Printf("[TOOL STACK] %s", trace.ErrorStack)
		}
	}
}

// AddMetadata adds contextual information to the trace
func (d *DebugLogger) AddMetadata(trace *ExecutionTrace, key, value string) {
	if trace.Metadata == nil {
		trace.Metadata = make(map[string]string)
	}
	trace.Metadata[key] = value

	if d.Level >= DebugTrace {
		log.Printf("[TOOL META] %s = %s", key, value)
	}
}

// logTraceResult logs the execution result to the log file (never stdout/stderr).
func (d *DebugLogger) logTraceResult(trace *ExecutionTrace, result ToolResult) {
	status := "SUCCESS"
	if !trace.Success {
		status = "FAILED"
	}

	log.Printf("[TOOL %s] %s | Duration: %v", status, trace.ToolName, trace.Duration)

	if !trace.Success {
		log.Printf("[TOOL ERROR] %s | Type: %s", trace.ToolName, trace.ErrorType)
		if trace.Error != nil {
			log.Printf("[TOOL ERROR] %s | Error: %v", trace.ToolName, trace.Error)
		}
		if result.Error != "" {
			log.Printf("[TOOL ERROR] %s | Details: %s", trace.ToolName, result.Error)
		}
	}

	if d.Level >= DebugVerbose && len(trace.OutputPreview) > 0 {
		preview := trace.OutputPreview
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		log.Printf("[TOOL OUTPUT] %s | %s", trace.ToolName, strings.ReplaceAll(preview, "\n", " "))
	}

	if d.Level >= DebugVerbose && len(trace.Metadata) > 0 {
		for k, v := range trace.Metadata {
			log.Printf("[TOOL META] %s | %s: %s", trace.ToolName, k, v)
		}
	}

	if d.Level >= DebugTrace && !trace.Success && trace.ErrorStack != "" {
		lines := strings.Split(trace.ErrorStack, "\n")
		limit := 20
		if len(lines) < limit {
			limit = len(lines)
		}
		log.Printf("[TOOL STACK] %s | %s", trace.ToolName, strings.Join(lines[:limit], " | "))
	}
}

// GetRecentTraces returns the last N traces
func (d *DebugLogger) GetRecentTraces(n int) []ExecutionTrace {
	if len(d.traces) < n {
		n = len(d.traces)
	}

	start := len(d.traces) - n
	return d.traces[start:]
}

// GetFailedTraces returns all failed executions
func (d *DebugLogger) GetFailedTraces() []ExecutionTrace {
	failed := make([]ExecutionTrace, 0)
	for _, trace := range d.traces {
		if !trace.Success {
			failed = append(failed, trace)
		}
	}
	return failed
}

// GetTracesByTool returns all traces for a specific tool
func (d *DebugLogger) GetTracesByTool(toolName string) []ExecutionTrace {
	matches := make([]ExecutionTrace, 0)
	for _, trace := range d.traces {
		if trace.ToolName == toolName {
			matches = append(matches, trace)
		}
	}
	return matches
}

// GenerateReport generates a summary report of all traces
func (d *DebugLogger) GenerateReport() string {
	var report strings.Builder

	total := len(d.traces)
	if total == 0 {
		return "No tool executions recorded."
	}

	successful := 0
	failed := 0
	var totalDuration time.Duration

	errorsByType := make(map[string]int)
	toolCounts := make(map[string]int)

	for _, trace := range d.traces {
		if trace.Success {
			successful++
		} else {
			failed++
			errorsByType[trace.ErrorType]++
		}
		totalDuration += trace.Duration
		toolCounts[trace.ToolName]++
	}

	report.WriteString("╔══════════════════════════════════════════════════════════════\n")
	report.WriteString("║ 📊 TOOL EXECUTION REPORT\n")
	report.WriteString("╠══════════════════════════════════════════════════════════════\n")
	report.WriteString(fmt.Sprintf("║ Total Executions: %d\n", total))
	report.WriteString(fmt.Sprintf("║ Successful: %d (%.1f%%)\n", successful, float64(successful)/float64(total)*100))
	report.WriteString(fmt.Sprintf("║ Failed: %d (%.1f%%)\n", failed, float64(failed)/float64(total)*100))
	report.WriteString(fmt.Sprintf("║ Average Duration: %v\n", totalDuration/time.Duration(total)))
	report.WriteString("╠══════════════════════════════════════════════════════════════\n")

	if len(errorsByType) > 0 {
		report.WriteString("║ Errors by Type:\n")
		for errType, count := range errorsByType {
			report.WriteString(fmt.Sprintf("║   - %s: %d\n", errType, count))
		}
		report.WriteString("╠══════════════════════════════════════════════════════════════\n")
	}

	report.WriteString("║ Tool Usage:\n")
	for tool, count := range toolCounts {
		report.WriteString(fmt.Sprintf("║   - %s: %d\n", tool, count))
	}
	report.WriteString("╚══════════════════════════════════════════════════════════════\n")

	return report.String()
}

// Clear clears all stored traces
func (d *DebugLogger) Clear() {
	d.traces = make([]ExecutionTrace, 0)
}

// Global debug logger instance
var GlobalDebugLogger = NewDebugLogger(DebugOff)

// SetGlobalDebugLevel sets the global debug level
func SetGlobalDebugLevel(level DebugLevel) {
	GlobalDebugLogger.Level = level
}
