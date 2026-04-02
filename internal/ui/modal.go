package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
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
	ti.Prompt = "" // Remove default "> " prompt
	ti.SetValue(value)
	ti.Focus()
	ti.CharLimit = 200
	ti.Width = 54 // Account for modal padding

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
		// Text input - no extra border, modal border is enough
		content.WriteString(m.Input.View())
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

// Dimmed renders content with faint styling even across nested ANSI resets.
func Dimmed(content string) string {
	if content == "" {
		return ""
	}
	return "\x1b[2m" + strings.ReplaceAll(content, "\x1b[0m", "\x1b[0m\x1b[2m") + "\x1b[0m"
}

// RenderMessageModal renders a centered informational modal body.
func RenderMessageModal(width int, title, body string) string {
	modalWidth := 60
	if maxWidth := width - 4; modalWidth > maxWidth {
		modalWidth = maxWidth
	}
	if modalWidth < 24 {
		modalWidth = 24
	}

	var content strings.Builder
	content.WriteString(lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render(title))
	if body != "" {
		content.WriteString("\n\n")
		content.WriteString(lipgloss.NewStyle().Foreground(ColorWhite).Render(body))
	}

	return OverlayStyle.Width(modalWidth).Render(content.String())
}

// RenderLoadingModal renders a centered loading modal with an activity indicator.
func RenderLoadingModal(width int, title, indicator, body string) string {
	modalWidth := 60
	if maxWidth := width - 4; modalWidth > maxWidth {
		modalWidth = maxWidth
	}
	if modalWidth < 24 {
		modalWidth = 24
	}

	var content strings.Builder
	content.WriteString(lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render(title))
	content.WriteString("\n\n")

	statusLine := lipgloss.JoinHorizontal(
		lipgloss.Center,
		lipgloss.NewStyle().Foreground(ColorAccent).Bold(true).Render(indicator),
		" ",
		lipgloss.NewStyle().Foreground(ColorWhite).Render(body),
	)
	content.WriteString(statusLine)
	content.WriteString("\n\n")
	content.WriteString(lipgloss.NewStyle().Foreground(ColorMuted).Italic(true).Render("The task list will appear as soon as initialization completes."))

	return OverlayStyle.Width(modalWidth).Render(content.String())
}

// RenderOverlay composites a centered overlay on top of the background.
func RenderOverlay(background, overlay string, width, height int) string {
	if width <= 0 || height <= 0 {
		return overlay
	}

	bgLines := normalizeLines(background, width, height)
	ovLines := normalizeLines(overlay, lipgloss.Width(overlay), lipgloss.Height(overlay))
	ovWidth := lipgloss.Width(overlay)
	ovHeight := len(ovLines)
	rendered := make([]string, len(bgLines))
	for i, line := range bgLines {
		rendered[i] = Dimmed(line)
	}
	if ovWidth <= 0 || ovHeight == 0 {
		return strings.Join(rendered, "\n")
	}

	if ovWidth > width {
		ovWidth = width
	}
	if ovHeight > height {
		ovHeight = height
		ovLines = ovLines[:ovHeight]
	}

	x := 0
	if width > ovWidth {
		x = (width - ovWidth) / 2
	}
	y := 0
	if height > ovHeight {
		y = (height - ovHeight) / 2
	}

	for i := 0; i < ovHeight && y+i < len(bgLines); i++ {
		overlayLine := fitLine(ovLines[i], ovWidth)
		backgroundLine := bgLines[y+i]
		left := Dimmed(ansi.Cut(backgroundLine, 0, x))
		right := Dimmed(ansi.Cut(backgroundLine, x+ovWidth, width))
		rendered[y+i] = left + "\x1b[0m" + overlayLine + "\x1b[0m" + right
	}

	return strings.Join(rendered, "\n")
}

func normalizeLines(content string, width, height int) []string {
	lines := strings.Split(content, "\n")
	if height < 0 {
		height = 0
	}
	if len(lines) > height && height > 0 {
		lines = lines[:height]
	}
	if height > 0 {
		for len(lines) < height {
			lines = append(lines, "")
		}
	}
	for i := range lines {
		lines[i] = fitLine(lines[i], width)
	}
	return lines
}

func fitLine(line string, width int) string {
	if width <= 0 {
		return ""
	}
	line = ansi.Truncate(line, width, "")
	if pad := width - ansi.StringWidth(line); pad > 0 {
		line += strings.Repeat(" ", pad)
	}
	return line
}
