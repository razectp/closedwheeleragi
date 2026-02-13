package tui

// TUIState represents the current active view or overlay in the TUI.
type TUIState int

const (
	StateMain TUIState = iota
	StateModelPicker
	StateHelpMenu
	StateInfoPanel
	StateSettings
	StateDebateWizard
	StateDebateViewer
)

// SetState updates the TUI state and ensures proper cleanup/initialization.
func (m *EnhancedModel) SetState(s TUIState) {
	m.state = s
}

// GetState returns the current TUI state.
func (m EnhancedModel) GetState() TUIState {
	return m.state
}
