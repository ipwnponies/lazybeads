package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// ModalType defines the type of modal
type ModalType int

const (
	ModalInput ModalType = iota
	ModalSelect
)

// ModalOption represents an option in a select modal
type ModalOption struct {
	Label    string
	Value    string
	Shortcut string // Single key shortcut (e.g., "0", "1", "2")
}

// Modal represents a centered overlay dialog
type Modal struct {
	Type     ModalType
	Title    string
	Subtitle string // e.g., issue ID

	// For input modals
	Input textinput.Model

	// For select modals
	Options  []ModalOption
	Selected int
}

// NewModalInput creates a new text input modal
func NewModalInput(title, subtitle, value string) Modal {
	ti := textinput.New()
	ti.SetValue(value)
	ti.Focus()
	ti.CharLimit = 200
	ti.Width = 50

	return Modal{
		Type:     ModalInput,
		Title:    title,
		Subtitle: subtitle,
		Input:    ti,
	}
}

// NewModalSelect creates a new select modal
func NewModalSelect(title, subtitle string, options []ModalOption, currentValue string) Modal {
	selected := 0
	for i, opt := range options {
		if opt.Value == currentValue {
			selected = i
			break
		}
	}

	return Modal{
		Type:     ModalSelect,
		Title:    title,
		Subtitle: subtitle,
		Options:  options,
		Selected: selected,
	}
}

// MoveUp moves selection up in select modal
func (m *Modal) MoveUp() {
	if m.Type == ModalSelect && m.Selected > 0 {
		m.Selected--
	}
}

// MoveDown moves selection down in select modal
func (m *Modal) MoveDown() {
	if m.Type == ModalSelect && m.Selected < len(m.Options)-1 {
		m.Selected++
	}
}

// SelectByShortcut selects an option by its shortcut key
// Returns true if a shortcut matched
func (m *Modal) SelectByShortcut(key string) bool {
	if m.Type != ModalSelect {
		return false
	}
	for i, opt := range m.Options {
		if opt.Shortcut == key {
			m.Selected = i
			return true
		}
	}
	return false
}

// SelectedValue returns the currently selected value
func (m Modal) SelectedValue() string {
	if m.Type == ModalSelect && m.Selected >= 0 && m.Selected < len(m.Options) {
		return m.Options[m.Selected].Value
	}
	return ""
}

// InputValue returns the input value
func (m Modal) InputValue() string {
	return m.Input.Value()
}

// View renders the modal centered in the given dimensions
func (m Modal) View(width, height int) string {
	var content strings.Builder

	// Modal width - fixed reasonable size
	modalWidth := 60
	if modalWidth > width-4 {
		modalWidth = width - 4
	}

	// Build modal content
	titleStyle := lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(ColorMuted).
		Italic(true)

	// Title line
	titleLine := titleStyle.Render(m.Title)
	if m.Subtitle != "" {
		titleLine += " " + subtitleStyle.Render(m.Subtitle)
	}
	content.WriteString(titleLine)
	content.WriteString("\n\n")

	if m.Type == ModalInput {
		// Text input
		inputStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 1).
			Width(modalWidth - 4)
		content.WriteString(inputStyle.Render(m.Input.View()))
		content.WriteString("\n\n")

		// Help text
		helpStyle := lipgloss.NewStyle().Foreground(ColorMuted)
		content.WriteString(helpStyle.Render("enter: save  esc: cancel"))
	} else {
		// Vertical select options
		for i, opt := range m.Options {
			var optText string
			if opt.Shortcut != "" {
				optText = "[" + opt.Shortcut + "] " + opt.Label
			} else {
				optText = "    " + opt.Label
			}

			if i == m.Selected {
				style := lipgloss.NewStyle().
					Foreground(ColorAccent).
					Bold(true)
				content.WriteString("> " + style.Render(optText))
			} else {
				style := lipgloss.NewStyle().
					Foreground(ColorWhite)
				content.WriteString("  " + style.Render(optText))
			}
			content.WriteString("\n")
		}
		content.WriteString("\n")

		// Help text
		helpStyle := lipgloss.NewStyle().Foreground(ColorMuted)
		content.WriteString(helpStyle.Render("j/k: nav  enter: select  esc: cancel"))
	}

	// Style the modal box
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(1, 2).
		Width(modalWidth)

	modalBox := modalStyle.Render(content.String())

	// Center the modal in the available space
	return lipgloss.Place(
		width, height,
		lipgloss.Center, lipgloss.Center,
		modalBox,
	)
}
