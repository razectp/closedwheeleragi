package builtin

import (
	"fmt"
	"runtime"
	"time"

	"ClosedWheeler/pkg/tools"
)

// GetSystemInfoTool creates a tool for getting system information
func GetSystemInfoTool() *tools.Tool {
	return &tools.Tool{
		Name:        "get_system_info",
		Description: "Get information about the host system (OS, Arch, CPU, Memory)",
		Parameters: &tools.JSONSchema{
			Type:       "object",
			Properties: map[string]tools.Property{},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			info := "System Information:\n"
			info += fmt.Sprintf("- OS: %s\n", runtime.GOOS)
			info += fmt.Sprintf("- Architecture: %s\n", runtime.GOARCH)
			info += fmt.Sprintf("- CPUs: %d\n", runtime.NumCPU())
			info += fmt.Sprintf("- Go Version: %s\n", runtime.Version())
			info += fmt.Sprintf("- Memory Usage: %d MB\n", m.Alloc/1024/1024)
			info += fmt.Sprintf("- System Time: %s\n", time.Now().Format(time.RFC1123))

			return tools.ToolResult{
				Success: true,
				Output:  info,
				Data: map[string]any{
					"os":     runtime.GOOS,
					"arch":   runtime.GOARCH,
					"cpus":   runtime.NumCPU(),
					"memory": m.Alloc,
				},
			}, nil
		},
	}
}

// RegisterDiagnosticsTools registers diagnostics-related tools
func RegisterDiagnosticsTools(registry *tools.Registry) {
	registry.Register(GetSystemInfoTool())
}
