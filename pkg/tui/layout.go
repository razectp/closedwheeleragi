package tui

// Layout constants for consistent spacing
const (
	HeaderHeight     = 2 // 1 line + MarginBottom(1)
	StatusBarHeight  = 1
	DividerHeight    = 1
	TextareaHeight   = 5 // 3 inner lines + 2 for border
	HelpBarHeight    = 1
	ProcessingHeight = 2
)

// GetFixedVerticalHeight returns the total height of non-viewport elements.
func (m *EnhancedModel) GetFixedVerticalHeight() int {
	h := HeaderHeight + StatusBarHeight + HelpBarHeight + TextareaHeight + (DividerHeight * 2)

	if len(m.activeTools) > 0 && !m.processing {
		h += 1 // ToolsSectionHeight
	}

	if m.processing {
		h += ProcessingHeight
	}

	return h
}

// RecalculateLayout updates viewport and textarea dimensions.
func (m *EnhancedModel) RecalculateLayout() {
	fixedH := m.GetFixedVerticalHeight()
	viewportHeight := m.height - fixedH
	if viewportHeight < 5 {
		viewportHeight = 5
	}

	vpWidth := m.width - 2
	if vpWidth < 10 {
		vpWidth = 10
	}

	// Calculate YPosition: header + status + optional tools + divider
	toolsH := 0
	if len(m.activeTools) > 0 && !m.processing {
		toolsH = 1
	}
	yPos := HeaderHeight + StatusBarHeight + toolsH + DividerHeight

	if !m.ready {
		m.viewport.Width = vpWidth
		m.viewport.Height = viewportHeight
		m.viewport.YPosition = yPos
		m.ready = true
	} else {
		m.viewport.Width = vpWidth
		m.viewport.Height = viewportHeight
		m.viewport.YPosition = yPos
	}

	m.textarea.SetWidth(m.width - 8)
	m.textarea.SetHeight(3)

	// Update overlay viewports if they are active
	if m.state == StateDebateViewer {
		visibleH := m.debateViewerVisibleHeight()
		boxWidth := m.width - 10
		if boxWidth < 30 {
			boxWidth = 30
		}
		m.debateView.Viewport.Width = boxWidth - 4
		m.debateView.Viewport.Height = visibleH
	}
}
